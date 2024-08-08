package dba

import (
	"fmt"
	"reflect"
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

// ReflectUtils 提供反射工具方法
type ReflectUtils struct {
	value reflect.Value
	typ   reflect.Type
}

// NewReflectUtils 创建一个新的 ReflectUtils 对象
func NewReflectUtils(a any) (*ReflectUtils, error) {
	val := reflect.ValueOf(a)
	typ := reflect.TypeOf(a)

	return &ReflectUtils{
		value: val,
		typ:   typ,
	}, nil
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

// GetFieldOrKey 获取元素的指定字段或键的值
func (ru *ReflectUtils) GetFieldOrKey(elem any, name string) (any, error) {
	val := reflect.ValueOf(elem)
	switch val.Kind() {
	case reflect.Struct:
		fieldVal := val.FieldByName(name)
		if !fieldVal.IsValid() {
			return nil, fmt.Errorf("字段%s不存在", name)
		}
		return fieldVal.Interface(), nil
	case reflect.Ptr:
		if val.Elem().Kind() == reflect.Struct {
			return ru.GetFieldOrKey(val.Elem().Interface(), name)
		}
	case reflect.Map:
		keyVal := val.MapIndex(reflect.ValueOf(name))
		if !keyVal.IsValid() {
			return nil, fmt.Errorf("键%s不存在", name)
		}
		return keyVal.Interface(), nil
	}
	return nil, fmt.Errorf("不支持的类型")
}

// SetFieldOrKey 更新元素的指定字段或键的值
func (ru *ReflectUtils) SetFieldOrKey(elem any, name string, value any) error {
	val := reflect.ValueOf(elem)
	switch val.Kind() {
	case reflect.Struct:
		fieldVal := val.FieldByName(name)
		if !fieldVal.IsValid() {
			return fmt.Errorf("字段%s不存在", name)
		}
		if !fieldVal.CanSet() {
			return fmt.Errorf("字段%s不可设置", name)
		}
		fieldVal.Set(reflect.ValueOf(value))
		return nil
	case reflect.Ptr:
		if val.Elem().Kind() == reflect.Struct {
			return ru.SetFieldOrKey(val.Elem().Interface(), name, value)
		}
	case reflect.Map:
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
