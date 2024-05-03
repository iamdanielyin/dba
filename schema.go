package dba

import (
	"dario.cat/mergo"
	"database/sql"
	"github.com/guregu/null/v5"
	"github.com/iamdanielyin/structs"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"reflect"
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
	Object  SchemaType = "object"
	Array   SchemaType = "array"
)

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
	Name        string
	NativeName  string
	Description string
	Fields      map[string]Field
}

func (s *Schema) Clone() *Schema {
	copied := new(Schema)
	*copied = *s
	return copied
}

type Field struct {
	Name         string
	NativeName   string
	Type         SchemaType
	ItemType     string
	NativeType   string
	Description  string
	Relationship Relationship
	RelConfig    string
	IsPrimary    bool

	// TODO 默认值配置实现
	//DefaultConfig  string
	// TODO 必填配置实现
	//IsRequired     bool
	//RequiredConfig string
	// TODO 唯一值配置实现
	//IsUnique       bool
	//UniqueConfig   string
	// TODO 枚举值配置实现
	//IsEnum         bool
	//EnumConfig     string
}

type Relationship struct {
	Kind           SchemaRelationship
	SrcSchemaName  string
	SrcSchemaField string
	DstSchemaName  string
	DstSchemaField string

	BrgSchemaName     string
	BrgSchemaSrcField string
	BrgSchemaDstField string
	BrgIsNative       bool
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
	if v, ok := value.(*Schema); ok {
		return v, nil
	}
	reflectValue := reflect.Indirect(reflect.ValueOf(value))
	reflectType := reflectValue.Type()
	if reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	if reflectType.Kind() != reflect.Struct {
		return nil, errors.New("dba: schema must be a struct or a struct pointer")
	}
	parsedKey := reflectType
	if cachedValue, ok := structParsedMap.Load(parsedKey); ok {
		return cachedValue.(*Schema), nil
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
		p := Field{
			Name:       fieldName,
			NativeName: strcase.ToSnake(fieldName),
		}
		for k, v := range ParseTag(field.Tag("dba")) {
			switch k {
			case "name":
				p.NativeName = v
			case "type":
				p.NativeType = v
			case "desc":
				p.Description = v
			case "pk":
				p.IsPrimary = true
			case "rel":
				p.RelConfig = v
				fieldReflectType := reflect.TypeOf(fieldValue)
				if fieldReflectType.Kind() == reflect.Ptr {
					fieldReflectType = fieldReflectType.Elem()
				}

				if fieldReflectType.Kind() == reflect.Array || fieldReflectType.Kind() == reflect.Slice {
					elemType := fieldReflectType.Elem()
					if elemType.Kind() == reflect.Ptr {
						elemType = elemType.Elem()
					}
					fieldZeroValue := reflect.New(elemType).Interface()
					fieldStructs := structs.New(fieldZeroValue)
					p.Relationship.DstSchemaName = fieldStructs.Name()
				} else {
					fieldZeroValue := reflect.New(fieldReflectType).Interface()
					fieldStructs := structs.New(fieldZeroValue)
					p.Relationship.DstSchemaName = fieldStructs.Name()
				}
			}
		}
		if p.Type == "" {
			switch fieldValue.(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, sql.NullInt16, sql.NullInt32, sql.NullInt64, null.Int,
				*int, *int8, *int16, *int32, *int64, *uint, *uint8, *uint16, *uint32, *uint64, *uintptr, *sql.NullInt16, *sql.NullInt32, *sql.NullInt64, *null.Int:
				p.Type = Integer
			case float32, float64, sql.NullFloat64, null.Float, complex64, complex128,
				*float32, *float64, *sql.NullFloat64, *null.Float, *complex64, *complex128:
				p.Type = Float
			case bool, sql.NullBool, null.Bool,
				*bool, *sql.NullBool, *null.Bool:
				p.Type = Boolean
			case string, sql.NullString, null.String,
				*string, *sql.NullString, *null.String:
				p.Type = String
			case time.Time, sql.NullTime, null.Time,
				*time.Time, *sql.NullTime, *null.Time:
				p.Type = Object
			}
			if p.Type == "" {
				switch field.Kind() {
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
		if p.Type == "" {
			continue
		}
		schema.Fields[p.Name] = p
	}
	structParsedMap.Store(parsedKey, &schema)
	return &schema, nil
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
