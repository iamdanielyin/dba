package dba

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/jinzhu/now"
	"go/ast"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
	"reflect"
	"strconv"
	"time"
)

type ReflectValue struct {
	src any
	raw reflect.Value
	reflect.Value
}

func NewReflectValue(src any) *ReflectValue {
	if src == nil {
		return &ReflectValue{
			src:   nil,
			raw:   reflect.Value{},
			Value: reflect.Value{},
		}
	}

	var raw reflect.Value
	switch v := src.(type) {
	case reflect.Value:
		raw = v
	case *reflect.Value:
		if v == nil {
			return &ReflectValue{
				src:   nil,
				raw:   reflect.Value{},
				Value: reflect.Value{},
			}
		}
		raw = *v
	default:
		raw = reflect.ValueOf(v)
	}
	return &ReflectValue{
		src:   src,
		raw:   raw,
		Value: reflect.Indirect(raw),
	}
}

func NewVar(src any) reflect.Value {
	val := reflect.Indirect(reflect.ValueOf(src))
	return reflect.New(val.Type())
}

type ValueIs string

const (
	ValueIsStruct      ValueIs = "STRUCT"
	ValueIsStructArray ValueIs = "STRUCT_ARRAY"
	ValueIsMap         ValueIs = "MAP"
	ValueIsMapArray    ValueIs = "MAP_ARRAY"
	ValueIsUnknown     ValueIs = "UNKNOWN"
)

func (rv *ReflectValue) ValueIs() ValueIs {
	v := rv.Value
	if !v.IsValid() {
		return ValueIsUnknown
	}

	// 处理指针类型
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ValueIsUnknown
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return ValueIsStruct
	case reflect.Map:
		return ValueIsMap
	case reflect.Slice, reflect.Array:
		if v.IsNil() {
			return ValueIsUnknown
		}
		elemType := v.Type().Elem()
		// 处理元素为指针的情况
		for elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		switch elemType.Kind() {
		case reflect.Struct:
			return ValueIsStructArray
		case reflect.Map:
			return ValueIsMapArray
		default:
			return ValueIsUnknown
		}
	default:
		return ValueIsUnknown
	}
}

func (rv *ReflectValue) IsArray() bool {
	vi := rv.ValueIs()
	return vi == ValueIsStructArray || vi == ValueIsMapArray
}

func (rv *ReflectValue) Src() any {
	return rv.src
}

func (rv *ReflectValue) Raw() *reflect.Value {
	return &(rv.raw)
}

func (rv *ReflectValue) FieldByName(fieldName string) *reflect.Value {
	if !rv.Value.IsValid() {
		return nil
	}

	switch rv.Value.Kind() {
	case reflect.Struct:
		v := rv.Value
		if v.IsZero() {
			return nil
		}

		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			return nil
		}
		return &field

	case reflect.Map:
		if rv.Value.IsNil() {
			return nil
		}

		for _, key := range rv.Value.MapKeys() {
			if key.String() == fieldName {
				val := rv.Value.MapIndex(key)
				if !val.IsValid() {
					return nil
				}
				return &val
			}
		}
	}
	return nil
}

func (rv *ReflectValue) Keys() []string {
	entries := rv.Map()
	var keys []string
	if len(entries) > 0 {
		for k, _ := range entries {
			keys = append(keys, k)
		}
	}
	return keys
}

func (rv *ReflectValue) Values() []any {
	entries := rv.Map()
	var values []any
	if len(entries) > 0 {
		for _, v := range entries {
			values = append(values, v)
		}
	}
	return values
}

func (rv *ReflectValue) Map() map[string]any {
	if !rv.Value.IsValid() {
		return nil
	}

	entries := make(map[string]any)

	switch rv.Value.Kind() {
	case reflect.Struct:
		parseStructToMap(rv.Value, entries, nil)
	case reflect.Map:
		if rv.Value.IsNil() {
			return entries
		}
		for _, k := range rv.Value.MapKeys() {
			v := rv.Value.MapIndex(k)
			if !v.IsValid() {
				continue
			}
			// 处理接口类型
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
			entries[fmt.Sprintf("%v", k)] = v.Interface()
		}
	}

	return entries
}

func parseStructToMap(v reflect.Value, result map[string]any, visited map[uintptr]bool) {
	if !v.IsValid() {
		return
	}

	// 防止循环引用
	if visited == nil {
		visited = make(map[uintptr]bool)
	}
	ptr := v.Pointer()
	if visited[ptr] {
		return
	}
	visited[ptr] = true

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.IsValid() || !fieldType.IsExported() {
			continue
		}

		// 处理接口类型
		if field.Kind() == reflect.Interface {
			if !field.IsNil() {
				field = field.Elem()
			}
		}

		// 处理指针类型
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}

		if field.IsZero() {
			continue
		}

		// 处理嵌套结构体
		if field.Kind() == reflect.Struct {
			parseStructToMap(field, result, visited)
		} else {
			result[fieldType.Name] = field.Interface()
		}
	}
}

func hasValue(v *reflect.Value) bool {
	// 如果 v 是零值
	if v == nil || !v.IsValid() {
		return false // 无效的值
	}

	// 对于指针、接口、切片、映射、通道等类型，可以使用 IsNil() 判断
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan:
		// 只有这些类型可以是 nil
		return !v.IsNil()
	default:
		// 对于其他类型，检查它们是否是零值
		// 对于整数、浮点数、布尔类型、字符串等基本类型，零值都可以通过 `IsZero` 判断
		return !v.IsZero()
	}
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

func SetFieldOrKey(elem any, k string, v any) (err error) {
	value := reflect.ValueOf(elem)
	if !value.IsValid() {
		return fmt.Errorf("invalid value")
	}

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
	default:
	}

	return
}

func ReflectValueOf(ctx context.Context, structField reflect.StructField, structValue reflect.Value) reflect.Value {
	if !structValue.IsValid() {
		return reflect.Value{}
	}

	v := reflect.Indirect(structValue)
	if !v.IsValid() {
		return reflect.Value{}
	}

	// 处理单个索引的情况
	if len(structField.Index) == 1 {
		if structField.Index[0] >= v.NumField() {
			return reflect.Value{}
		}
		return v.Field(structField.Index[0])
	}

	// 处理嵌套字段
	for idx, fieldIdx := range structField.Index {
		if !v.IsValid() {
			return reflect.Value{}
		}

		if fieldIdx >= 0 {
			if fieldIdx >= v.NumField() {
				return reflect.Value{}
			}
			v = v.Field(fieldIdx)
		} else {
			fieldIdx = -fieldIdx - 1
			if fieldIdx >= v.NumField() {
				return reflect.Value{}
			}
			v = v.Field(fieldIdx)

			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}
				if idx < len(structField.Index)-1 {
					v = v.Elem()
				}
			}
		}
	}

	return v
}

func CopyEmptyValue(typ reflect.Type) any {
	if typ == nil {
		return nil
	}

	switch typ.Kind() {
	case reflect.Struct:
		return reflect.New(typ).Elem().Interface()
	case reflect.Ptr:
		if typ.Elem() == nil {
			return nil
		}
		return reflect.New(typ.Elem()).Interface()
	case reflect.Map:
		return reflect.MakeMap(typ).Interface()
	case reflect.Slice:
		return reflect.MakeSlice(typ, 0, 0).Interface()
	case reflect.Array:
		return reflect.New(typ).Elem().Interface()
	case reflect.Chan:
		return reflect.MakeChan(typ, 0).Interface()
	default:
		return reflect.Zero(typ).Interface()
	}
}

func CopyEmptyArray(typ reflect.Type) any {
	if typ == nil {
		return nil
	}

	switch typ.Kind() {
	case reflect.Ptr:
		if typ.Elem() == nil {
			return nil
		}
		newPtr := reflect.New(typ.Elem())
		// 如果指针指向的是数组或切片，初始化它
		if typ.Elem().Kind() == reflect.Slice || typ.Elem().Kind() == reflect.Array {
			newPtr.Elem().Set(reflect.MakeSlice(typ.Elem(), 0, 0))
		}
		return newPtr.Interface()
	case reflect.Slice:
		return reflect.MakeSlice(typ, 0, 0).Interface()
	case reflect.Array:
		return reflect.New(typ).Elem().Interface()
	default:
		return reflect.Zero(typ).Interface()
	}
}
