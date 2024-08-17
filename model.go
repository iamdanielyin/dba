package dba

import (
	"bytes"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"strings"
	"sync"
	"text/template"
)

type Action string

const (
	CREATE Action = "CREATE"
	UPDATE Action = "UPDATE"
	DELETE Action = "DELETE"
	ONE    Action = "ONE"
	ALL    Action = "ALL"
	COUNT  Action = "COUNT"
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
					_ = ruv.SetFieldOrKey(ruv.Value(), aif.Name, insertID)
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
		"TableName": dm.schema.NativeName,
		"Columns":   strings.Join(columns, ", "),
		"Rows":      strings.Join(placeholders, ", "),
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
	action       Action
	cache        *sync.Map
	dm           *DataModel
	filters      []*Filter
	orderBys     map[string]bool
	fields       []string
	isOmitFields bool
	limit        int
	offset       int
	preload      map[string]*PreloadOptions
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
	r.fields = names
	r.isOmitFields = false
	return r
}

func (r *Result) Omit(names ...string) *Result {
	r.fields = names
	r.isOmitFields = true
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

func (r *Result) getFieldNativeName(key string) (*Field, string) {
	schema := r.dm.schema
	if field := schema.Fields[key]; field.Valid() {
		return field, field.NativeName
	}
	return nil, ""
}

func (r *Result) setFilters(gdb *gorm.DB, filters []*Filter) *gorm.DB {
	if _, ok := r.cache.Load("SET_FILTERS"); ok {
		return gdb
	}

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
						field, nv := r.getFieldNativeName(key)
						if nv != "" {
							key = nv
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
		query, attrs := setItem(filterOperatorAnd, filters)
		gdb = gdb.Where(query, attrs...)
	}
	r.cache.Store("SET_FILTERS", true)
	return gdb
}

func (r *Result) setOrderBys(gdb *gorm.DB, orderBys map[string]bool) *gorm.DB {
	if _, ok := r.cache.Load("SET_ORDER_BYS"); ok {
		return gdb
	}
	for key, val := range orderBys {
		if _, nv := r.getFieldNativeName(key); nv != "" {
			key = nv
		}
		if val {
			gdb = gdb.Order(fmt.Sprintf("%s DESC", key))
		} else {
			gdb = gdb.Order(fmt.Sprintf("%s", key))
		}
	}
	r.cache.Store("SET_ORDER_BYS", true)
	return gdb
}

func (r *Result) beforeQuery(action Action, dst ...any) *gorm.DB {
	//TODO implement me
	panic("implement me")
}

func (r *Result) One(dst any) error {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(ONE, dst)
	if err := gdb.First(dst).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (r *Result) All(dst any) error {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(ALL, dst)
	return gdb.Find(dst).Error
}

func (r *Result) Count() (int, error) {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(COUNT)
	var count int64
	err := gdb.Count(&count).Error
	return int(count), err
}

func (r *Result) Paginate(pageNum int, pageSize int, dst any) (totalRecords int, totalPages int, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *Result) Update(doc any) (int, error) {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(UPDATE)
	values := r.dm.schema.ParseValue(doc, true)
	ret := gdb.Updates(values)
	return int(ret.RowsAffected), ret.Error
}

func (r *Result) Delete() (int, error) {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(DELETE)
	ret := gdb.Delete(nil)
	return int(ret.RowsAffected), ret.Error
}

func (r *Result) reset() {
	r.filters = nil
	r.orderBys = make(map[string]bool)
	r.limit = 0
	r.offset = 0
	r.cache = new(sync.Map)
	r.preload = make(map[string]*PreloadOptions)
}
