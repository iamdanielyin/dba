package dba

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"math"
	"reflect"
	"strings"
	"sync"
)

type DataModel struct {
	conn   *Connection
	schema *Schema
	gdb    *gorm.DB
}

func (dm *DataModel) Create(value any) error {
	return dm.gdb.Create(value).Error
}

func (dm *DataModel) CreateInBatches(value any, batchSize int) error {
	return dm.gdb.CreateInBatches(value, batchSize).Error
}

func (dm *DataModel) parseEntryList(v map[string]any) []*Entry {
	var entries []*Entry
	for key, val := range v {
		key = strings.TrimSpace(key)
		if key != "" && val != nil {
			entries = append(entries, &Entry{
				Key:   key,
				Op:    entryOpEqual,
				Value: val,
			})
		}
	}
	return entries
}

func (dm *DataModel) parseConditions(operator filterOperator, conditions []any) []*Filter {
	if len(conditions) == 0 {
		return nil
	}
	if operator == "" {
		operator = filterOperatorAnd
	}
	// 支持以下几种传参模式
	// 1、dba.Cond
	// 2、key1=value1、key1=value1&key2=value2
	// 3、struct or struct pointer
	// 4、map
	if len(conditions)%2 == 0 {
		// key1=value1、key1=value1&key2=value2
		var (
			isPair  = true
			entries []*Entry
		)
		for i, item := range conditions {
			if i%2 == 0 {
				key, ok := item.(string)
				if !ok {
					isPair = false
					break
				}
				var op entryOp
				key, op = parseEntryOp(key)
				val := conditions[i+1]
				entries = append(entries, &Entry{
					Key:   key,
					Op:    op,
					Value: val,
				})
			} else {
				continue
			}
		}
		if isPair {
			return []*Filter{
				{
					operator:  operator,
					entryType: entryTypeEntryList,
					entryList: entries,
				},
			}
		}
	}
	var filters []*Filter
	for _, item := range conditions {
		switch v := item.(type) {
		case Cond:
			// Cond
			filters = append(filters, &Filter{
				operator:  operator,
				entryType: entryTypeEntryList,
				entryList: dm.parseEntryList(v),
			})
		case map[string]any:
			// map
			filters = append(filters, &Filter{
				operator:  operator,
				entryType: entryTypeEntryList,
				entryList: dm.parseEntryList(v),
			})
		default:
			reflectValue := reflect.Indirect(reflect.ValueOf(item))
			if reflectValue.Kind() == reflect.Struct {
				var entries []*Entry
				for key, val := range ParseStruct(reflectValue.Addr().Interface()) {
					var op entryOp
					key, op = parseEntryOp(key)
					entries = append(entries, &Entry{
						Key:   key,
						Op:    op,
						Value: val,
					})
				}
				filters = append(filters, &Filter{
					operator:  operator,
					entryType: entryTypeEntryList,
					entryList: entries,
				})
			}
		}
	}
	return filters
}

func (dm *DataModel) Find(conditions ...any) *Result {
	res := &Result{
		dm:       dm,
		orderBys: make(map[string]bool),
		cache:    new(sync.Map),
	}
	if len(conditions) > 0 {
		if filters := dm.parseConditions(filterOperatorAnd, conditions); len(filters) > 0 {
			res.filters = filters
		}
	}
	return res
}

type Result struct {
	dm       *DataModel
	filters  []*Filter
	orderBys map[string]bool
	limit    int
	offset   int
	cache    *sync.Map
}

func (r *Result) And(conditions ...any) *Result {
	filters := r.dm.parseConditions(filterOperatorAnd, conditions)
	if len(filters) > 0 {
		r.filters = append(r.filters, filters...)
	}
	return r
}

func (r *Result) Or(conditions ...any) *Result {
	filters := r.dm.parseConditions(filterOperatorOr, conditions)
	if len(filters) > 0 {
		r.filters = append(r.filters, filters...)
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

func (r *Result) One(dst any) error {
	gdb := r.beforeQuery()
	if err := gdb.First(dst).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (r *Result) All(dst any) error {
	gdb := r.beforeQuery()
	return gdb.Find(dst).Error
}

func (r *Result) Count() (int, error) {
	gdb := r.beforeQuery()
	var count int64
	err := gdb.Count(&count).Error
	return int(count), err
}

func (r *Result) Paginate(pageNum int, pageSize int, dst any) (totalRecords int, totalPages int, err error) {
	r.offset = (pageNum - 1) * pageSize
	r.limit = pageSize
	if err = r.All(dst); err != nil {
		return
	}

	gdb := r.dm.gdb
	var count int64
	if err = gdb.Limit(-1).Offset(-1).Count(&count).Error; err == nil {
		totalRecords = int(count)
		totalPages = int(math.Ceil(float64(count) / float64(pageSize)))
	}
	return
}

func (r *Result) UpdateOne(doc any) error {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	values := r.dm.schema.ParseValue(doc, true)
	ret := gdb.Limit(1).Updates(values)
	return ret.Error
}

func (r *Result) UpdateMany(doc any) (int, error) {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	values := r.dm.schema.ParseValue(doc, true)
	ret := gdb.Updates(values)
	return int(ret.RowsAffected), ret.Error
}

func (r *Result) DeleteOne() error {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	ret := gdb.Limit(1).Delete(nil)
	return ret.Error
}

func (r *Result) DeleteMany() (int, error) {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	ret := gdb.Delete(nil)
	return int(ret.RowsAffected), ret.Error
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
		return &field, field.NativeName
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
					if item.operator == filterOperatorOr {
						s := fmt.Sprintf("(%s)", strings.Join(subSQLs, " OR "))
						sqls = append(sqls, s)
					} else {
						s := fmt.Sprintf("(%s)", strings.Join(subSQLs, " AND "))
						sqls = append(sqls, s)
					}
					if len(subAttrs) > 0 {
						attrs = append(attrs, subAttrs...)
					}
				}
			}
			if sfo == filterOperatorOr {
				s := fmt.Sprintf("(%s)", strings.Join(sqls, " OR "))
				return s, attrs
			} else {
				s := fmt.Sprintf("(%s)", strings.Join(sqls, " AND "))
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

func (r *Result) setLimitAndOffset(gdb *gorm.DB, limit, offset int) *gorm.DB {
	if limit != 0 {
		gdb = gdb.Limit(limit)
	}
	if offset != 0 {
		gdb = gdb.Offset(offset)
	}
	return gdb
}

func (r *Result) beforeQuery() *gorm.DB {
	gdb := r.dm.gdb
	gdb = r.setFilters(gdb, r.filters)
	gdb = r.setOrderBys(gdb, r.orderBys)
	gdb = r.setLimitAndOffset(gdb, r.limit, r.offset)
	return gdb
}
