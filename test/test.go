package main

import (
	"fmt"
	"reflect"
)

func main() {
	// 示例结构体
	type Person struct {
		Name string
		Age  int
	}

	// 示例数据
	person := Person{Name: "John", Age: 30}
	personPtr := &Person{Name: "Doe", Age: 40}
	personArray := []Person{{Name: "Alice", Age: 25}, {Name: "Bob", Age: 35}}
	personPtrArray := []*Person{{Name: "Charlie", Age: 28}, {Name: "Dave", Age: 38}}

	// 调用处理函数
	processAny(&person)
	processAny(personPtr)
	processAny(&personArray)
	processAny(&personPtrArray)

	// 输出结果
	fmt.Printf("Modified person: %+v\n", person)
	fmt.Printf("Modified personPtr: %+v\n", personPtr)
	fmt.Printf("Modified personArray: %+v\n", personArray)
	fmt.Printf("Modified personPtrArray: %+v\n", personPtrArray)
}

func processAny(data any) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		fmt.Println("Processing struct")
		processStruct(v)

	case reflect.Slice:
		fmt.Println("Processing slice")
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if elem.Kind() == reflect.Struct {
				processStruct(elem)
			}
		}

	default:
		fmt.Println("Unsupported type")
	}
}

func processStruct(v reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			fmt.Printf("Field %d is not settable\n", i)
			continue
		}
		fmt.Printf("Field %d: %v\n", i, field.Interface())

		// 示例处理：如果字段是字符串类型，将其值修改为大写
		if field.Kind() == reflect.String {
			field.SetString(field.String() + "__Modified")
		} else if field.Kind() == reflect.Int {
			field.SetInt(field.Int() + 1)
		}
	}
	fmt.Printf("Modified struct: %+v\n", v.Interface())
}
