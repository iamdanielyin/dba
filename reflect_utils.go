package dba

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/iamdanielyin/structs"
	"github.com/jinzhu/now"
	"go/ast"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// TypeCategory 表示变量类型的类别
type TypeCategory string

const (
	CategoryStruct                          TypeCategory = "Struct"
	CategoryStructPointer                   TypeCategory = "StructPointer"
	CategoryMapStringAny                    TypeCategory = "MapStringAny"
	CategoryStructSliceOrArray              TypeCategory = "StructSliceOrArray"
	CategoryStructPointerSliceOrArray       TypeCategory = "StructPointerSliceOrArray"
	CategoryMapStringAnyPointerSliceOrArray TypeCategory = "MapStringAnyPointerSliceOrArray"
	CategoryUnknown                         TypeCategory = "Unknown"
)

type ReflectUtils struct {
	raw          any
	rawVal       reflect.Value
	rawTyp       reflect.Type
	indirectVal  reflect.Value
	indirectTyp  reflect.Type
	indirectKind reflect.Kind
	isArray      bool
	cat          TypeCategory
	cachedValues sync.Map // 使用 sync.Map 替代普通 map
}

// NewReflectUtils 创建一个新的 ReflectUtils 对象
func NewReflectUtils(a any) (*ReflectUtils, error) {
	rawVal := reflect.ValueOf(a)
	indirectVal := reflect.Indirect(rawVal)

	ru := ReflectUtils{
		raw:          a,
		rawVal:       rawVal,
		rawTyp:       rawVal.Type(),
		indirectVal:  indirectVal,
		indirectTyp:  indirectVal.Type(),
		indirectKind: indirectVal.Kind(),
	}
	var cat TypeCategory
	switch ru.indirectKind {
	case reflect.Struct:
		cat = CategoryStruct
	case reflect.Map:
		if ru.indirectTyp.Key().Kind() == reflect.String && ru.indirectTyp.Elem().Kind() == reflect.Interface {
			cat = CategoryMapStringAny
		}
	case reflect.Slice, reflect.Array:
		elemType := ru.indirectTyp.Elem()
		switch elemType.Kind() {
		case reflect.Struct:
			cat = CategoryStructSliceOrArray
			ru.isArray = true
		case reflect.Ptr:
			if elemType.Elem().Kind() == reflect.Struct {
				cat = CategoryStructPointerSliceOrArray
				ru.isArray = true
			}
		case reflect.Map:
			if elemType.Key().Kind() == reflect.String && elemType.Elem().Kind() == reflect.Interface {
				cat = CategoryMapStringAnyPointerSliceOrArray
				ru.isArray = true
			}
		default:
			cat = CategoryUnknown
		}
	default:
		cat = CategoryUnknown
	}

	ru.cat = cat
	return &ru, nil
}

// Raw 返回原始值
func (ru *ReflectUtils) Raw() any {
	return ru.raw
}

// Value 返回反射值
func (ru *ReflectUtils) Value() reflect.Value {
	return ru.rawVal
}

// IndirectVal 返回反射值
func (ru *ReflectUtils) IndirectVal() reflect.Value {
	return ru.indirectVal
}

// TypeCategory 返回变量的类型类别
func (ru *ReflectUtils) TypeCategory() TypeCategory {
	return ru.cat
}

// CreateEmptyElement 返回切片或数组元素的空值对象
func (ru *ReflectUtils) CreateEmptyElement() any {
	if ru.isArray {
		elemType := ru.indirectTyp.Elem()

		// 根据元素类型创建相应的空值对象
		switch elemType.Kind() {
		case reflect.Struct:
			return reflect.New(elemType).Elem().Interface()
		case reflect.Ptr:
			if elemType.Elem().Kind() == reflect.Struct {
				return reflect.New(elemType.Elem()).Interface()
			}
		case reflect.Map:
			if elemType.Key().Kind() == reflect.String && elemType.Elem().Kind() == reflect.Interface {
				return reflect.MakeMap(elemType).Interface()
			}
		default:
			return reflect.Zero(elemType).Interface()
		}
	}

	return nil
}

// CreateEmptyCopy 创建变量a的空副本
func (ru *ReflectUtils) CreateEmptyCopy() any {
	// 如果是指针类型，使用 reflect.New 创建一个相同类型的新指针
	if ru.rawTyp.Kind() == reflect.Ptr {
		// 使用 reflect.New 创建一个指向该类型的新指针
		newPtr := reflect.New(ru.rawTyp.Elem())
		return newPtr.Interface()
	}

	// 如果不是指针类型，直接创建一个相同类型的新值
	newVal := reflect.New(ru.rawTyp).Elem()
	return newVal.Interface()
}

// GetLen 获取数组或切片长度
func (ru *ReflectUtils) GetLen() int {
	if ru.isArray {
		return ru.indirectVal.Len()
	}
	return 0
}

// GetElement 获取指定下标的元素
func (ru *ReflectUtils) GetElement(index int) any {
	if ru.isArray {
		if index < 0 || index >= ru.indirectVal.Len() {
			return nil
		}
		return ru.indirectVal.Index(index).Interface()
	}
	return nil
}

// getStructField 获取指定字段的 reflect.StructField
func getStructField(v reflect.Value, fieldName string) (reflect.StructField, error) {
	v = reflect.Indirect(v) // 解引用指针，以获取实际值而非指针本身
	for v.Kind() == reflect.Struct {
		modelType := v.Type()
		for i := 0; i < modelType.NumField(); i++ {
			if fieldStruct := modelType.Field(i); ast.IsExported(fieldStruct.Name) && fieldStruct.Name == fieldName {
				return fieldStruct, nil
			}
		}
	}
	return reflect.StructField{}, fmt.Errorf("字段%s无法获取值", fieldName)
}

// GetFieldOrKey 获取指定结构体字段或 map 键的值。
// 如果是结构体，将尝试获取字段的 reflect.Value 并返回其 Interface()。
// 如果是 map，将尝试获取指定键的值。
// 如果字段或键不存在，或类型不支持，返回错误。
func (ru *ReflectUtils) GetFieldOrKey(elem any, k string) (result any, isEmpty bool) {
	isEmpty = true

	value := reflect.ValueOf(elem)

	switch value.Kind() {
	case reflect.Struct, reflect.Ptr:
		// 获取目标字段的 reflect.Value
		structField, err := getStructField(value, k)
		if err != nil {
			return
		}
		fieldIndex := structField.Index[0]
		if len(structField.Index) == 1 && fieldIndex > 0 {
			fieldValue := reflect.Indirect(value).Field(fieldIndex)
			result, isEmpty = fieldValue.Interface(), fieldValue.IsZero()
		} else {
			v := reflect.Indirect(value)
			for _, fieldIdx := range structField.Index {
				if fieldIdx >= 0 {
					v = v.Field(fieldIdx)
				} else {
					v = v.Field(-fieldIdx - 1)

					if !v.IsNil() {
						v = v.Elem()
					} else {
						return nil, isEmpty
					}
				}
			}

			result, isEmpty = v.Interface(), v.IsZero()
		}
	case reflect.Map:
		// 处理 map 类型，通过键名获取对应的值
		keyVal := value.MapIndex(reflect.ValueOf(k))
		if !keyVal.IsValid() {
			return
		}
		result, isEmpty = keyVal.Interface(), keyVal.IsZero()
	default:
	}

	return
}

// SetFieldOrKey 设置指定结构体字段或 map 键的值。
// 对于结构体类型，尝试获取字段的 reflect.Value 并设置新值。
// 对于 map 类型，通过键名设置对应的值。
// 如果字段不可设置或类型不支持，返回错误。
func (ru *ReflectUtils) SetFieldOrKey(elem any, k string, v any) (err error) {
	value := reflect.ValueOf(elem)

	switch value.Kind() {
	case reflect.Struct, reflect.Ptr:
		// 获取目标字段的 reflect.Value
		structField, e := getStructField(value, k)
		if e != nil {
			return e
		}
		ctx := context.Background()
		// Set
		switch structField.Type.Kind() {
		case reflect.Bool:
			switch data := v.(type) {
			case **bool:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetBool(**data)
				}
			case bool:
				ReflectValueOf(ctx, structField, value).SetBool(data)
			case int64:
				ReflectValueOf(ctx, structField, value).SetBool(data > 0)
			case string:
				b, _ := strconv.ParseBool(data)
				ReflectValueOf(ctx, structField, value).SetBool(b)
			default:
				err = fallbackSetter(ctx, structField, value, v)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			switch data := v.(type) {
			case **int64:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetInt(**data)
				}
			case **int:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetInt(int64(**data))
				}
			case **int8:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetInt(int64(**data))
				}
			case **int16:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetInt(int64(**data))
				}
			case **int32:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetInt(int64(**data))
				}
			case int64:
				ReflectValueOf(ctx, structField, value).SetInt(data)
			case int:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case int8:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case int16:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case int32:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case uint:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case uint8:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case uint16:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case uint32:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case uint64:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case float32:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case float64:
				ReflectValueOf(ctx, structField, value).SetInt(int64(data))
			case []byte:
				// field.Set(ctx, value, string(data))
			case string:
				if i, err := strconv.ParseInt(data, 0, 64); err == nil {
					ReflectValueOf(ctx, structField, value).SetInt(i)
				}
			case time.Time:
				ReflectValueOf(ctx, structField, value).SetInt(data.UnixNano())
			case *time.Time:
				if data != nil {
					ReflectValueOf(ctx, structField, value).SetInt(data.UnixNano())
				} else {
					ReflectValueOf(ctx, structField, value).SetInt(0)
				}
			default:
				err = fallbackSetter(ctx, structField, value, v)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			switch data := v.(type) {
			case **uint64:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetUint(**data)
				}
			case **uint:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetUint(uint64(**data))
				}
			case **uint8:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetUint(uint64(**data))
				}
			case **uint16:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetUint(uint64(**data))
				}
			case **uint32:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetUint(uint64(**data))
				}
			case uint64:
				ReflectValueOf(ctx, structField, value).SetUint(data)
			case uint:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case uint8:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case uint16:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case uint32:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case int64:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case int:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case int8:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case int16:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case int32:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case float32:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case float64:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data))
			case []byte:
				// field.Set(ctx, value, string(data))
			case time.Time:
				ReflectValueOf(ctx, structField, value).SetUint(uint64(data.UnixNano()))
			case string:
				if i, err := strconv.ParseUint(data, 0, 64); err == nil {
					ReflectValueOf(ctx, structField, value).SetUint(i)
				}
			default:
				err = fallbackSetter(ctx, structField, value, v)
			}
		case reflect.Float32, reflect.Float64:
			switch data := v.(type) {
			case **float64:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetFloat(**data)
				}
			case **float32:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetFloat(float64(**data))
				}
			case float64:
				ReflectValueOf(ctx, structField, value).SetFloat(data)
			case float32:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case int64:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case int:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case int8:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case int16:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case int32:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case uint:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case uint8:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case uint16:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case uint32:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case uint64:
				ReflectValueOf(ctx, structField, value).SetFloat(float64(data))
			case []byte:
				// field.Set(ctx, value, string(data))
			case string:
				if i, err := strconv.ParseFloat(data, 64); err == nil {
					ReflectValueOf(ctx, structField, value).SetFloat(i)
				}
			default:
				err = fallbackSetter(ctx, structField, value, v)
			}
		case reflect.String:
			switch data := v.(type) {
			case **string:
				if data != nil && *data != nil {
					ReflectValueOf(ctx, structField, value).SetString(**data)
				}
			case string:
				ReflectValueOf(ctx, structField, value).SetString(data)
			case []byte:
				ReflectValueOf(ctx, structField, value).SetString(string(data))
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				ReflectValueOf(ctx, structField, value).SetString(utils.ToString(data))
			case float64, float32:
				ReflectValueOf(ctx, structField, value).SetString(fmt.Sprintf("%."+strconv.Itoa(0)+"f", data))
			default:
				err = fallbackSetter(ctx, structField, value, v)
			}
		default:
			fieldValue := reflect.New(structField.Type)
			switch fieldValue.Elem().Interface().(type) {
			case time.Time:
				switch data := v.(type) {
				case **time.Time:
					//if data != nil && *data != nil {
					//	field.Set(ctx, value, *data)
					//}
				case time.Time:
					ReflectValueOf(ctx, structField, value).Set(reflect.ValueOf(v))
				case *time.Time:
					if data != nil {
						ReflectValueOf(ctx, structField, value).Set(reflect.ValueOf(data).Elem())
					} else {
						ReflectValueOf(ctx, structField, value).Set(reflect.ValueOf(time.Time{}))
					}
				case string:
					if t, err := now.Parse(data); err == nil {
						ReflectValueOf(ctx, structField, value).Set(reflect.ValueOf(t))
					} else {
						err = fmt.Errorf("failed to set string %v to time.Time field %s, failed to parse it as time, got error %v", v, k, err)
					}
				default:
					err = fallbackSetter(ctx, structField, value, v)
				}
			case *time.Time:
				switch data := v.(type) {
				case **time.Time:
					if data != nil && *data != nil {
						ReflectValueOf(ctx, structField, value).Set(reflect.ValueOf(*data))
					}
				case time.Time:
					fieldValue := ReflectValueOf(ctx, structField, value)
					if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(structField.Type.Elem()))
					}
					fieldValue.Elem().Set(reflect.ValueOf(v))
				case *time.Time:
					ReflectValueOf(ctx, structField, value).Set(reflect.ValueOf(v))
				case string:
					if t, err := now.Parse(data); err == nil {
						fieldValue := ReflectValueOf(ctx, structField, value)
						if fieldValue.IsNil() {
							if v == "" {
								return nil
							}
							fieldValue.Set(reflect.New(structField.Type.Elem()))
						}
						fieldValue.Elem().Set(reflect.ValueOf(t))
					} else {
						err = fmt.Errorf("failed to set string %v to time.Time field %s, failed to parse it as time, got error %v", v, k, err)
					}
				default:
					err = fallbackSetter(ctx, structField, value, v)
				}
			default:
				if _, ok := fieldValue.Elem().Interface().(sql.Scanner); ok {
					// pointer scanner
					reflectV := reflect.ValueOf(v)
					if !reflectV.IsValid() {
						ReflectValueOf(ctx, structField, value).Set(reflect.New(structField.Type).Elem())
					} else if reflectV.Kind() == reflect.Ptr && reflectV.IsNil() {
						return nil
					} else if reflectV.Type().AssignableTo(structField.Type) {
						ReflectValueOf(ctx, structField, value).Set(reflectV)
					} else if reflectV.Kind() == reflect.Ptr {
						//return field.Set(ctx, value, reflectV.Elem().Interface())
						return nil
					} else {
						fieldValue := ReflectValueOf(ctx, structField, value)
						if fieldValue.IsNil() {
							fieldValue.Set(reflect.New(structField.Type.Elem()))
						}

						if valuer, ok := v.(driver.Valuer); ok {
							v, _ = valuer.Value()
						}

						err = fieldValue.Interface().(sql.Scanner).Scan(v)
					}
				} else if _, ok := fieldValue.Interface().(sql.Scanner); ok {
					// struct scanner
					reflectV := reflect.ValueOf(v)
					if !reflectV.IsValid() {
						ReflectValueOf(ctx, structField, value).Set(reflect.New(structField.Type).Elem())
					} else if reflectV.Kind() == reflect.Ptr && reflectV.IsNil() {
						return nil
					} else if reflectV.Type().AssignableTo(structField.Type) {
						ReflectValueOf(ctx, structField, value).Set(reflectV)
					} else if reflectV.Kind() == reflect.Ptr {
						//return field.Set(ctx, value, reflectV.Elem().Interface())
						return nil
					} else {
						if valuer, ok := v.(driver.Valuer); ok {
							v, _ = valuer.Value()
						}

						err = ReflectValueOf(ctx, structField, value).Addr().Interface().(sql.Scanner).Scan(v)
					}
				} else {
					err = fallbackSetter(ctx, structField, value, v)
				}
			}
		}
	case reflect.Map:
		// 处理 map 类型，通过键名设置对应的值
		value.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(value))
	}

	return
}

// GetAllFieldNamesOrKeys 获取所有字段名或键名
func (ru *ReflectUtils) GetAllFieldNamesOrKeys(elem any) ([]string, error) {
	val := reflect.ValueOf(elem)
	var names []string

	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			names = append(names, val.Type().Field(i).Name)
		}
	case reflect.Ptr:
		if val.Elem().Kind() == reflect.Struct {
			return ru.GetAllFieldNamesOrKeys(val.Elem().Interface())
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			names = append(names, fmt.Sprintf("%v", key.Interface()))
		}
	default:
		return nil, fmt.Errorf("不支持的类型")
	}

	return names, nil
}

// GetAllFieldValuesOrValues 获取所有字段值或键值
func (ru *ReflectUtils) GetAllFieldValuesOrValues(elem any) ([]any, error) {
	val := reflect.ValueOf(elem)
	var values []any

	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			values = append(values, val.Field(i).Interface())
		}
	case reflect.Ptr:
		if val.Elem().Kind() == reflect.Struct {
			return ru.GetAllFieldValuesOrValues(val.Elem().Interface())
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			values = append(values, val.MapIndex(key).Interface())
		}
	default:
		return nil, fmt.Errorf("不支持的类型")
	}

	return values, nil
}

// GetAllFieldsOrKeysAndValues 获取所有字段名或键名及其对应的值
func (ru *ReflectUtils) GetAllFieldsOrKeysAndValues(elem any) (map[string]any, error) {
	val := reflect.ValueOf(elem)
	result := make(map[string]any)

	switch val.Kind() {
	case reflect.Struct:
		for _, f := range structs.New(elem).Fields() {
			if f.IsExported() && !f.IsZero() {
				result[f.Name()] = f.Value()
			}
		}
	case reflect.Ptr:
		if val.Elem().Kind() == reflect.Struct {
			return ru.GetAllFieldsOrKeysAndValues(val.Elem().Interface())
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			result[fmt.Sprintf("%v", key.Interface())] = val.MapIndex(key).Interface()
		}
	default:
		return nil, fmt.Errorf("不支持的类型")
	}

	return result, nil
}

func ReflectValueOf(ctx context.Context, structField reflect.StructField, structValue reflect.Value) reflect.Value {
	if len(structField.Index) == 1 && structField.Index[0] > 0 {
		return reflect.Indirect(structValue).Field(structField.Index[0])
	} else {
		v := reflect.Indirect(structValue)
		for idx, fieldIdx := range structField.Index {
			if fieldIdx >= 0 {
				v = v.Field(fieldIdx)
			} else {
				v = v.Field(-fieldIdx - 1)

				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}

				if idx < len(structField.Index)-1 {
					v = v.Elem()
				}
			}
		}
		return v
	}
}

func fallbackSetter(ctx context.Context, structField reflect.StructField, value reflect.Value, v any) (err error) {
	if v == nil {
		ReflectValueOf(ctx, structField, value).Set(reflect.New(structField.Type).Elem())
	} else {
		reflectV := reflect.ValueOf(v)
		// Optimal value type acquisition for v
		reflectValType := reflectV.Type()

		if reflectValType.AssignableTo(structField.Type) {
			if reflectV.Kind() == reflect.Ptr && reflectV.Elem().Kind() == reflect.Ptr {
				reflectV = reflect.Indirect(reflectV)
			}
			ReflectValueOf(ctx, structField, value).Set(reflectV)
			return
		} else if reflectValType.ConvertibleTo(structField.Type) {
			ReflectValueOf(ctx, structField, value).Set(reflectV.Convert(structField.Type))
			return
		} else if structField.Type.Kind() == reflect.Ptr {
			fieldValue := ReflectValueOf(ctx, structField, value)
			fieldType := structField.Type.Elem()

			if reflectValType.AssignableTo(fieldType) {
				if !fieldValue.IsValid() {
					fieldValue = reflect.New(fieldType)
				} else if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(fieldType))
				}
				fieldValue.Elem().Set(reflectV)
				return
			} else if reflectValType.ConvertibleTo(fieldType) {
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(fieldType))
				}

				fieldValue.Elem().Set(reflectV.Convert(fieldType))
				return
			}
		}

		if reflectV.Kind() == reflect.Ptr {
			if reflectV.IsNil() {
				ReflectValueOf(ctx, structField, value).Set(reflect.New(structField.Type).Elem())
			} else if reflectV.Type().Elem().AssignableTo(structField.Type) {
				ReflectValueOf(ctx, structField, value).Set(reflectV.Elem())
				return
			}
		} else if _, ok := v.(clause.Expr); !ok {
			return fmt.Errorf("failed to set value %#v to field %s", v, structField.Name)
		}
	}

	return
}

// GetZeroValueOfField 返回字段类型的零值
func GetZeroValueOfField(input any, fieldName string) (reflect.Value, error) {
	// 获取输入的反射值
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)

	// 如果是指针，获取指针指向的值
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	// 处理结构体
	if v.Kind() == reflect.Struct {
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			return reflect.Value{}, fmt.Errorf("field '%s' not found in struct", fieldName)
		}
		return reflect.Zero(fieldVal.Type()), nil
	}

	// 处理 map
	if v.Kind() == reflect.Map {
		keyType := t.Key()
		elemType := t.Elem()

		// 假设字段名可以作为 map 的 key
		key := reflect.ValueOf(fieldName)
		if !key.Type().ConvertibleTo(keyType) {
			return reflect.Value{}, fmt.Errorf("field name '%s' is not convertible to map key type", fieldName)
		}

		// 返回 map 元素类型的空值
		return reflect.Zero(elemType), nil
	}

	return reflect.Value{}, fmt.Errorf("unsupported type: %s", v.Kind().String())
}

// GetZeroSliceValueOfField 返回字段值的切片类型的零值
func GetZeroSliceValueOfField(input any, fieldName string) (reflect.Value, error) {
	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if v.Kind() == reflect.Struct {
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			return reflect.Value{}, fmt.Errorf("field '%s' not found in struct", fieldName)
		}
		sliceType := reflect.SliceOf(fieldVal.Type())
		return reflect.MakeSlice(sliceType, 0, 0), nil
	}

	if v.Kind() == reflect.Map {
		elemType := t.Elem()
		sliceType := reflect.SliceOf(elemType)
		return reflect.MakeSlice(sliceType, 0, 0), nil
	}

	return reflect.Value{}, fmt.Errorf("unsupported type: %s", v.Kind().String())
}
