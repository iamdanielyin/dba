package dba

import "strings"

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
		tmp := key
		key = tmp[0:idx]
		op = entryOp(strings.ToUpper(tmp[idx+1:]))
	} else {
		key = rawKey
	}
	return
}
