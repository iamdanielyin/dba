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

type ConflictResolutionStrategy string

const (
	ConflictIgnore         ConflictResolutionStrategy = "IGNORE"
	ConflictUpdatePartial  ConflictResolutionStrategy = "UPDATE_PARTIAL"
	ConflictUpdateComputed ConflictResolutionStrategy = "UPDATE_COMPUTED"
)

type CreateOptions struct {
	BatchSize            int
	SharedTx             bool                       // 用于指定是否所有批次共用一个事务
	ConflictResolution   ConflictResolutionStrategy // 指定冲突处理方式
	UpdateColumns        []string                   // 在部分更新情况下指定要更新的列
	ComputedUpdateValues map[string]any             // 在计算更新情况下指定更新的值
}

func (dm *DataModel) Create(value any, options ...*CreateOptions) error {
	var opts CreateOptions
	if len(options) > 0 && options[0] != nil {
		opts = *options[0]
	}
	if opts.BatchSize == 0 {
		opts.BatchSize = 50
	}

	ruv, err := NewReflectUtils(value)
	if err != nil {
		return err
	}

	columns := dm.schema.ScalarFieldNativeNames()
	nativeFields := dm.schema.NativeFields()
	var allVars [][]any

	var isArray bool
	switch ruv.TypeCategory() {
	case CategoryStruct, CategoryStructPointer, CategoryMapStringAny:
		// 单个值插入
		rowVars := extractRowVars(ruv, value, columns, nativeFields)
		allVars = append(allVars, rowVars)

	case CategoryStructSliceOrArray, CategoryStructPointerSliceOrArray, CategoryMapStringAnyPointerSliceOrArray:
		isArray = true
		// 切片或数组插入
		size, _ := ruv.GetLen()
		allVars = make([][]any, size)
		for i := 0; i < size; i++ {
			elem, e := ruv.GetElement(i)
			if e != nil {
				continue
			}
			rowVars := extractRowVars(ruv, elem, columns, nativeFields)
			allVars[i] = rowVars
		}
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
				if isArray {
					elem, _ := ruv.GetElement(j)
					_ = ruv.SetFieldOrKey(elem, aif.Name, insertID)
				} else {
					_ = ruv.SetFieldOrKey(ruv.Raw(), aif.Name, insertID)
				}
				insertID--
			}
		}
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
func extractRowVars(ruv *ReflectUtils, value any, columns []string, nativeFields map[string]*Field) []any {
	var rowVars []any
	for _, column := range columns {
		field := nativeFields[column]
		v, _ := ruv.GetFieldOrKey(value, field.Name)
		rowVars = append(rowVars, v)
	}
	return rowVars
}

// insertBatchWithTx 使用指定的事务插入一批数据
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

	if opts.ConflictResolution != "" {
		data["ConflictResolution"] = opts.ConflictResolution

		var (
			conflictUpdateColumns []string
			conflictUpdateValues  []any
		)
		switch opts.ConflictResolution {
		case ConflictIgnore:
		case ConflictUpdatePartial:
			for _, column := range opts.UpdateColumns {
				conflictUpdateColumns = append(conflictUpdateColumns, fmt.Sprintf("%s = VALUES(%s)", column, column))
			}
		case ConflictUpdateComputed:
			if len(opts.ComputedUpdateValues) > 0 {
				for key, val := range opts.ComputedUpdateValues {
					conflictUpdateColumns = append(conflictUpdateColumns, fmt.Sprintf("%s = ?", key))
					conflictUpdateValues = append(conflictUpdateValues, val)
				}
			}
		}
		if len(conflictUpdateColumns) > 0 {
			data["ConflictUpdateValues"] = strings.Join(conflictUpdateColumns, ", ")
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

func (dm *DataModel) Find(conditions ...any) *Result {
	res := &Result{
		dm:       dm,
		orderBys: make(map[string]bool),
		cache:    new(sync.Map),
		preload:  make(map[string]*PreloadOptions),
	}
	if len(conditions) > 0 {
		if filters := parseConditions(filterOperatorAnd, conditions); len(filters) > 0 {
			res.filters = filters
		}
	}
	return res
}

type Result struct {
	cache      *sync.Map
	dm         *DataModel
	filters    []*Filter
	orderBys   map[string]bool
	fieldNames []string
	isOmit     bool
	limit      int
	offset     int
	preload    map[string]*PreloadOptions
}

func (r *Result) And(conditions ...any) *Result {
	if f := And(conditions...); f != nil {
		r.filters = append(r.filters, f)
	}
	return r
}

func (r *Result) Or(conditions ...any) *Result {
	if f := Or(conditions...); f != nil {
		r.filters = append(r.filters, f)
	}
	return r
}

func (r *Result) OrderBy(name ...string) *Result {
	return r.orderBy(false, name)
}

func (r *Result) OrderByDESC(name ...string) *Result {
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

func (r *Result) Select(names ...string) *Result {
	r.fieldNames = names
	r.isOmit = false
	return r
}

func (r *Result) Omit(names ...string) *Result {
	r.fieldNames = names
	r.isOmit = true
	return r
}

type PreloadOptions struct {
}

func (r *Result) PreloadBy(name string, options ...*PreloadOptions) *Result {
	s := r.dm.schema
	opts := new(PreloadOptions)
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}
	if f := s.Fields[name]; f.Valid() && f.Relationship != nil {
		r.preload[name] = opts
	}
	return r
}

func (r *Result) Preload(names ...string) *Result {
	for _, name := range names {
		r.PreloadBy(name)
	}
	return r
}

func (r *Result) orderBy(isDesc bool, names []string) *Result {
	for _, name := range names {
		name = strings.TrimSpace(name)
		r.orderBys[name] = isDesc
	}
	return r
}

func parseWhere(schema *Schema, filters []*Filter) (string, []any) {
	nativeFields := schema.NativeFields()

	if len(filters) > 0 {
		var setItem func(filterOperator, []*Filter) (string, []any)
		setItem = func(sfo filterOperator, sfs []*Filter) (string, []any) {
			var (
				sqls  []string
				attrs []any
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
						field := nativeFields[key]
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
								subSQLs = append(subSQLs, fmt.Sprintf("(%s IS NULL)", key))
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
						sqls = append(sqls, s)
					} else {
						s := fmt.Sprintf(format, strings.Join(subSQLs, " AND "))
						sqls = append(sqls, s)
					}
					if len(subAttrs) > 0 {
						attrs = append(attrs, subAttrs...)
					}
				}
			}

			sfoFormat := "%s"
			if len(sqls) > 1 {
				sfoFormat = "(%s)"
			}
			if sfo == filterOperatorOr {
				s := fmt.Sprintf(sfoFormat, strings.Join(sqls, " OR "))
				return s, attrs
			} else {
				s := fmt.Sprintf(sfoFormat, strings.Join(sqls, " AND "))
				return s, attrs
			}
		}
		return setItem(filterOperatorAnd, filters)
	}
	return "", nil
}

func parseOrderBys(schema *Schema, orderBys map[string]bool) (string, []any) {
	nativeFields := schema.NativeFields()

	var clauses []string
	for key, val := range orderBys {
		if f := nativeFields[key]; f.Valid() && f.NativeName != "" {
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
		"Table":    r.dm.schema.NativeName,
		"Where":    whereClause,
		"GroupBys": orderByClause,
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
	if len(r.fieldNames) > 0 {
		if r.isOmit {
			scalarFields := r.dm.schema.ScalarFields()
			var omitFieldsMap = make(map[string]bool)
			for _, n := range r.fieldNames {
				omitFieldsMap[n] = true
			}
			for _, f := range scalarFields {
				if omitFieldsMap[f.Name] {
					continue
				}
				columns = append(columns, f.NativeName)
			}
		} else {
			for _, n := range r.fieldNames {
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

func autoScan(dst any, xdb *sqlx.DB, sql string, attrs []any) error {
	ruv, err := NewReflectUtils(dst)
	if err != nil {
		return err
	}

	switch ruv.TypeCategory() {
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
		sliceValue := reflect.ValueOf(ruv.CreateEmptyCopy())
		for rows.Next() {
			elem, _ := ruv.CreateEmptyElement()
			if err = rows.MapScan(elem.(map[string]any)); err == nil {
				elemValue := reflect.ValueOf(elem)
				sliceValue = reflect.Append(sliceValue, elemValue)
			}
		}
		ruv.Value().Set(sliceValue)
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

	return autoScan(dst, r.dm.xdb, sql, attrs)
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

	return autoScan(dst, r.dm.xdb, sql, attrs)
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
	ruv, err := NewReflectUtils(doc)
	if err != nil {
		return 0, err
	}
	pairs, err := ruv.GetAllFieldsOrKeysAndValues(ruv.Raw())
	if err != nil {
		return 0, err
	}
	fields := r.dm.schema.Fields
	var sets []string
	for k, v := range pairs {
		f := fields[k]
		if f.Valid() && f.NativeName != "" {
			sets = append(sets, fmt.Sprintf("%s = ?", f.NativeName))
		} else {
			sets = append(sets, fmt.Sprintf("%s = ?", k))
		}
		attrs = append(attrs, v)
	}
	if len(sets) > 0 {
		data["Sets"] = strings.Join(sets, ", ")
	}
	var buff bytes.Buffer
	if err := r.dm.updateTemplate.Execute(&buff, data); err != nil {
		return 0, err
	}
	sql := buff.String()

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
	r.preload = make(map[string]*PreloadOptions)
}
