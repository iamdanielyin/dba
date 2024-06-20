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
	return dm.gdb.Table(dm.schema.NativeName).Create(value).Error
}

func (dm *DataModel) CreateInBatches(value any, batchSize int) error {
	return dm.gdb.Table(dm.schema.NativeName).CreateInBatches(value, batchSize).Error
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

func (dm *DataModel) parseConditions(conditions []any) []*Filter {
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
				val := conditions[i+1]
				entries = append(entries, &Entry{
					Key:   key,
					Value: val,
				})
			} else {
				continue
			}
		}
		if isPair {
			return []*Filter{
				{
					operator:  filterOperatorAnd,
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
				operator:  filterOperatorAnd,
				entryType: entryTypeEntryList,
				entryList: dm.parseEntryList(v),
			})
		case map[string]any:
			// map
			filters = append(filters, &Filter{
				operator:  filterOperatorAnd,
				entryType: entryTypeEntryList,
				entryList: dm.parseEntryList(v),
			})
		default:
			reflectValue := reflect.Indirect(reflect.ValueOf(item))
			if reflectValue.Kind() == reflect.Struct {
				var entries []*Entry
				for key, val := range ParseStruct(reflectValue.Addr().Interface()) {
					op := entryOpEqual
					if i := strings.IndexAny(key, " "); i > 0 {
						key = key[0:i]
						op = entryOp(key[i+1:])
					}
					entries = append(entries, &Entry{
						Key:   key,
						Op:    op,
						Value: val,
					})
				}
				filters = append(filters, &Filter{
					operator:  filterOperatorAnd,
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
	filters := dm.parseConditions(conditions)
	if len(filters) > 0 {
		res.filters = filters
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
	filters := r.dm.parseConditions(conditions)
	if len(filters) > 0 {
		r.filters = append(r.filters, &Filter{
			operator:  filterOperatorAnd,
			entryType: entryTypeFilterList,
			entryList: filters,
		})
	}
	return r
}

func (r *Result) Or(conditions ...any) *Result {
	filters := r.dm.parseConditions(conditions)
	if len(filters) > 0 {
		r.filters = append(r.filters, &Filter{
			operator:  filterOperatorOr,
			entryType: entryTypeFilterList,
			entryList: filters,
		})
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

func (r *Result) OrderByASC(name ...string) *Result {
	return r.orderBy(false, name)
}

func (r *Result) OrderByDESC(name ...string) *Result {
	return r.orderBy(true, name)
}

func (r *Result) getFieldNativeName(key string) string {
	schema := r.dm.schema
	if field := schema.Fields[key]; field.Valid() {
		return field.NativeName
	}
	return ""
}

func (r *Result) setFilters(gdb *gorm.DB, filters []*Filter) {
	if _, ok := r.cache.Load("SET_FILTERS"); ok {
		return
	}
	if len(filters) > 0 {
		for _, item := range filters {
			switch item.entryType {
			case entryTypeFilterList:
				filterList := item.entryList.([]*Filter)
				r.setFilters(gdb, filterList)
			case entryTypeEntryList:
				entryList := item.entryList.([]*Entry)
				for _, entry := range entryList {
					key := entry.Key
					if nv := r.getFieldNativeName(key); nv != "" {
						key = nv
					}
					switch entry.Op {
					case entryOpEqual:
						gdb.Where(fmt.Sprintf("%s = ?", key), entry.Value)
					case entryOpNotEqual:
						gdb.Where(fmt.Sprintf("%s <> ?", key), entry.Value)
					case entryOpLike:
						gdb.Where(fmt.Sprintf("%s LIKE ?", key), "%"+fmt.Sprintf("%v", entry.Value)+"%")
					case entryOpPrefix:
						gdb.Where(fmt.Sprintf("%s LIKE ?", key), fmt.Sprintf("%v", entry.Value)+"%")
					case entryOpSuffix:
						gdb.Where(fmt.Sprintf("%s LIKE ?", key), "%"+fmt.Sprintf("%v", entry.Value))
					case entryOpGreaterThan:
						gdb.Where(fmt.Sprintf("%s > ?", key), entry.Value)
					case entryOpGreaterThanOrEqual:
						gdb.Where(fmt.Sprintf("%s >= ?", key), entry.Value)
					case entryOpLessThan:
						gdb.Where(fmt.Sprintf("%s < ?", key), entry.Value)
					case entryOpLessThanOrEqual:
						gdb.Where(fmt.Sprintf("%s <= ?", key), entry.Value)
					case entryOpIn:
						gdb.Where(fmt.Sprintf("%s IN ?", key), entry.Value)
					case entryOpNotIn:
						gdb.Where(fmt.Sprintf("%s NOT IN ?", key), entry.Value)
					case entryOpExists:
						var isExists bool
						if v, ok := entry.Value.(bool); ok {
							isExists = v
						} else {
							isExists = true
						}
						if isExists {
							gdb.Where(fmt.Sprintf("%s IS NOT NULL", key))
						} else {
							gdb.Where(fmt.Sprintf("%s IS NULL", key))
						}
					default:
						gdb.Where(fmt.Sprintf("%s = ?", key), entry.Value)
					}
				}
			}
		}
		r.cache.Store("SET_FILTERS", true)
	}
}

func (r *Result) setOrderBys(gdb *gorm.DB, orderBys map[string]bool) {
	if _, ok := r.cache.Load("SET_ORDER_BYS"); ok {
		return
	}
	for key, val := range orderBys {
		if nv := r.getFieldNativeName(key); nv != "" {
			key = nv
		}
		if val {
			gdb.Order(fmt.Sprintf("%s DESC", key))
		} else {
			gdb.Order(fmt.Sprintf("%s", key))
		}
	}
	r.cache.Store("SET_ORDER_BYS", true)
}

func (r *Result) setLimitAndOffset(gdb *gorm.DB, limit, offset int) {
	gdb.Limit(limit).Offset(offset)
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
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	r.setOrderBys(gdb, r.orderBys)
	r.setLimitAndOffset(gdb, r.limit, r.offset)

	if err := gdb.Table(r.dm.schema.NativeName).First(dst).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (r *Result) All(dst any) error {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	r.setOrderBys(gdb, r.orderBys)
	r.setLimitAndOffset(gdb, r.limit, r.offset)

	return gdb.Table(r.dm.schema.NativeName).Find(dst).Error
}

func (r *Result) Count() (int, error) {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	r.setOrderBys(gdb, r.orderBys)
	r.setLimitAndOffset(gdb, r.limit, r.offset)

	var count int64
	err := gdb.Table(r.dm.schema.NativeName).Count(&count).Error
	return int(count), err
}

func (r *Result) FindByPage(pageNum int, pageSize int, dst any) (totalRecords int, totalPages int, err error) {
	r.offset = (pageNum - 1) * pageSize
	r.limit = pageSize
	if err = r.All(dst); err != nil {
		return
	}

	gdb := r.dm.gdb
	var count int64
	if err = gdb.Table(r.dm.schema.NativeName).Limit(-1).Offset(-1).Count(&count).Error; err == nil {
		totalRecords = int(count)
		totalPages = int(math.Ceil(float64(count) / float64(pageSize)))
	}
	return
}

func (r *Result) UpdateOne(doc any) error {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	values := r.dm.schema.ParseValue(doc, true)
	ret := gdb.Table(r.dm.schema.NativeName).Limit(1).Updates(values)
	return ret.Error
}

func (r *Result) UpdateMany(doc any) (int, error) {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	values := r.dm.schema.ParseValue(doc, true)
	ret := gdb.Table(r.dm.schema.NativeName).Updates(values)
	return int(ret.RowsAffected), ret.Error
}

func (r *Result) DeleteOne() error {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	ret := gdb.Table(r.dm.schema.NativeName).Limit(1).Delete(nil)
	return ret.Error
}

func (r *Result) DeleteMany() (int, error) {
	gdb := r.dm.gdb
	r.setFilters(gdb, r.filters)
	ret := gdb.Table(r.dm.schema.NativeName).Delete(nil)
	return int(ret.RowsAffected), ret.Error
}
