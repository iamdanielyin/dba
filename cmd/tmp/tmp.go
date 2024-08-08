package main

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

func main() {
	// 示例1：结构体
	type SampleStruct struct {
		Code string
	}
	var a1 = SampleStruct{Code: "test1"}
	utils1, err := NewReflectUtils(a1)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("类型类别: %v\n", utils1.TypeCategory())

		// 获取字段值
		fieldValue, err := utils1.GetFieldOrKey(a1, "Code")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("字段值: %v\n", fieldValue)
		}

		// 更新字段值
		err = utils1.SetFieldOrKey(&a1, "Code", "newTest1")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("更新后的结构体: %v\n", a1)
		}

		// 获取所有字段名
		fieldNames, err := utils1.GetAllFieldNamesOrKeys(a1)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名: %v\n", fieldNames)
		}

		// 获取所有字段值
		fieldValues, err := utils1.GetAllFieldValuesOrValues(a1)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段值: %v\n", fieldValues)
		}

		// 获取所有字段名及值
		fieldsAndValues, err := utils1.GetAllFieldsOrKeysAndValues(a1)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名及值: %v\n", fieldsAndValues)
		}
	}

	// 示例2：结构体指针
	var a2 = &SampleStruct{Code: "test2"}
	utils2, err := NewReflectUtils(a2)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("类型类别: %v\n", utils2.TypeCategory())

		// 获取字段值
		fieldValue, err := utils2.GetFieldOrKey(a2, "Code")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("字段值: %v\n", fieldValue)
		}

		// 更新字段值
		err = utils2.SetFieldOrKey(a2, "Code", "newTest2")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("更新后的结构体指针: %v\n", a2)
		}

		// 获取所有字段名
		fieldNames, err := utils2.GetAllFieldNamesOrKeys(a2)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名: %v\n", fieldNames)
		}

		// 获取所有字段值
		fieldValues, err := utils2.GetAllFieldValuesOrValues(a2)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段值: %v\n", fieldValues)
		}

		// 获取所有字段名及值
		fieldsAndValues, err := utils2.GetAllFieldsOrKeysAndValues(a2)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名及值: %v\n", fieldsAndValues)
		}
	}

	// 示例3：map[string]any
	var a3 = map[string]any{"key": "value"}
	utils3, err := NewReflectUtils(a3)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("类型类别: %v\n", utils3.TypeCategory())

		// 获取键值
		keyValue, err := utils3.GetFieldOrKey(a3, "key")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("键值: %v\n", keyValue)
		}

		// 更新键值
		err = utils3.SetFieldOrKey(a3, "key", "newValue")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("更新后的map: %v\n", a3)
		}

		// 获取所有键名
		fieldNames, err := utils3.GetAllFieldNamesOrKeys(a3)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有键名: %v\n", fieldNames)
		}

		// 获取所有键值
		fieldValues, err := utils3.GetAllFieldValuesOrValues(a3)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有键值: %v\n", fieldValues)
		}

		// 获取所有键名及值
		fieldsAndValues, err := utils3.GetAllFieldsOrKeysAndValues(a3)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有键名及值: %v\n", fieldsAndValues)
		}
	}

	// 示例4：结构体切片
	var a4 = []SampleStruct{{Code: "test3"}}
	utils4, err := NewReflectUtils(a4)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("类型类别: %v\n", utils4.TypeCategory())

		// 获取元素
		elem4, err := utils4.GetElement(0)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("获取的元素: %v\n", elem4)
		}

		// 获取字段值
		fieldValue, err := utils4.GetFieldOrKey(elem4, "Code")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("字段值: %v\n", fieldValue)
		}

		// 更新字段值
		err = utils4.SetFieldOrKey(&a4[0], "Code", "newTest3")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("更新后的元素: %v\n", a4[0])
		}

		// 获取所有字段名
		fieldNames, err := utils4.GetAllFieldNamesOrKeys(elem4)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名: %v\n", fieldNames)
		}

		// 获取所有字段值
		fieldValues, err := utils4.GetAllFieldValuesOrValues(elem4)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段值: %v\n", fieldValues)
		}

		// 获取所有字段名及值
		fieldsAndValues, err := utils4.GetAllFieldsOrKeysAndValues(elem4)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名及值: %v\n", fieldsAndValues)
		}
	}

	// 示例5：结构体指针切片
	var a5 = []*SampleStruct{{Code: "test4"}}
	utils5, err := NewReflectUtils(a5)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("类型类别: %v\n", utils5.TypeCategory())

		// 获取元素
		elem5, err := utils5.GetElement(0)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("获取的元素: %v\n", elem5)
		}

		// 获取字段值
		fieldValue, err := utils5.GetFieldOrKey(elem5, "Code")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("字段值: %v\n", fieldValue)
		}

		// 更新字段值
		err = utils5.SetFieldOrKey(a5[0], "Code", "newTest4")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("更新后的元素: %v\n", a5[0])
		}

		// 获取所有字段名
		fieldNames, err := utils5.GetAllFieldNamesOrKeys(elem5)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名: %v\n", fieldNames)
		}

		// 获取所有字段值
		fieldValues, err := utils5.GetAllFieldValuesOrValues(elem5)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段值: %v\n", fieldValues)
		}

		// 获取所有字段名及值
		fieldsAndValues, err := utils5.GetAllFieldsOrKeysAndValues(elem5)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有字段名及值: %v\n", fieldsAndValues)
		}
	}

	// 示例6：map[string]any指针切片
	var a6 = []*map[string]any{{"key": "value"}}
	utils6, err := NewReflectUtils(a6)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("类型类别: %v\n", utils6.TypeCategory())

		// 获取元素
		elem6, err := utils6.GetElement(0)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("获取的元素: %v\n", elem6)
		}

		// 获取键值
		keyValue, err := utils6.GetFieldOrKey(elem6, "key")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("键值: %v\n", keyValue)
		}

		// 更新键值
		err = utils6.SetFieldOrKey(a6[0], "key", "newValue")
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("更新后的元素: %v\n", a6[0])
		}

		// 获取所有键名
		fieldNames, err := utils6.GetAllFieldNamesOrKeys(elem6)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有键名: %v\n", fieldNames)
		}

		// 获取所有键值
		fieldValues, err := utils6.GetAllFieldValuesOrValues(elem6)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有键值: %v\n", fieldValues)
		}

		// 获取所有键名及值
		fieldsAndValues, err := utils6.GetAllFieldsOrKeysAndValues(elem6)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("所有键名及值: %v\n", fieldsAndValues)
		}
	}
}
