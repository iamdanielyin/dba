package dba

import (
	"dario.cat/mergo"
	"database/sql"
	"github.com/guregu/null/v5"
	"github.com/iamdanielyin/structs"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"log"
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
	Integer SchemaType = "integer"
	Float   SchemaType = "float"
	Boolean SchemaType = "boolean"
	Time    SchemaType = "time"
	Object  SchemaType = "object"
	Array   SchemaType = "array"
)

var allBasicTypeMap = map[SchemaType]bool{
	String:  true,
	Integer: true,
	Float:   true,
	Boolean: true,
	Time:    true,
	Object:  true,
	Array:   true,
}

type SchemaRelationship string

const (
	RelationshipHasOne  SchemaRelationship = "HAS_ONE"
	RelationshipHasMany SchemaRelationship = "HAS_MANY"
	RelationshipRefOne  SchemaRelationship = "REF_ONE"
	RelationshipRefMany SchemaRelationship = "REF_MANY"
)

type SchemaInterface interface {
	Schema() Schema
}

type Schema struct {
	cache *sync.Map

	Name        string           `json:"name,omitempty"`
	NativeName  string           `json:"native_name,omitempty"`
	Description string           `json:"description,omitempty"`
	Fields      map[string]Field `json:"fields,omitempty"`
}

func (s *Schema) Clone() *Schema {
	copied := new(Schema)
	*copied = *s
	return copied
}

func (s *Schema) Cache() *sync.Map {
	if s.cache == nil {
		s.cache = new(sync.Map)
	}
	return s.cache
}

func (s *Schema) PrimaryKeys() []*Field {
	if v, ok := s.Cache().Load("PRIMARY_KEYS"); ok {
		return v.([]*Field)
	}

	var pks []*Field
	for _, field := range s.Fields {
		if field.IsPrimary {
			pks = append(pks, &field)
		}
	}
	s.Cache().Store("PRIMARY_KEYS", pks)

	return pks
}

func (s *Schema) PrimaryKey() *Field {
	pks := s.PrimaryKeys()
	if len(pks) == 0 {
		return nil
	}
	return pks[0]
}

type Field struct {
	Name            string       `json:"name,omitempty"`
	NativeName      string       `json:"native_name,omitempty"`
	Type            SchemaType   `json:"type,omitempty"`
	ItemType        string       `json:"item_type,omitempty"`
	NativeType      string       `json:"native_type,omitempty"`
	Title           string       `json:"title,omitempty"`
	Description     null.String  `json:"description,omitempty"`
	Relationship    Relationship `json:"relationship,omitempty"`
	RelConfig       string       `json:"rel_config,omitempty"`
	IsPrimary       bool         `json:"is_primary"`
	IsUnsigned      bool         `json:"is_unsigned"`
	IsAutoIncrement bool         `json:"is_auto_increment"`

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

type Relationship struct {
	Kind           SchemaRelationship `json:"kind,omitempty"`
	SrcSchemaName  string             `json:"src_schema_name,omitempty"`
	SrcSchemaField string             `json:"src_schema_field,omitempty"`
	DstSchemaName  string             `json:"dst_schema_name,omitempty"`
	DstSchemaField string             `json:"dst_schema_field,omitempty"`

	BrgSchemaName     string `json:"brg_schema_name,omitempty"`
	BrgSchemaSrcField string `json:"brg_schema_src_field,omitempty"`
	BrgSchemaDstField string `json:"brg_schema_dst_field,omitempty"`
	BrgIsNative       bool   `json:"brg_is_native,omitempty"`
}

func ParseSchemas(values ...any) ([]*Schema, error) {
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
	schema := Schema{
		Name:       structName,
		NativeName: strcase.ToSnake(structName),
		Fields:     make(map[string]Field),
	}
	if si, ok := value.(SchemaInterface); ok {
		d := si.Schema()
		if err := mergo.Merge(&schema, d); err != nil {
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
			log.Println(elemNewValue)
		}

		if field.IsEmbedded() {
			embeddedValue := fieldValue
			if field.Kind() == reflect.Ptr && field.IsZero() {
				embeddedValue = fieldNewValue
			}
			embeddedSchema, err := parseSchema(embeddedValue)
			if err == nil {
				for _, embeddedField := range embeddedSchema.Fields {
					schema.Fields[embeddedField.Name] = embeddedField
				}
			}
			continue
		}
		p := Field{
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
				p.Relationship = Relationship{}
				p.RelConfig = v
				if fieldReflectType.Kind() == reflect.Array || fieldReflectType.Kind() == reflect.Slice {
					p.Relationship.DstSchemaName = elemType.Name()
					p.ItemType = elemType.Name()
				} else {
					p.Relationship.DstSchemaName = fieldReflectType.Name()
				}
			}
		}
		parseFieldType(fieldNewValue, fieldKind, &p)
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
		schema.Fields[p.Name] = p
	}
	structParsedMap.Store(parsedKey, schema)
	return &schema, nil
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

func parseRel(config string, currentSchema *Schema, currentField *Field, allSchemas map[string]*Schema) *Relationship {
	config = strings.TrimSpace(config)

	var (
		kind   SchemaRelationship
		others string
	)
	if i := strings.Index(config, ","); i <= 0 {
		return nil
	} else {
		kind = SchemaRelationship(strings.ToUpper(config[:i]))
		others = config[i+1:]
	}

	rel := Relationship{
		Kind:          kind,
		SrcSchemaName: currentSchema.Name,
	}

	_ = mergo.Merge(&rel, &currentField.Relationship)
	switch kind {
	case RelationshipHasOne,
		RelationshipHasMany,
		RelationshipRefOne:
		// HAS_ONE,ID->UserID
		// HAS_MANY,ID->UserID
		// REF_ONE,OrgID->ID
		split := strings.Split(others, "->")
		if len(split) != 2 {
			return nil
		}
		rel.SrcSchemaField = split[0]
		rel.DstSchemaField = split[1]
	case RelationshipRefMany:
		// 直接对表：REF_MANY,UserDept(ID->UserID,ID->DeptID)
		// 对结构体：REF_MANY,user_role_ref(id->user_id,id->role_id)
		fi := strings.Index(others, "(")
		li := strings.LastIndex(others, ")")
		rel.BrgSchemaName = others[:fi]
		for i, item := range strings.Split(others[fi+1:li], ",") {
			item = strings.TrimSpace(item)
			split := strings.Split(item, "->")
			if len(split) == 2 {
				if i == 0 {
					// src
					rel.SrcSchemaField = split[0]
					rel.BrgSchemaSrcField = split[1]
				} else {
					// dst
					rel.DstSchemaField = split[0]
					rel.BrgSchemaDstField = split[1]
				}
			}
		}
		if _, has := allSchemas[rel.BrgSchemaName]; has {
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
