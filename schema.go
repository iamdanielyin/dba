package dba

import (
	"dario.cat/mergo"
	"database/sql"
	"github.com/guregu/null/v5"
	"github.com/iamdanielyin/structs"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var structParsedMap sync.Map

type SchemaType string

const (
	String  SchemaType = "string"
	Integer SchemaType = "int"
	Float   SchemaType = "float"
	Boolean SchemaType = "bool"
	Time    SchemaType = "time"
	Object  SchemaType = "object"
	Array   SchemaType = "array"
)

var scalarTypeMap = map[SchemaType]bool{
	String:  true,
	Integer: true,
	Float:   true,
	Boolean: true,
	Time:    true,
	Object:  true,
	Array:   true,
}

type RelationType string

const (
	HasOne         RelationType = "HAS_ONE"         // HasEntity
	HasMany        RelationType = "HAS_MANY"        // HasEntities
	ReferencesOne  RelationType = "REFERENCES_ONE"  // ReferencedEntity
	ReferencesMany RelationType = "REFERENCES_MANY" // ReferencedEntities
)

type SchemaInterface interface {
	Schema() Schema
}

type Schema struct {
	cache      *sync.Map
	structType reflect.Type

	Name        string            `json:"name,omitempty"`
	NativeName  string            `json:"native_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Fields      map[string]*Field `json:"fields,omitempty"`

	CreateClauses string `json:"create_clauses,omitempty"`
	UpdateClauses string `json:"update_clauses,omitempty"`
	DeleteClauses string `json:"delete_clauses,omitempty"`
	QueryClauses  string `json:"query_clauses,omitempty"`
}

func (s *Schema) Clone() *Schema {
	copied := new(Schema)
	*copied = *s
	copied.Fields = make(map[string]*Field)
	for _, f := range s.Fields {
		copied.Fields[f.Name] = f.Clone()
	}
	return copied
}

func (s *Schema) NativeFields() map[string]*Field {
	if v, ok := s.Cache().Load("NATIVE_FIELDS"); ok {
		return v.(map[string]*Field)
	}
	nf := make(map[string]*Field)
	for _, field := range s.Fields {
		nf[field.NativeName] = field
	}
	s.Cache().Store("NATIVE_FIELDS", nf)
	return nf
}

func (s *Schema) ScalarFields() []*Field {
	if v, ok := s.Cache().Load("SCALAR_FIELDS"); ok {
		return v.([]*Field)
	}
	var fields []*Field
	for _, field := range s.Fields {
		if field.IsScalarType() {
			fields = append(fields, field)
		}
	}
	s.Cache().Store("SCALAR_FIELDS", fields)
	return fields
}

func (s *Schema) ScalarFieldNativeNames() []string {
	var names []string
	for _, f := range s.ScalarFields() {
		names = append(names, f.NativeName)
	}
	return names
}

func (s *Schema) ParseValue(data any, useNativeName bool) map[string]any {
	values := make(map[string]any)

	switch reflect.Indirect(reflect.ValueOf(data)).Kind() {
	case reflect.Map:
		values = data.(map[string]any)
	case reflect.Struct:
		values = ParseStruct(data)
	default:
	}

	result := make(map[string]any)
	if useNativeName {
		for key, val := range values {
			if field := s.Fields[key]; field.Valid() {
				result[field.NativeName] = val
			} else {
				result[key] = val
			}
		}
	} else {
		result = values
	}

	return result
}

func (s *Schema) Cache() *sync.Map {
	if s.cache == nil {
		s.cache = new(sync.Map)
	}
	return s.cache
}

func (s *Schema) PrimaryFields() []*Field {
	if v, ok := s.Cache().Load("PRIMARY_KEYS"); ok {
		return v.([]*Field)
	}

	var pks []*Field
	for _, field := range s.Fields {
		if field.IsPrimary {
			pks = append(pks, field)
		}
	}
	s.Cache().Store("PRIMARY_KEYS", pks)

	return pks
}

func (s *Schema) PrimaryField() *Field {
	pks := s.PrimaryFields()
	if len(pks) == 0 {
		return nil
	}
	return pks[0]
}

func (s *Schema) AutoIncrField() *Field {
	for _, f := range s.Fields {
		if f.IsAutoIncrement {
			return f
		}
	}
	return nil
}

func (s *Schema) NativeFieldNames(names []string, scalarTypeOnly bool) []string {
	var result []string
	for _, name := range names {
		if field := s.Fields[name]; field.Valid() {
			if scalarTypeOnly && !field.IsScalarType() {
				continue
			}
			result = append(result, field.NativeName)
		}
	}
	return result
}

type Field struct {
	Name            string      `json:"name,omitempty"`
	NativeName      string      `json:"native_name,omitempty"`
	Type            SchemaType  `json:"type,omitempty"`
	ItemType        string      `json:"item_type,omitempty"`
	NativeType      string      `json:"native_type,omitempty"`
	Title           string      `json:"title,omitempty"`
	Description     null.String `json:"description,omitempty"`
	Relation        *Relation   `json:"relation,omitempty"`
	RelationConfig  string      `json:"relation_config,omitempty"`
	IsPrimary       bool        `json:"is_primary"`
	IsUnsigned      bool        `json:"is_unsigned"`
	IsAutoIncrement bool        `json:"is_auto_increment"`

	// TODO 默认值配置实现
	//DefaultConfig  string
	// TODO 必填配置实现
	IsRequired     bool   `json:"is_required"`
	RequiredConfig string `json:"required_config,omitempty"`
	// TODO 唯一值配置实现
	//IsUnique       bool
	//UniqueConfig   string
	// TODO 枚举值配置实现
	//IsEnum         bool
	//EnumConfig     string
}

func (f *Field) Valid() bool {
	return f != nil && f.Type != ""
}

func (f *Field) IsScalarType() bool {
	if f.Type == Object || f.Type == Array {
		return false
	}
	return scalarTypeMap[f.Type]
}

func (f *Field) Clone() *Field {
	copied := new(Field)
	*copied = *f
	copied.Relation = f.Relation.Clone()
	return copied
}

type Relation struct {
	Type      RelationType `json:"kind,omitempty"`
	SrcSchema string       `json:"src_schema,omitempty"`
	SrcField  string       `json:"src_field,omitempty"`
	DstSchema string       `json:"dst_schema,omitempty"`
	DstField  string       `json:"dst_field,omitempty"`

	BrgSchema   string `json:"brg_schema,omitempty"`
	BrgSrcField string `json:"brg_src_field,omitempty"`
	BrgDstField string `json:"brg_dst_field,omitempty"`
	BrgIsNative bool   `json:"brg_is_native,omitempty"`
}

func (rs *Relation) Valid() bool {
	return rs != nil && rs.Type != ""
}

func (rs *Relation) Clone() *Relation {
	if rs == nil {
		return nil
	}
	copied := new(Relation)
	*copied = *rs
	return copied
}

func ParseSchema(values ...any) ([]*Schema, error) {
	var results []*Schema
	for _, value := range values {
		s, err := parseSchema(value)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func parseSchema(value any) (*Schema, error) {
	if value == nil {
		return nil, errors.New("dba: value is nil")
	}
	if v, ok := value.(*Schema); ok {
		return v, nil
	}
	reflectType := reflect.TypeOf(value)
	if reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	if reflectType.Kind() != reflect.Struct {
		return nil, errors.New("dba: schema must be a struct or a struct pointer")
	}
	parsedKey := reflectType
	if cachedValue, ok := structParsedMap.Load(parsedKey); ok {
		s := cachedValue.(Schema)
		return &s, nil
	}

	s := structs.New(value)
	structName := s.Name()
	sch := Schema{
		structType: reflectType,
		Name:       structName,
		NativeName: strcase.ToSnake(structName),
		Fields:     make(map[string]*Field),
	}
	if si, ok := value.(SchemaInterface); ok {
		d := si.Schema()
		if err := mergo.Merge(&sch, d); err != nil {
			return nil, errors.Wrap(err, "dba: failed to merge schema")
		}
	}
	for _, field := range s.Fields() {
		fieldName := field.Name()
		fieldValue := field.Value()
		fieldKind := field.Kind()
		fieldReflectType := reflect.TypeOf(fieldValue)
		if fieldReflectType.Kind() == reflect.Ptr {
			fieldReflectType = fieldReflectType.Elem()
		}
		fieldNewValue := reflect.New(fieldReflectType).Interface()
		var (
			elemType     reflect.Type
			elemNewValue any
		)
		if fieldReflectType.Kind() == reflect.Array || fieldReflectType.Kind() == reflect.Slice {
			elemType = fieldReflectType.Elem()
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			elemNewValue = reflect.New(elemType).Interface()
		}

		if field.IsEmbedded() {
			embeddedValue := fieldValue
			if field.Kind() == reflect.Ptr && field.IsZero() {
				embeddedValue = fieldNewValue
			}
			embeddedSchema, err := parseSchema(embeddedValue)
			if err == nil {
				for _, embeddedField := range embeddedSchema.Fields {
					sch.Fields[embeddedField.Name] = embeddedField
				}
			}
			continue
		}
		p := &Field{
			Name:        fieldName,
			NativeName:  strcase.ToSnake(fieldName),
			IsRequired:  true,
			Description: null.NewString("", false),
		}
		for k, v := range ParseTag(field.Tag("dba")) {
			switch k {
			case "name":
				p.NativeName = v
			case "type":
				p.NativeType = v
			case "title":
				p.Title = v
			case "null":
				b, err := strconv.ParseBool(v)
				p.IsRequired = !b
				if v != "" && err != nil {
					p.RequiredConfig = v
				}
			case "desc":
				p.Description = null.StringFrom(v)
			case "pk":
				p.IsPrimary = true
			case "incr":
				p.IsAutoIncrement = true
			case "rel":
				p.Relation = new(Relation)
				p.RelationConfig = v
				if fieldReflectType.Kind() == reflect.Array || fieldReflectType.Kind() == reflect.Slice {
					p.Relation.DstSchema = elemType.Name()
					p.ItemType = elemType.Name()
				} else {
					p.Relation.DstSchema = fieldReflectType.Name()
				}
			}
		}
		parseFieldType(fieldNewValue, fieldKind, p)
		if elemType != nil {
			p.Type = Array
			if p.ItemType == "" {
				var ef Field
				parseFieldType(elemNewValue, elemType.Kind(), &ef)
				p.ItemType = string(ef.Type)
			}
		}
		if p.Type == "" {
			continue
		}
		sch.Fields[p.Name] = p
	}
	structParsedMap.Store(parsedKey, sch)
	return &sch, nil
}

func parseFieldType(fieldNewValue any, fieldKind reflect.Kind, p *Field) {
	switch fieldNewValue.(type) {
	case int, int8, int16, int32, int64,
		*int, *int8, *int16, *int32, *int64:
		p.Type = Integer
	case sql.NullInt16, sql.NullInt32, sql.NullInt64, null.Int,
		*sql.NullInt16, *sql.NullInt32, *sql.NullInt64, *null.Int:
		p.Type = Integer
		p.IsRequired = false
	case uint, uint8, uint16, uint32, uint64, uintptr,
		*uint, *uint8, *uint16, *uint32, *uint64, *uintptr:
		p.Type = Integer
		p.IsUnsigned = true
	case float32, float64,
		*float32, *float64:
		p.Type = Float
	case sql.NullFloat64, null.Float,
		*sql.NullFloat64, *null.Float:
		p.Type = Float
		p.IsRequired = false
	case bool, *bool:
		p.Type = Boolean
	case sql.NullBool, null.Bool,
		*sql.NullBool, *null.Bool:
		p.Type = Boolean
		p.IsRequired = false
	case string, *string:
	case sql.NullString, null.String,
		*sql.NullString, *null.String:
		p.Type = String
		p.IsRequired = false
	case time.Time, *time.Time:
		p.Type = Time
	case sql.NullTime, null.Time,
		*sql.NullTime, *null.Time:
		p.Type = Time
		p.IsRequired = false
	}
	if p.Type == "" {
		switch fieldKind {
		case reflect.Bool:
			p.Type = Boolean
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			p.Type = Integer
		case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
			p.Type = Float
		case reflect.Array, reflect.Slice:
			p.Type = Array
		case reflect.Map, reflect.Struct, reflect.Interface, reflect.Pointer:
			p.Type = Object
		case reflect.String:
			p.Type = String
		default:
			p.Type = ""
		}
	}
}

func parseRelation(config string, currentSchema *Schema, currentField *Field, schs map[string]*Schema) *Relation {
	config = strings.TrimSpace(config)

	var (
		typ    RelationType
		others string
	)
	if i := strings.Index(config, ","); i <= 0 {
		return nil
	} else {
		typ = RelationType(strings.ToUpper(config[:i]))
		others = config[i+1:]
	}

	rel := Relation{
		Type:      typ,
		SrcSchema: currentSchema.Name,
	}

	_ = mergo.Merge(&rel, &currentField.Relation)
	switch typ {
	case HasOne,
		HasMany,
		ReferencesOne:
		// HAS_ONE,ID->UserID
		// HAS_MANY,ID->UserID
		// REF_ONE,OrgID->ID
		split := strings.Split(others, "->")
		if len(split) != 2 {
			return nil
		}
		rel.SrcField = split[0]
		rel.DstField = split[1]
	case ReferencesMany:
		// 直接对表：REF_MANY,UserDept(ID->UserID,ID->DeptID)
		// 对结构体：REF_MANY,user_role_ref(id->user_id,id->role_id)
		fi := strings.Index(others, "(")
		li := strings.LastIndex(others, ")")
		rel.BrgSchema = others[:fi]
		for i, item := range strings.Split(others[fi+1:li], ",") {
			item = strings.TrimSpace(item)
			split := strings.Split(item, "->")
			if len(split) == 2 {
				if i == 0 {
					// src
					rel.SrcField = split[0]
					rel.BrgSrcField = split[1]
				} else {
					// dst
					rel.DstField = split[0]
					rel.BrgDstField = split[1]
				}
			}
		}
		if _, has := schs[rel.BrgSchema]; has {
			rel.BrgIsNative = false
		} else {
			rel.BrgIsNative = true
		}
	default:
		return nil
	}

	return &rel
}

// ParseTag "KEY=VALUE;KEY=VALUE;..."
func ParseTag(tagName string) map[string]string {
	tagName = strings.TrimSpace(tagName)

	var result = make(map[string]string)

	for _, item := range strings.Split(tagName, ";") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		var (
			key   string
			value string
		)
		if i := strings.Index(item, "="); i > 0 {
			key = item[:i]
			value = item[i+1:]
		} else {
			key = item
			value = "true"
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		result[key] = value
	}
	return result
}
