package dba

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
	entryOpLike               entryOp = "LIKE"
	entryOpPrefix             entryOp = "PREF"
	entryOpSuffix             entryOp = "SUFF"
	entryOpGreaterThan        entryOp = ">"
	entryOpGreaterThanOrEqual entryOp = ">="
	entryOpLessThan           entryOp = "<"
	entryOpLessThanOrEqual    entryOp = "<="
	entryOpIn                 entryOp = "IN"
	entryOpNotIn              entryOp = "NIN"
	entryOpExists             entryOp = "EXISTS"
)

type Entry struct {
	Key   string
	Op    entryOp
	Value any
}

type Cond map[string]any
