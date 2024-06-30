package dba

import (
	"reflect"
	"strings"
)

type Filter struct {
	operator  filterOperator
	entryType entryListType
	entryList any // []*Entry、[]*Filter
}

type entryListType string

const (
	entryTypeFilterList entryListType = "FILTER_LIST"
	entryTypeEntryList  entryListType = "ENTRY_LIST"
)

type filterOperator string

const (
	filterOperatorAnd filterOperator = "AND"
	filterOperatorOr  filterOperator = "OR"
)

type entryOp string

const (
	entryOpEqual              entryOp = "="
	entryOpNotEqual           entryOp = "!="
	entryOpLike               entryOp = "$LIKE"
	entryOpPrefix             entryOp = "$PREFIX"
	entryOpSuffix             entryOp = "$SUFFIX"
	entryOpGreaterThan        entryOp = ">"
	entryOpGreaterThanOrEqual entryOp = ">="
	entryOpLessThan           entryOp = "<"
	entryOpLessThanOrEqual    entryOp = "<="
	entryOpIn                 entryOp = "$IN"
	entryOpNotIn              entryOp = "$NIN"
	entryOpExists             entryOp = "$EXISTS"
)

type Entry struct {
	Key   string
	Op    entryOp
	Value any
}

type Cond map[string]any

func parseEntryOp(rawKey string) (key string, op entryOp) {
	rawKey = strings.TrimSpace(rawKey)
	op = entryOpEqual
	if idx := strings.IndexAny(rawKey, " "); idx > 0 {
		tmp := rawKey
		key = tmp[0:idx]
		op = entryOp(strings.ToUpper(tmp[idx+1:]))
	} else {
		key = rawKey
	}
	return
}

func And(conditions ...any) *Filter {
	filters := parseConditions(filterOperatorAnd, conditions)
	if len(filters) > 0 {
		return &Filter{
			operator:  filterOperatorAnd,
			entryType: entryTypeFilterList,
			entryList: filters,
		}
	}
	return nil
}

func Or(conditions ...any) *Filter {
	filters := parseConditions(filterOperatorOr, conditions)
	if len(filters) > 0 {
		return &Filter{
			operator:  filterOperatorAnd,
			entryType: entryTypeFilterList,
			entryList: filters,
		}
	}
	return nil
}

func parseConditions(operator filterOperator, conditions []any) []*Filter {
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
		if item == nil {
			continue
		}
		switch v := item.(type) {
		case *Filter:
			// *Filter
			filters = append(filters, v)
		case []*Filter:
			// []*Filter
			filters = append(filters, v...)
		case Cond:
			// Cond
			filters = append(filters, &Filter{
				operator:  operator,
				entryType: entryTypeEntryList,
				entryList: parseEntryList(v),
			})
		case map[string]any:
			// map
			filters = append(filters, &Filter{
				operator:  operator,
				entryType: entryTypeEntryList,
				entryList: parseEntryList(v),
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

func parseEntryList(v map[string]any) []*Entry {
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
