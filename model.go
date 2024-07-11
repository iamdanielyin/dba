package dba

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
	"sync"
)

type Action string

const (
	CREATE Action = "CREATE"
	BATCH  Action = "BATCH"
	UPDATE Action = "UPDATE"
	DELETE Action = "DELETE"
	ONE    Action = "ONE"
	ALL    Action = "ALL"
	COUNT  Action = "COUNT"
)

type DataModel struct {
	conn   *Connection
	schema *Schema
	gdb    *gorm.DB
}

func (dm *DataModel) Create(value any) error {
	gdb := dm.beforeCreate(CREATE)
	return gdb.Create(value).Error
}

func (dm *DataModel) CreateInBatches(value any, batchSize int) error {
	gdb := dm.beforeCreate(BATCH)
	return gdb.CreateInBatches(value, batchSize).Error
}

func (dm *DataModel) beforeCreate(action Action) *gorm.DB {
	dm.gdb.InstanceSet("DBA_ACTION", action)
	dm.gdb.InstanceSet("DBA_MODEL", dm)
	gdb := dm.omitNotScalarFields().gdb
	gdb = gdb.Table(dm.schema.NativeName).Model(nil)
	return gdb
}

func (dm *DataModel) omitNotScalarFields() *DataModel {
	var fields []string
	for _, field := range dm.schema.Fields {
		if !field.IsScalarType() {
			fields = append(fields, field.Name)
		}
	}
	if len(fields) > 0 {
		dm.gdb = dm.gdb.Omit(fields...)
	}
	return dm
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
	gdb := r.dm.gdb

	// 设置过滤
	gdb = r.setFilters(gdb, r.filters)

	// 设置排序
	gdb = r.setOrderBys(gdb, r.orderBys)

	// 设置offset和limit
	if r.limit != 0 {
		gdb = gdb.Limit(r.limit)
	}
	if r.offset != 0 {
		gdb = gdb.Offset(r.offset)
	}

	// 设置select或omit字段
	if len(r.fields) > 0 {
		if r.isOmitFields {
			gdb = gdb.Omit(r.fields...)
		} else {
			gdb = gdb.Select(r.fields)
		}
	}

	// 设置缓存
	r.action = action
	gdb.InstanceSet("DBA_ACTION", action)
	gdb.InstanceSet("DBA_MODEL", r.dm)
	gdb.InstanceSet("DBA_RESULT", r)
	if len(dst) > 0 && dst[0] != nil {
		gdb.InstanceSet("DBA_DST", dst[0])
	}
	return gdb
}
