package main

import (
	"fmt"
	"reflect"
)

type Others struct {
	CC string
}

type Address struct {
	City  string
	State string
}

type Person struct {
	Name string
	Age  int
	*Address
	*Others
}

// getFieldValueByName 根据字段名称获取结构体的值，支持嵌套结构体
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

func main() {
	person := Person{
		Name: "John",
		Age:  30,
		Address: &Address{
			City:  "New York",
			State: "NY",
		},
		Others: &Others{},
	}

	fmt.Println("Name:", getFieldValueByName(person, "Name"))       // 输出 "John"
	fmt.Println("City:", getFieldValueByName(person, "City"))       // 输出 "New York"
	fmt.Println("State:", getFieldValueByName(person, "State"))     // 输出 "NY"
	fmt.Println("CC:", getFieldValueByName(person, "CC"))           // 输出 <nil>
	fmt.Println("Invalid:", getFieldValueByName(person, "Invalid")) // 输出 <nil>
}
