package dba

import (
	"reflect"
	"strings"
)

type DataModel struct {
	conn   *Connection
	schema *Schema
}

func (dm *DataModel) Create(value any) error {
	return dm.conn.gdb.Create(value).Error
}

func (dm *DataModel) CreateInBatches(value any, batchSize int) error {
	return dm.conn.gdb.CreateInBatches(value, batchSize).Error
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
			var entries []*Entry
			for key, val := range v {
				key = strings.TrimSpace(key)
				if key != "" && val != nil {
					entries = append(entries, &Entry{
						Key:   key,
						Value: val,
					})
				}
			}
			filters = append(filters, &Filter{
				operator:  filterOperatorAnd,
				entryType: entryTypeEntryList,
				entryList: entries,
			})
		case map[string]any:
			// map
			var entries []*Entry
			for key, val := range v {
				key = strings.TrimSpace(key)
				if key != "" && val != nil {
					entries = append(entries, &Entry{
						Key:   key,
						Value: val,
					})
				}
			}
			filters = append(filters, &Filter{
				operator:  filterOperatorAnd,
				entryType: entryTypeEntryList,
				entryList: entries,
			})
		default:
			reflectValue := reflect.Indirect(reflect.ValueOf(item))
			if reflectValue.Kind() == reflect.Struct {
				var entries []*Entry
				for key, val := range ParseStruct(reflectValue.Addr().Interface()) {
					entries = append(entries, &Entry{
						Key:   key,
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
		dm: dm,
	}
	filters := dm.parseConditions(conditions)
	if len(filters) > 0 {
		res.filters = filters
	}
	return res
}

type Result struct {
	dm      *DataModel
	filters []*Filter
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

func (r *Result) One(dst any) error {
	// TODO 单个查询
	return nil
}

func (r *Result) All(dst any) error {
	// TODO 多个查询
	return nil
}
