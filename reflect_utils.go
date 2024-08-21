package dba

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/now"
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
	value        reflect.Value
	typ          reflect.Type
	cachedValues sync.Map // 使用 sync.Map 替代普通 map
}

// NewReflectUtils 创建一个新的 ReflectUtils 对象
func NewReflectUtils(a any) (*ReflectUtils, error) {
	val := reflect.ValueOf(a)
	typ := reflect.TypeOf(a)

	return &ReflectUtils{
		raw:   a,
		value: val,
		typ:   typ,
	}, nil
}

// Raw 返回原始值
func (ru *ReflectUtils) Raw() any {
	return ru.raw
}

// Value 返回反射值
func (ru *ReflectUtils) Value() reflect.Value {
	return ru.value
}

// TypeCategory 返回变量的类型类别
func (ru *ReflectUtils) TypeCategory() TypeCategory {
	switch ru.typ.Kind() {
	case reflect.Struct:
		return CategoryStruct
	case reflect.Ptr:
		if ru.typ.Elem().Kind() == reflect.Struct {
			return CategoryStructPointer
		}
	case reflect.Map:
		if ru.typ.Key().Kind() == reflect.String && ru.typ.Elem().Kind() == reflect.Interface {
			return CategoryMapStringAny
		}
	case reflect.Slice, reflect.Array:
		elemType := ru.typ.Elem()
		switch elemType.Kind() {
		case reflect.Struct:
			return CategoryStructSliceOrArray
		case reflect.Ptr:
			if elemType.Elem().Kind() == reflect.Struct {
				return CategoryStructPointerSliceOrArray
			}
		case reflect.Map:
			if elemType.Key().Kind() == reflect.String && elemType.Elem().Kind() == reflect.Interface {
				return CategoryMapStringAnyPointerSliceOrArray
			}
		}
	}
	return CategoryUnknown
}

// CreateEmptyElement 返回切片或数组元素的空值对象
func (ru *ReflectUtils) CreateEmptyElement() (any, error) {
	elemType := ru.typ

	// 如果是切片或数组，获取元素类型
	if ru.typ.Kind() == reflect.Slice || ru.typ.Kind() == reflect.Array {
		elemType = ru.typ.Elem()
	}

	// 根据元素类型创建相应的空值对象
	switch elemType.Kind() {
	case reflect.Struct:
		return reflect.New(elemType).Elem().Interface(), nil
	case reflect.Ptr:
		if elemType.Elem().Kind() == reflect.Struct {
			return reflect.New(elemType.Elem()).Interface(), nil
		}
	case reflect.Map:
		if elemType.Key().Kind() == reflect.String && elemType.Elem().Kind() == reflect.Interface {
			return reflect.MakeMap(elemType).Interface(), nil
		}
	default:
		return reflect.Zero(elemType).Interface(), nil
	}

	return nil, fmt.Errorf("未知的元素类型")
}

// Clone 返回变量a的完整副本
func (ru *ReflectUtils) Clone() any {
	return reflect.Indirect(reflect.New(ru.typ)).Interface()
}

// CreateEmptyCopy 创建变量a的空副本
func (ru *ReflectUtils) CreateEmptyCopy() any {
	if ru.typ.Kind() == reflect.Slice {
		return reflect.MakeSlice(ru.typ, 0, 0).Interface()
	} else if ru.typ.Kind() == reflect.Array {
		return reflect.New(ru.typ).Elem().Interface()
	}
	return nil
}

// GetLen 获取数组或切片长度
func (ru *ReflectUtils) GetLen() (int, error) {
	if ru.typ.Kind() != reflect.Slice && ru.typ.Kind() != reflect.Array {
		return 0, fmt.Errorf("变量a不是切片或数组类型")
	}
	return ru.value.Len(), nil
}

// GetElement 获取指定下标的元素
func (ru *ReflectUtils) GetElement(index int) (any, error) {
	if ru.typ.Kind() != reflect.Slice && ru.typ.Kind() != reflect.Array {
		return nil, fmt.Errorf("变量a不是切片或数组类型")
	}
	if index < 0 || index >= ru.value.Len() {
		return nil, fmt.Errorf("索引超出范围")
	}
	return ru.value.Index(index).Interface(), nil
}

// getFieldReflectValue 获取指定字段的 reflect.Value。
// 如果字段是指针类型且为空，则创建新实例并解引用。
// 如果字段不存在或无效，则返回错误。
// 支持嵌套结构体的递归获取。
func getFieldReflectValue(v reflect.Value, fieldName string) (reflect.Value, error) {
	v = reflect.Indirect(v) // 解引用指针，以获取实际值而非指针本身
	for v.Kind() == reflect.Struct {
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			return reflect.Value{}, fmt.Errorf("字段%s不存在", fieldName)
		}
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldVal.Type().Elem())) // 如果指针为空，创建新实例
			}
			v = fieldVal.Elem() // 继续解引用指针
		} else {
			return fieldVal, nil
		}
	}
	return reflect.Value{}, fmt.Errorf("字段%s无法获取值", fieldName)
}

// GetFieldOrKey 获取指定结构体字段或 map 键的值。
// 如果是结构体，将尝试获取字段的 reflect.Value 并返回其 Interface()。
// 如果是 map，将尝试获取指定键的值。
// 如果字段或键不存在，或类型不支持，返回错误。
func (ru *ReflectUtils) GetFieldOrKey(elem any, name string) (any, error) {
	val := reflect.ValueOf(elem)

	switch val.Kind() {
	case reflect.Struct, reflect.Ptr:
		// 获取目标字段的 reflect.Value
		fieldVal, err := getFieldReflectValue(val, name)
		if err != nil {
			return nil, err
		}
		return fieldVal.Interface(), nil

	case reflect.Map:
		// 处理 map 类型，通过键名获取对应的值
		keyVal := val.MapIndex(reflect.ValueOf(name))
		if !keyVal.IsValid() {
			return nil, fmt.Errorf("键%s不存在", name)
		}
		return keyVal.Interface(), nil
	}

	return nil, fmt.Errorf("不支持的类型")
}

// SetFieldOrKey 设置指定结构体字段或 map 键的值。
// 对于结构体类型，尝试获取字段的 reflect.Value 并设置新值。
// 对于 map 类型，通过键名设置对应的值。
// 如果字段不可设置或类型不支持，返回错误。
func (ru *ReflectUtils) SetFieldOrKey(elem any, name string, value any) error {
	val := reflect.ValueOf(elem)

	switch val.Kind() {
	case reflect.Struct, reflect.Ptr:
		// 获取目标字段的 reflect.Value
		fieldVal, err := getFieldReflectValue(val, name)
		if err != nil {
			return err
		}

		if !fieldVal.CanSet() {
			return fmt.Errorf("字段%s不可设置", name)
		}

		// 根据字段类型进行相应的设置操作
		switch fieldVal.Kind() {
		case reflect.Bool:
			if data, ok := value.(bool); ok {
				fieldVal.SetBool(data)
				return nil
			} else if data, ok := value.(string); ok {
				b, _ := strconv.ParseBool(data)
				fieldVal.SetBool(b)
				return nil
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if data, ok := value.(int64); ok {
				fieldVal.SetInt(data)
				return nil
			} else if data, ok := value.(string); ok {
				if i, err := strconv.ParseInt(data, 0, 64); err == nil {
					fieldVal.SetInt(i)
					return nil
				} else {
					return err
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if data, ok := value.(uint64); ok {
				fieldVal.SetUint(data)
				return nil
			} else if data, ok := value.(string); ok {
				if i, err := strconv.ParseUint(data, 0, 64); err == nil {
					fieldVal.SetUint(i)
					return nil
				} else {
					return err
				}
			}
		case reflect.Float32, reflect.Float64:
			if data, ok := value.(float64); ok {
				fieldVal.SetFloat(data)
				return nil
			} else if data, ok := value.(string); ok {
				if f, err := strconv.ParseFloat(data, 64); err == nil {
					fieldVal.SetFloat(f)
					return nil
				} else {
					return err
				}
			}
		case reflect.String:
			if data, ok := value.(string); ok {
				fieldVal.SetString(data)
				return nil
			}
		case reflect.Struct:
			// 处理 time.Time 类型
			if fieldVal.Type() == reflect.TypeOf(time.Time{}) {
				if data, ok := value.(time.Time); ok {
					fieldVal.Set(reflect.ValueOf(data))
					return nil
				} else if data, ok := value.(string); ok {
					if t, err := now.Parse(data); err == nil {
						fieldVal.Set(reflect.ValueOf(t))
						return nil
					} else {
						return fmt.Errorf("无法将值 %v 设置为 time.Time 类型字段 %s: %v", value, name, err)
					}
				}
			}
		case reflect.Ptr:
			// 处理指针类型，创建新的实例并解引用设置值
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
			}
			if reflect.TypeOf(value).AssignableTo(fieldVal.Type().Elem()) {
				fieldVal.Elem().Set(reflect.ValueOf(value))
				return nil
			} else if reflect.TypeOf(value).ConvertibleTo(fieldVal.Type().Elem()) {
				fieldVal.Elem().Set(reflect.ValueOf(value).Convert(fieldVal.Type().Elem()))
				return nil
			}
			// 处理指向时间类型的指针
			if fieldVal.Type() == reflect.TypeOf(&time.Time{}) {
				if data, ok := value.(*time.Time); ok {
					fieldVal.Set(reflect.ValueOf(data))
					return nil
				} else if data, ok := value.(string); ok {
					if t, err := now.Parse(data); err == nil {
						fieldVal.Set(reflect.ValueOf(&t))
						return nil
					} else {
						return fmt.Errorf("无法将值 %v 设置为 *time.Time 类型字段 %s: %v", value, name, err)
					}
				}
			}
		}

		// 序列化与扫描器处理
		if scanner, ok := fieldVal.Addr().Interface().(sql.Scanner); ok {
			if err := scanner.Scan(value); err != nil {
				return fmt.Errorf("无法将值 %v 扫描到字段 %s: %v", value, name, err)
			}
			return nil
		}

		// 如果没有匹配到特殊类型，使用默认的 Set 方式
		fieldVal.Set(reflect.ValueOf(value))
		return nil

	case reflect.Map:
		// 处理 map 类型，通过键名设置对应的值
		val.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(value))
		return nil
	}

	return fmt.Errorf("不支持的类型")
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
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			result[field.Name] = val.Field(i).Interface()
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
