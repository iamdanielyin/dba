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

type Entry struct {
	Key   string
	Value any
}

type Cond map[string]any
