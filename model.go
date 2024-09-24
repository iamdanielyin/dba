package dba

import (
	"bytes"
	"fmt"
	"github.com/jmoiron/sqlx"
	"math"
	"reflect"
	"strings"
	"sync"
	"text/template"
)

type DataModel struct {
	conn           *Connection
	schema         *Schema
	xdb            *sqlx.DB
	createTemplate *template.Template
	deleteTemplate *template.Template
	updateTemplate *template.Template
	queryTemplate  *template.Template
}

type ConflictKind string

const (
	ConflictIgnore ConflictKind = "IGNORE"
	ConflictUpdate ConflictKind = "UPDATE"
)

type RelationWriteMode string

const (
	RelatesWriteModeIgnore  RelationWriteMode = "IGNORE"
	RelatesWriteModeAppend  RelationWriteMode = "APPEND"
	RelatesWriteModeUpsert  RelationWriteMode = "UPSERT"
	RelatesWriteModeReplace RelationWriteMode = "REPLACE"
)

type CreateOptions struct {
	BatchSize        int
	SharedTx         bool                   // 用于指定是否所有批次共用一个事务
	ConflictUpdates  *ConflictUpdateOptions // 指定冲突处理方式
	RelatesWriteMode RelationWriteMode
	RelatesWrites    *RelatesWriteOptions
}

type ConflictUpdateOptions struct {
	Kind    ConflictKind
	Columns map[string]any // 在部分更新情况下指定要更新的列（保留原值的话，value为nil即可）
}

// 关系更新逻辑设计
//
// 创建时/更新时
// * HAS_ONE：根据关系字段查询--有档案修改，无档案新增
// * HAS_MANY：根据关系字段查询
//  -1. 忽略模式：无
//   1. 追加模式：无ID创建
//   2. 更新模式：无ID创建，有ID更新关系字段+其他字段（如有）
//   3. 替换模式：无ID创建，有ID更新关系字段+其他字段（如有），删除不在范围内的档案（默认策略）
// * REF_ONE：更新关系字段（忽略引用档案传参）
// * REF_MANY：更新中间表（忽略引用档案传参）
//
// 删除时
// * HAS_ONE：删除档案（主+子）
// * HAS_MANY：删除档案（主+子）
// * REF_ONE：删除关系（置空当前表关系字段）
// * REF_MANY：删除关系（清空中间表）

type RelatesWriteOptions struct {
	IgnoreFields  []string // 无 -1
	AppendFields  []string // 无ID创建 1
	UpsertFields  []string // 无ID创建；有ID更新关系字段+其他字段（如有） 2
	ReplaceFields []string // 无ID创建；有ID更新关系字段+其他字段（如有）；删除不在范围内的档案（默认策略） 3
}

func (dm *DataModel) Create(value any, options ...*CreateOptions) error {
	var opts CreateOptions
	if len(options) > 0 && options[0] != nil {
		opts = *options[0]
	}
	if opts.BatchSize == 0 {
		opts.BatchSize = 50
	}

	ru, err := NewReflectUtils(value)
	if err != nil {
		return err
	}

	columns := dm.schema.ScalarFieldNativeNames()
	nativeFields := dm.schema.NativeFields()
	var allVars [][]any

	if ru.isArray {
		// 切片或数组插入
		size := ru.GetLen()
		allVars = make([][]any, size)
		for i := 0; i < size; i++ {
			elem := ru.GetElement(i)
			rowVars := extractRowVars(ru, elem, columns, nativeFields)
			allVars[i] = rowVars
		}
	} else {
		// 单个值插入
		rowVars := extractRowVars(ru, value, columns, nativeFields)
		allVars = append(allVars, rowVars)
	}

	aif := dm.schema.AutoIncrField()

	var tx *sqlx.Tx
	if opts.SharedTx {
		// 开启一个事务供所有批次使用
		tx, err = dm.xdb.Beginx()
		if err != nil {
			return err
		}
	}

	// 分批插入
	for i := 0; i < len(allVars); i += opts.BatchSize {
		end := i + opts.BatchSize
		if end > len(allVars) {
			end = len(allVars)
		}

		varsBatch := allVars[i:end]

		// 如果不共用事务，每个批次单独开启事务
		if !opts.SharedTx {
			tx, err = dm.xdb.Beginx()
			if err != nil {
				return err
			}
		}

		lastInsertId, err := dm.insertBatchWithTx(tx, columns, varsBatch, &opts)
		if err != nil {
			if tx != nil {
				_ = tx.Rollback()
			}
			return err
		}

		if !opts.SharedTx {
			// 提交每个批次的事务
			if err := tx.Commit(); err != nil {
				return err
			}
		}

		insertID := lastInsertId
		if lastInsertId > 0 && aif != nil {
			// 设置自增字段
			for j := end - 1; j >= i; j-- {
				if ru.isArray {
					elem := ru.GetElement(j)
					_ = ru.SetFieldOrKey(elem, aif.Name, insertID)
				} else {
					_ = ru.SetFieldOrKey(ru.Raw(), aif.Name, insertID)
				}
				insertID--
			}
		}
		dm.afterCreate()
	}

	if opts.SharedTx {
		// 所有批次共用一个事务，提交事务
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// extractRowVars 提取单行数据
func extractRowVars(ru *ReflectUtils, value any, columns []string, nativeFields map[string]*Field) []any {
	var rowVars []any
	for _, column := range columns {
		field := nativeFields[column]
		v, _ := ru.GetElemFieldOrKey(value, field.Name)
		rowVars = append(rowVars, v)
	}
	return rowVars
}

func (dm *DataModel) insertBatchWithTx(tx *sqlx.Tx, columns []string, vars [][]any, opts *CreateOptions) (int64, error) {
	placeholders := make([]string, len(vars))
	for i := 0; i < len(placeholders); i++ {
		rowPlaceholders := make([]string, len(vars[i]))
		for j := range rowPlaceholders {
			rowPlaceholders[j] = "?"
		}
		placeholders[i] = "(" + strings.Join(rowPlaceholders, ",") + ")"
	}

	data := map[string]any{
		"Table":   dm.schema.NativeName,
		"Columns": strings.Join(columns, ", "),
		"Rows":    strings.Join(placeholders, ", "),
	}

	if opts.ConflictUpdates != nil {
		if opts.ConflictUpdates.Kind == "" {
			opts.ConflictUpdates.Kind = ConflictIgnore
		}
		data["ConflictKind"] = opts.ConflictUpdates.Kind

		var (
			conflictUpdateColumns []string
			conflictUpdateValues  []any
		)
		switch opts.ConflictUpdates.Kind {
		case ConflictIgnore:
		case ConflictUpdate:
			for column, customValue := range opts.ConflictUpdates.Columns {
				refVal := reflect.Indirect(reflect.ValueOf(customValue))
				if refVal.IsNil() {
					conflictUpdateColumns = append(conflictUpdateColumns, fmt.Sprintf("%s = VALUES(%s)", column, column))
				} else {
					conflictUpdateColumns = append(conflictUpdateColumns, fmt.Sprintf("%s = ?", column))
					conflictUpdateValues = append(conflictUpdateValues, customValue)
				}
			}
		}
		if len(conflictUpdateColumns) > 0 {
			data["ConflictUpdates"] = strings.Join(conflictUpdateColumns, ", ")
		}
		if len(conflictUpdateValues) > 0 {
			for i, items := range vars {
				items = append(items, conflictUpdateValues...)
				vars[i] = items
			}
		}
	}

	var buff bytes.Buffer
	if err := dm.createTemplate.Execute(&buff, data); err != nil {
		return 0, err
	}
	sql := buff.String()
	sql = formatSQL(sql)

	var args []any
	for _, row := range vars {
		args = append(args, row...)
	}

	r, err := tx.Exec(sql, args...)
	if err != nil {
		return 0, err
	}

	return r.LastInsertId()
}

func (dm *DataModel) afterCreate() {

}

func (dm *DataModel) Find(conditions ...any) *Result {
	res := &Result{
		dm:        dm,
		orderBys:  make(map[string]bool),
		cache:     new(sync.Map),
		populates: make([]*PopulateOptions, 0),
	}
	if len(conditions) > 0 {
		if filters := parseConditions(filterOperatorAnd, conditions); len(filters) > 0 {
			res.filters = filters
		}
	}
	return res
}

type Result struct {
	cache     *sync.Map
	dm        *DataModel
	filters   []*Filter
	orderBys  map[string]bool
	fields    []string
	isOmit    bool
	limit     int
	offset    int
	populates []*PopulateOptions
}

func (r *Result) Where(conditions ...any) *Result {
	return r.AddAnd(conditions...)
}

func (r *Result) AddAnd(conditions ...any) *Result {
	if f := And(conditions...); f != nil {
		r.filters = append(r.filters, f)
	}
	return r
}

func (r *Result) AddOr(conditions ...any) *Result {
	if f := Or(conditions...); f != nil {
		r.filters = append(r.filters, f)
	}
	return r
}

func (r *Result) OrderBy(name ...string) *Result {
	return r.orderBy(false, name)
}

func (r *Result) OrderByDesc(name ...string) *Result {
	return r.orderBy(true, name)
}

func (r *Result) Limit(limit int) *Result {
	r.limit = limit
	return r
}

func (r *Result) Offset(offset int) *Result {
	r.offset = offset
	return r
}

func (r *Result) WithFields(names []string, isOmit bool) *Result {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		r.fields = append(r.fields, name)
	}
	r.fields = names
	r.isOmit = false
	return r
}

func (r *Result) Select(names ...string) *Result {
	return r.WithFields(names, false)
}

func (r *Result) Omit(names ...string) *Result {
	return r.WithFields(names, true)
}

type PopulateOptions struct {
	Path      string
	Match     *Filter
	BrgMatch  *Filter
	Fields    []string
	IsOmit    bool
	OrderBys  map[string]bool
	Limit     int
	Offset    int
	CustomRel *Relation
}

func (r *Result) PopulateBy(options ...*PopulateOptions) *Result {
	for _, option := range options {
		if option == nil {
			continue
		}
		option.Path = strings.TrimSpace(option.Path)
		if option.Path == "" {
			continue
		}
		r.populates = append(r.populates, option)
	}
	return r
}

func (r *Result) Populate(names ...string) *Result {
	for _, name := range names {
		r.PopulateBy(&PopulateOptions{
			Path: name,
		})
	}
	return r
}

func (r *Result) orderBy(isDesc bool, names []string) *Result {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		r.orderBys[name] = isDesc
	}
	return r
}

func parseWhere(sch *Schema, filters []*Filter) (string, []any) {
	if len(filters) > 0 {
		var setItem func(filterOperator, []*Filter) (string, []any)
		setItem = func(sfo filterOperator, sfs []*Filter) (string, []any) {
			var (
				clauses []string
				attrs   []any
			)
			for _, item := range sfs {
				if item == nil {
					continue
				}
				var (
					subSQLs  []string
					subAttrs []any
				)
				switch item.entryType {
				case entryTypeFilterList:
					subFilterList := item.entryList.([]*Filter)
					if s, a := setItem(item.operator, subFilterList); s != "" {
						subSQLs = append(subSQLs, s)
						subAttrs = append(subAttrs, a...)
					}
				case entryTypeEntryList:
					entryList := item.entryList.([]*Entry)
					for _, entry := range entryList {
						key := entry.Key
						field := sch.Fields[key]
						if field.Valid() && field.NativeName != "" {
							key = field.NativeName
						}
						switch entry.Op {
						case entryOpEqual:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s = ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpNotEqual:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s <> ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpLike:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s LIKE ?)", key))
							subAttrs = append(subAttrs, "%"+fmt.Sprintf("%v", entry.Value)+"%")
						case entryOpPrefix:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s LIKE ?)", key))
							subAttrs = append(subAttrs, fmt.Sprintf("%v", entry.Value)+"%")
						case entryOpSuffix:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s LIKE ?)", key))
							subAttrs = append(subAttrs, "%"+fmt.Sprintf("%v", entry.Value))
						case entryOpGreaterThan:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s > ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpGreaterThanOrEqual:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s >= ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpLessThan:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s < ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpLessThanOrEqual:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s <= ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpIn:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s IN ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpNotIn:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s NOT IN ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						case entryOpExists:
							var isExists bool
							if v, ok := entry.Value.(bool); ok {
								isExists = v
							} else {
								isExists = true
							}
							if isExists {
								if field != nil && field.Type == String {
									subSQLs = append(subSQLs, fmt.Sprintf("(%s IS NOT NULL AND %s <> '')", key, key))
								} else {
									subSQLs = append(subSQLs, fmt.Sprintf("(%s IS NOT NULL)", key))
								}
							} else {
								if field != nil && field.Type == String {
									subSQLs = append(subSQLs, fmt.Sprintf("(%s IS NULL OR %s <> '')", key, key))
								} else {
									subSQLs = append(subSQLs, fmt.Sprintf("(%s IS NULL)", key))
								}
							}
						default:
							subSQLs = append(subSQLs, fmt.Sprintf("(%s = ?)", key))
							subAttrs = append(subAttrs, entry.Value)
						}
					}
				}
				if len(subSQLs) > 0 {
					format := "%s"
					if len(subSQLs) > 1 {
						format = "(%s)"
					}
					if item.operator == filterOperatorOr {
						s := fmt.Sprintf(format, strings.Join(subSQLs, " OR "))
						clauses = append(clauses, s)
					} else {
						s := fmt.Sprintf(format, strings.Join(subSQLs, " AND "))
						clauses = append(clauses, s)
					}
					if len(subAttrs) > 0 {
						attrs = append(attrs, subAttrs...)
					}
				}
			}

			sfoFormat := "%s"
			if len(clauses) > 1 {
				sfoFormat = "(%s)"
			}
			if sfo == filterOperatorOr {
				s := fmt.Sprintf(sfoFormat, strings.Join(clauses, " OR "))
				return s, attrs
			} else {
				s := fmt.Sprintf(sfoFormat, strings.Join(clauses, " AND "))
				return s, attrs
			}
		}
		return setItem(filterOperatorAnd, filters)
	}
	return "", nil
}

func parseOrderBys(sch *Schema, orderBys map[string]bool) (string, []any) {
	var clauses []string
	for key, val := range orderBys {
		if f := sch.Fields[key]; f.Valid() && f.NativeName != "" {
			key = f.NativeName
		}
		if val {
			clauses = append(clauses, fmt.Sprintf("%s DESC", key))
		} else {
			clauses = append(clauses, fmt.Sprintf("%s", key))
		}
	}
	if len(clauses) > 0 {
		return strings.Join(clauses, ","), nil
	}
	return "", nil
}

func (r *Result) beforeQuery() (map[string]any, []any) {
	var attrs []any
	// 解析过滤
	whereClause, whereAttrs := parseWhere(r.dm.schema, r.filters)
	if len(whereAttrs) > 0 {
		attrs = append(attrs, whereAttrs...)
	}
	// 解析排序
	orderByClause, orderByAttrs := parseOrderBys(r.dm.schema, r.orderBys)
	if len(orderByAttrs) > 0 {
		attrs = append(attrs, orderByAttrs...)
	}
	data := map[string]any{
		"Table": r.dm.schema.NativeName,
		"Where": whereClause,
	}
	if orderByClause != "" {
		data["GroupBys"] = orderByClause
	}

	// 设置limit
	if r.limit > 0 {
		data["Limit"] = r.limit
	}
	// 设置offset
	if r.offset > 0 {
		data["Offset"] = r.offset
	}
	// 设置select或omit字段
	var columns []string
	if len(r.fields) > 0 {
		if r.isOmit {
			scalarFields := r.dm.schema.ScalarFields()
			var omitFieldsMap = make(map[string]bool)
			for _, n := range r.fields {
				omitFieldsMap[n] = true
			}
			for _, f := range scalarFields {
				if omitFieldsMap[f.Name] {
					continue
				}
				columns = append(columns, f.NativeName)
			}
		} else {
			for _, n := range r.fields {
				f := r.dm.schema.Fields[n]
				if f.Valid() && f.NativeName != "" {
					columns = append(columns, f.NativeName)
				} else {
					columns = append(columns, n)
				}
			}
		}
	} else {
		columns = append(columns, "*")
	}
	if len(columns) > 0 {
		data["Columns"] = strings.Join(columns, ", ")
	}
	return data, attrs
}

func (r *Result) afterQuery(dst any) error {
	sch := r.dm.schema
	docs := dst
	for _, item := range r.populates {
		paths := strings.Split(item.Path, ".")
		for i, p := range paths {
			if sch == nil {
				continue
			}
			// 生成opts
			var opts *PopulateOptions
			if i == len(paths)-1 {
				opts = item
			} else {
				opts = new(PopulateOptions)
			}
			opts.Path = p
			ru, err := NewReflectUtils(docs)
			if err != nil {
				return err
			}
			if i != 0 {
				res, isEmpty := ru.GetFieldOrKey(opts.Path)
				if isEmpty {
					continue
				}
				docs = res
				field := sch.Fields[opts.Path]
				rel := field.Relation
				if opts.CustomRel != nil {
					rel = opts.CustomRel
				}
				switch rel.Kind {
				case ReferencesMany:
					sch = r.dm.conn.ns.SchemaBy(rel.BrgSchema)
				default:
					sch = r.dm.conn.ns.SchemaBy(rel.DstSchema)
				}
				if sch == nil {
					continue
				}
			}
			fru, err := NewReflectUtils(docs)
			if err != nil {
				return err
			}
			if fru.Value().CanAddr() {
				if res, err := relatesQuery(fru.IndirectVal().Addr().Interface(), r.dm.conn, sch, opts); err != nil {
					return err
				} else {
					if err := ru.SetFieldOrKey(ru.Raw(), opts.Path, res); err != nil {
						return err
					}
				}
			} else {
				if _, err := relatesQuery(docs, r.dm.conn, sch, opts); err != nil {
					return err
				}

			}
		}
	}
	return nil
}

func (r *Result) One(dst any) error {
	// FINAL
	defer r.reset()

	data, attrs := r.beforeQuery()
	var buff bytes.Buffer
	if err := r.dm.queryTemplate.Execute(&buff, data); err != nil {
		return err
	}
	sql := buff.String()
	sql = formatSQL(sql)

	if err := autoScan(dst, r.dm.xdb, sql, attrs); err != nil {
		return err
	}

	return r.afterQuery(dst)
}

func (r *Result) All(dst any) error {
	// FINAL
	defer r.reset()

	data, attrs := r.beforeQuery()
	var buff bytes.Buffer
	if err := r.dm.queryTemplate.Execute(&buff, data); err != nil {
		return err
	}
	sql := buff.String()
	sql = formatSQL(sql)

	if err := autoScan(dst, r.dm.xdb, sql, attrs); err != nil {
		return err
	}

	return r.afterQuery(dst)
}

func (r *Result) Count() (int, error) {
	// FINAL
	defer r.reset()

	data, attrs := r.beforeQuery()
	var buff bytes.Buffer
	if err := r.dm.queryTemplate.Execute(&buff, data); err != nil {
		return 0, err
	}
	sql := buff.String()
	sql = formatSQL(sql)

	var count int
	if err := r.dm.xdb.QueryRowx(sql, attrs...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Result) Paginate(pageNum int, pageSize int, dst any) (totalRecords int, totalPages int, err error) {
	// FINAL
	defer r.reset()

	r.offset = (pageNum - 1) * pageSize
	r.limit = pageSize
	if err = r.All(dst); err != nil {
		return
	}

	r.offset = 0
	r.limit = 0
	totalRecords, err = r.Count()
	if err != nil {
		return
	}
	totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
	return
}

func (r *Result) Update(doc any) (int, error) {
	// FINAL
	defer r.reset()

	data, attrs := r.beforeQuery()
	ru, err := NewReflectUtils(doc)
	if err != nil {
		return 0, err
	}
	pairs, err := ru.GetAllFieldsOrKeysAndValues(ru.Raw())
	if err != nil {
		return 0, err
	}
	fields := r.dm.schema.Fields
	var sets []string
	for k, v := range pairs {
		f := fields[k]
		if f.Valid() && f.NativeName != "" {
			if s, isStr := v.(string); isStr && s == SetToNullFlag {
				sets = append(sets, fmt.Sprintf("%s = NULL", f.NativeName))
			} else {
				sets = append(sets, fmt.Sprintf("%s = ?", f.NativeName))
				attrs = append(attrs, v)
			}
		} else {
			if s, isStr := v.(string); isStr && s == SetToNullFlag {
				sets = append(sets, fmt.Sprintf("%s = NULL", k))
			} else {
				sets = append(sets, fmt.Sprintf("%s = ?", k))
				attrs = append(attrs, v)
			}
		}
	}
	if len(sets) > 0 {
		data["Sets"] = strings.Join(sets, ", ")
	}
	var buff bytes.Buffer
	if err := r.dm.updateTemplate.Execute(&buff, data); err != nil {
		return 0, err
	}
	sql := buff.String()
	sql = formatSQL(sql)

	res, err := r.dm.xdb.Exec(sql, attrs...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

func (r *Result) Delete() (int, error) {
	// FINAL
	defer r.reset()

	data, attrs := r.beforeQuery()
	var buff bytes.Buffer
	if err := r.dm.deleteTemplate.Execute(&buff, data); err != nil {
		return 0, err
	}
	sql := buff.String()
	sql = formatSQL(sql)

	res, err := r.dm.xdb.Exec(sql, attrs...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

func (r *Result) reset() {
	r.filters = nil
	r.orderBys = make(map[string]bool)
	r.limit = 0
	r.offset = 0
	r.cache = new(sync.Map)
	r.populates = make([]*PopulateOptions, 0)
}

func autoScan(dst any, xdb *sqlx.DB, sql string, attrs []any) error {
	ru, err := NewReflectUtils(dst)
	if err != nil {
		return err
	}

	switch ru.TypeCategory() {
	case CategoryStruct, CategoryStructPointer:
		return xdb.Get(dst, sql, attrs...)
	case CategoryMapStringAny:
		return xdb.QueryRowx(sql, attrs...).MapScan(dst.(map[string]any))
	case CategoryStructSliceOrArray, CategoryStructPointerSliceOrArray:
		return xdb.Select(dst, sql, attrs...)
	case CategoryMapStringAnyPointerSliceOrArray:
		rows, err := xdb.Queryx(sql, attrs...)
		if err != nil {
			return err
		}
		sliceValue := reflect.ValueOf(ru.CreateEmptyCopy())
		for rows.Next() {
			elem := ru.CreateEmptyElement()
			if err = rows.MapScan(elem.(map[string]any)); err == nil {
				elemValue := reflect.ValueOf(elem)
				sliceValue = reflect.Append(sliceValue, elemValue)
			}
		}
		ru.IndirectVal().Set(sliceValue)
	}

	return nil
}

func relatesQuery(dst any, conn *Connection, sch *Schema, opts *PopulateOptions) (any, error) {
	field := sch.Fields[opts.Path]
	rel := field.Relation
	if opts.CustomRel != nil {
		rel = opts.CustomRel
	}

	if !field.Valid() || rel == nil {
		return dst, fmt.Errorf("populate field failed: %s.%s", sch.Name, opts.Path)
	}
	dst = Item2List(dst)
	ru, err := NewReflectUtils(dst)
	if err != nil {
		return dst, err
	}

	ns := conn.ns

	// 1.收集关联的ID
	var srcValues []any
	for i := 0; i < ru.GetLen(); i++ {
		elem := ru.GetElement(i)
		val, isEmpty := ru.GetElemFieldOrKey(elem, rel.SrcField)
		if !isEmpty {
			continue
		}
		srcValues = append(srcValues, val)
	}
	switch rel.Kind {
	case HasOne, ReferencesOne:
		// 2.统一查询关联数据
		DstModel := ns.Model(rel.DstSchema, &ModelOptions{ConnectionName: conn.name})
		dstSliceRef, err := GetZeroSliceValueOfField(ru.CreateEmptyElement(), opts.Path)
		if err != nil {
			return dst, err
		}
		if err := DstModel.Find(fmt.Sprintf("%s $IN", rel.DstField), srcValues).All(dstSliceRef.Addr().Interface()); err != nil {
			return dst, err
		}
		// 3.建立映射
		dstIdMapRef := make(map[any]*reflect.Value)
		for i := 0; i < dstSliceRef.Len(); i++ {
			re, err := NewReflectUtils(dstSliceRef.Index(i).Interface())
			if err != nil {
				continue
			}
			dstId, isEmpty := re.GetFieldOrKey(rel.DstField)
			if isEmpty {
				continue
			}
			tmp := re.Value()
			dstIdMapRef[dstId] = &tmp
		}
		// 4.回写字段
		for i := 0; i < ru.GetLen(); i++ {
			elem := ru.GetElement(i)
			val, isEmpty := ru.GetFieldOrKey(rel.SrcField)
			if !isEmpty {
				continue
			}
			_ = ru.SetFieldOrKey(elem, opts.Path, dstIdMapRef[val].Interface())
		}
	case HasMany:
		// 1.收集关联的ID
		//（同上）
		// 2.统一查询关联数据
		DstModel := ns.Model(rel.DstSchema, &ModelOptions{ConnectionName: conn.name})
		dstSliceRef, err := GetZeroSliceValueOfField(ru.CreateEmptyElement(), opts.Path)
		if err != nil {
			return dst, err
		}
		if err := DstModel.Find(fmt.Sprintf("%s $IN", rel.DstField), srcValues).All(dstSliceRef.Addr().Interface()); err != nil {
			return dst, err
		}
		// 3.建立映射
		dstIdMapRef := make(map[any]*reflect.Value)
		for i := 0; i < dstSliceRef.Len(); i++ {
			re, err := NewReflectUtils(reflect.Indirect(dstSliceRef.Index(i)).Addr().Interface())
			if err != nil {
				continue
			}
			dstId, isEmpty := re.GetFieldOrKey(rel.DstField)
			if isEmpty {
				continue
			}
			if dstIdMapRef[dstId] == nil {
				valuesRef, _ := GetZeroSliceValueOfField(ru.CreateEmptyElement(), opts.Path)
				dstIdMapRef[dstId] = &valuesRef
			}
			newValuesRef := reflect.Append(*dstIdMapRef[dstId], re.IndirectVal())
			dstIdMapRef[dstId] = &newValuesRef
		}
		// 4.回写字段
		for i := 0; i < ru.GetLen(); i++ {
			elem := ru.GetElement(i)
			val, isEmpty := ru.GetElemFieldOrKey(elem, rel.SrcField)
			if !isEmpty {
				continue
			}
			_ = ru.SetFieldOrKey(elem, opts.Path, dstIdMapRef[val].Interface())
		}

	case ReferencesMany:
		// 1.收集关联的ID
		//（同上）
		if rel.BrgIsNative {
			// 2.统一查询关联数据
			var allBrgData = make([]map[string]any, 0)
			if err := conn.Query(&allBrgData, fmt.Sprintf(`SELECT * FROM %s WHERE %s IN (?)`, rel.BrgSchema, rel.BrgSrcField), srcValues); err != nil {
				return dst, err
			}
			var allDstIds []any
			for _, v := range allBrgData {
				allDstIds = append(allDstIds, v[rel.BrgDstField])
			}
			DstModel := ns.Model(rel.DstSchema, &ModelOptions{ConnectionName: conn.name})
			allDstSliceRef, err := GetZeroSliceValueOfField(ru.CreateEmptyElement(), opts.Path)
			if err != nil {
				return dst, err
			}
			if err := DstModel.Find(fmt.Sprintf("%s $IN", rel.DstField), allDstIds).All(allDstSliceRef.Addr().Interface()); err != nil {
				return dst, err
			}
			allDstIdMapRef := make(map[any]*reflect.Value)
			for i := 0; i < allDstSliceRef.Len(); i++ {
				re, err := NewReflectUtils(allDstSliceRef.Index(i).Interface())
				if err != nil {
					continue
				}
				dstId, isEmpty := re.GetFieldOrKey(rel.DstField)
				if isEmpty {
					continue
				}
				tmp := re.Value()
				allDstIdMapRef[dstId] = &tmp
			}
			// 3.建立映射
			dstIdMapRef := make(map[any]*reflect.Value)
			for _, v := range allBrgData {
				srcIdValue := v[rel.BrgSrcField]
				dstIdValue := v[rel.BrgDstField]
				dstDataRef := allDstIdMapRef[dstIdValue]
				if dstDataRef == nil {
					continue
				}
				if dstIdMapRef[srcIdValue] == nil {
					valuesRef, _ := GetZeroSliceValueOfField(ru.CreateEmptyElement(), opts.Path)
					dstIdMapRef[srcIdValue] = &valuesRef
				}
				tmp := reflect.Append(*dstIdMapRef[srcIdValue], *dstDataRef)
				dstIdMapRef[srcIdValue] = &tmp

			}
			// 4.回写字段
			for i := 0; i < ru.GetLen(); i++ {
				elem := ru.GetElement(i)
				srcId, isEmpty := ru.GetElemFieldOrKey(elem, rel.SrcField)
				if !isEmpty {
					continue
				}
				_ = ru.SetFieldOrKey(elem, opts.Path, dstIdMapRef[srcId].Interface())
			}
		} else {
			// 2.统一查询关联数据
			BrgModel := ns.Model(rel.BrgSchema, &ModelOptions{ConnectionName: conn.name})
			allBrgSliceRef, err := GetZeroSliceValueOfField(ru.CreateEmptyElement(), opts.Path)
			if err != nil {
				return dst, err
			}
			if err := BrgModel.Find(fmt.Sprintf("%s $IN", rel.BrgSrcField), srcValues).All(allBrgSliceRef.Addr().Interface()); err != nil {
				return dst, err
			}
			// 3.建立映射
			srcIdBrgMapRef := make(map[any]*reflect.Value)
			for i := 0; i < allBrgSliceRef.Len(); i++ {
				re, err := NewReflectUtils(allBrgSliceRef.Index(i).Interface())
				if err != nil {
					continue
				}
				brgSrcId, isEmpty := re.GetFieldOrKey(rel.BrgSrcField)
				if isEmpty {
					continue
				}

				if srcIdBrgMapRef[brgSrcId] == nil {
					valuesRef, _ := GetZeroSliceValueOfField(ru.CreateEmptyElement(), opts.Path)
					srcIdBrgMapRef[brgSrcId] = &valuesRef
				}
				tmp := reflect.Append(*srcIdBrgMapRef[brgSrcId], re.Value())
				srcIdBrgMapRef[brgSrcId] = &tmp
			}
			// 4.回写字段
			for i := 0; i < ru.GetLen(); i++ {
				elem := ru.GetElement(i)
				srcId, isEmpty := ru.GetElemFieldOrKey(elem, rel.SrcField)
				if !isEmpty {
					continue
				}
				_ = ru.SetFieldOrKey(elem, opts.Path, srcIdBrgMapRef[srcId].Interface())
			}
		}
	default:
		return dst, fmt.Errorf("unknown relation: %s.%s[%s]", sch.Name, opts.Path, rel.Kind)
	}

	return dst, nil
}

func calcFieldStrategy(sch *Schema, opts *RelatesWriteOptions) map[string]int {
	fieldStrategy := make(map[string]int)
	if opts != nil {
		for _, item := range opts.IgnoreFields {
			if item == "" || fieldStrategy[item] > 0 {
				continue
			}
			fieldStrategy[item] = 1
		}
		for _, item := range opts.ReplaceFields {
			if item == "" || fieldStrategy[item] > 0 {
				continue
			}
			fieldStrategy[item] = 2
		}
		for _, item := range opts.UpsertFields {
			if item == "" || fieldStrategy[item] > 0 {
				continue
			}
			fieldStrategy[item] = 3
		}
		for _, item := range opts.AppendFields {
			if item == "" || fieldStrategy[item] > 0 {
				continue
			}
			fieldStrategy[item] = 4
		}
		if len(opts.ReplaceFields) == 0 {
			// 设置默认策略
			for _, f := range sch.Fields {
				if !f.Valid() || f.Relation == nil {
					continue
				}
				if fieldStrategy[f.Name] != 0 {
					continue
				}
				fieldStrategy[f.Name] = 4
			}
		}
	}
	return fieldStrategy
}

type updatePair struct {
	filter *Filter
	values any
}

func relatesWriteOnCreate(src any, SrcModel *DataModel, conn *Connection, sch *Schema, opts *RelatesWriteOptions) error {
	srcList := NewReflectValue(Item2List(src))
	fieldStrategy := calcFieldStrategy(sch, opts)
	ns := conn.ns

	for srcIdx := 0; srcIdx < srcList.Len(); srcIdx++ {
		srcItem := NewReflectValue(srcList.Index(srcIdx))
		srcItemValues := srcItem.Map()
		if len(srcItemValues) == 0 {
			continue
		}
		for name, fieldValue := range srcItemValues {
			field := sch.Fields[name]
			if !field.Valid() || !field.Relation.Valid() || fieldStrategy[name] <= 0 {
				continue
			}
			rel := field.Relation
			srcId, has := srcItemValues[rel.SrcField]
			if !has {
				continue
			}
			dstSch := ns.SchemaBy(rel.DstSchema)
			if dstSch == nil {
				continue
			}

			DstModel := ns.Model(rel.DstSchema, &ModelOptions{ConnectionName: conn.name})
			switch rel.Kind {
			case HasOne:
				storedDoc := NewReflectValue(NewVar(fieldValue))
				if err := DstModel.Find(fmt.Sprintf("%s", rel.DstField), srcId).One(storedDoc.Addr().Interface()); err != nil {
					return err
				}
				storedDocValues := storedDoc.Map()

				inputDocValues := NewReflectValue(fieldValue).Map()
				inputDocValues[rel.DstField] = srcId

				inputId, hasInputId := inputDocValues[dstSch.PrimaryField().Name]
				storedId, hasStoredId := storedDocValues[dstSch.PrimaryField().Name]

				if hasInputId && !IsNilOrZero(inputId) {
					// 传入有ID
					filter := make(map[string]any)
					values := make(map[string]any)
					for k, v := range inputDocValues {
						f := dstSch.Fields[k]
						if f.IsPrimary {
							filter[k] = v
						} else {
							values[k] = v
						}
					}
					if fieldStrategy[name] >= 2 {
						// update
						if len(filter) > 0 && len(values) > 0 {
							if _, err := DstModel.Find(filter).Update(values); err != nil {
								return err
							}
						}
					}
				} else {
					// create
					if err := DstModel.Create(inputDocValues); err != nil {
						return err
					}
				}
				if hasStoredId && !IsNilOrZero(storedId) && !reflect.DeepEqual(storedId, inputId) {
					// 不相同做置空关系字段
					filter := make(map[string]any)
					for _, f := range dstSch.PrimaryFields() {
						if v := storedDocValues[f.Name]; !IsNilOrZero(v) {
							filter[f.Name] = v
						}
					}
					if len(filter) > 0 {
						if fieldStrategy[name] >= 3 {
							if _, err := DstModel.Find(filter).Delete(); err != nil {
								return err
							}
						} else {
							values := map[string]any{
								rel.DstField: SetToNullFlag,
							}
							if _, err := DstModel.Find(filter).Update(values); err != nil {
								return err
							}
						}
					}
				}
			case HasMany:
				storedDocs := NewReflectValue(NewVar(fieldValue))
				if err := DstModel.Find(fmt.Sprintf("%s", rel.DstField), srcId).All(storedDocs.Addr().Interface()); err != nil {
					return err
				}
				inputDocs := NewReflectValue(fieldValue)

				var createDocs []any
				var updateDocs []*updatePair

				var existItems []map[string]any
				for i := 0; i < inputDocs.Len(); i++ {
					inputDoc := NewReflectValue(inputDocs.Index(i))
					inputDocValues := inputDoc.Map()
					inputDocValues[rel.DstField] = srcId

					filter := make(map[string]any)
					for _, pk := range dstSch.PrimaryFields() {
						v := inputDocValues[pk.Name]
						if !IsNilOrZero(v) {
							filter[pk.Name] = v
						}
						delete(inputDocValues, pk.Name)
					}
					if len(filter) > 0 {
						// 有主键，走update
						updateDocs = append(updateDocs, &updatePair{
							filter: And(filter),
							values: inputDocValues,
						})
						existItems = append(existItems, filter)
					} else {
						// 无主键，走create
						createDocs = append(createDocs, inputDocValues)
					}
				}

				var deleteFilters []any
				for j := 0; j < storedDocs.Len(); j++ {
					storedDoc := NewReflectValue(storedDocs.Index(j))
					storedDocValues := storedDoc.Map()
					for _, ids := range existItems {
						exists := true
						for k, inputID := range ids {
							storedID := storedDocValues[k]
							if !reflect.DeepEqual(inputID, storedID) {
								exists = false
							}
						}
						if !exists {
							deleteFilters = append(deleteFilters, And(ids))
						}
					}
				}

				if fieldStrategy[name] >= 1 {
					// APPEND
					if len(createDocs) > 0 {
						if err := DstModel.Create(createDocs); err != nil {
							return err
						}
					}
				}

				if fieldStrategy[name] >= 2 {
					// UPSERT
					if len(updateDocs) > 0 {
						for _, item := range updateDocs {
							if _, err := DstModel.Find(item.filter).Update(item.values); err != nil {
								return err
							}
						}
					}
				}

				if fieldStrategy[name] >= 3 {
					// REPLACE
					if len(deleteFilters) > 0 {
						if _, err := DstModel.Find(Or(deleteFilters...)).Delete(); err != nil {
							return err
						}
					}
				}
			case ReferencesOne:
				inputDocValues := NewReflectValue(fieldValue).Map()
				if dstID, ok := inputDocValues[rel.DstField]; ok && dstID != nil {
					if _, err := SrcModel.Find(rel.SrcField, dstID).Update(map[string]any{
						rel.SrcField: dstID,
					}); err != nil {
						return err
					}
				}
			case ReferencesMany:
				if rel.BrgIsNative {
					var storeBrgData = make([]map[string]any, 0)
					if err := conn.Query(&storeBrgData, fmt.Sprintf(`SELECT * FROM %s WHERE %s = ?`, rel.BrgSchema, rel.BrgSrcField), srcId); err != nil {
						return err
					}

					var createDocs []any
					var deleteFilters []*Filter

					doneInputIndex := make(map[int]bool)
					for _, brgData := range storeBrgData {
						dstId, ok := brgData[rel.BrgDstField]
						if !ok || dstId == nil {
							deleteFilters = append(deleteFilters, And(rel.BrgSrcField, srcId, rel.BrgDstField+" $EXISTS", false))
							continue
						}
						for k := 0; k < relatesDoc.Len(); k++ {
							if doneInputIndex[k] {
								continue
							}
							inputDoc := NewReflectValue(relatesDoc.Index(k))
							inputID := inputDoc.FieldByName(dstSch.PrimaryField().Name)
							if inputID.IsZero() || reflect.DeepEqual(dstId, inputID.Interface()) {
								continue
							}
							if !reflect.DeepEqual(dstId, inputID.Interface()) {
								createDocs = append(createDocs, inputDoc.Interface())
							} else {
								deleteDocIDs = append(deleteDocIDs, storeID.Interface())
							}
						}
					}
					for j := 0; j < dst.Len(); j++ {
						storeDoc := NewReflectValue(dst.Index(j))
						storeID := storeDoc.FieldByName(dstSch.PrimaryField().Name)
						for k := 0; k < relatesDoc.Len(); k++ {
							if doneInputIndex[k] {
								continue
							}
							inputDoc := NewReflectValue(relatesDoc.Index(k))
							inputID := inputDoc.FieldByName(dstSch.PrimaryField().Name)
							if inputID.IsZero() {
								createDocs = append(createDocs, inputDoc.Interface())
							} else if reflect.DeepEqual(storeID.Interface(), inputID.Interface()) {
								if !storeID.IsZero() {
									updateDocs[storeID.Interface()] = inputDoc
								}
							} else {
								deleteDocIDs = append(deleteDocIDs, storeID.Interface())
							}
							doneInputIndex[k] = true
						}
					}
				} else {

				}
			}
		}
	}
	return nil
}
