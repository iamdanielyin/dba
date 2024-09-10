package dba

import (
	"reflect"
)

type ReflectValue struct {
	src any
	raw reflect.Value
	reflect.Value
}

func NewReflectValue(src any) *ReflectValue {
	raw := reflect.ValueOf(src)
	return &ReflectValue{
		src:   src,
		raw:   raw,
		Value: reflect.Indirect(raw),
	}
}

func (rv *ReflectValue) Src() any {
	return rv.src
}

func (rv *ReflectValue) Raw() *reflect.Value {
	return &(rv.raw)
}

func (rv *ReflectValue) FieldByName(fieldName string) *reflect.Value {
	switch rv.Value.Kind() {
	case reflect.Struct:
		v := rv.Value
		if v.IsValid() && v.IsZero() {
			return nil
		}
		var fieldVal reflect.Value
		if f, ok := v.Type().FieldByName(fieldName); !ok {
			return nil
		} else {
			if len(f.Index) == 1 {
				fieldVal = v.Field(f.Index[0])
			} else {
				fieldVal = v
				for _, x := range f.Index {
					fieldVal = reflect.Indirect(fieldVal.Field(x))
					if !fieldVal.IsValid() || fieldVal.IsZero() {
						break
					}
				}
			}
		}

		if fieldVal.IsValid() && !fieldVal.IsZero() {
			return &fieldVal
		}
	case reflect.Map:
		for _, key := range rv.Value.MapKeys() {
			key = reflect.Indirect(key)
			switch key.Kind() {
			case reflect.String:
				if key.Interface().(string) == fieldName {
					fieldVal := rv.Value.MapIndex(key)
					return &fieldVal
				}
			default:
				continue
			}
		}
	default:
	}
	return nil
}

func StructToMap(input interface{}) map[string]any {
	result := make(map[string]any)
	v := reflect.ValueOf(input)

	if v.Kind() == reflect.Ptr {
		v = v.Elem() // 获取指针指向的值
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// 递归解析结构体字段
	parseStructToMap(v, result)
	return result
}

func parseStructToMap(v reflect.Value, result map[string]any) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		if !field.IsValid() {
			continue
		}
		// 检查是否为嵌套结构体（指针或非指针）
		if field.Kind() == reflect.Ptr {
			// 如果是指针且非空，解引用指针
			if !field.IsNil() {
				field = field.Elem()
			} else {
				// 跳过空指针的嵌套结构体
				continue
			}
		}

		// 只处理非零值字段
		if field.IsZero() {
			continue
		}

		// 处理嵌套结构体（包括匿名字段）
		if field.Kind() == reflect.Struct {
			// 递归解析嵌套结构体，直接将嵌套结构体的字段展开
			parseStructToMap(field, result)
		} else {
			// 存储非零字段名和值到 map 中
			result[fieldName] = field.Interface()
		}
	}
}

func getFieldValueByName(obj any, fieldName string) any {
	if obj == nil {
		return nil
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	if v.IsValid() && v.IsZero() {
		return nil
	}

	// 尝试获取字段
	var fieldVal reflect.Value
	if f, ok := v.Type().FieldByName(fieldName); !ok {
		return nil
	} else {
		if len(f.Index) == 1 {
			fieldVal = v.Field(f.Index[0])
		} else {
			fieldVal = v
			for _, x := range f.Index {
				fieldVal = reflect.Indirect(fieldVal.Field(x))
				if !fieldVal.IsValid() || fieldVal.IsZero() {
					break
				}
			}
		}
	}

	if fieldVal.IsValid() && !fieldVal.IsZero() {
		return fieldVal.Interface()
	}

	return nil
}
