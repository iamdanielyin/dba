package main

import (
	"fmt"
	"reflect"
)

type ValueWrapper struct {
	value    any
	refValue reflect.Value
	refType  reflect.Type
}

func NewValueWrapper(v any) *ValueWrapper {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Struct {
		ptr := reflect.New(val.Type())
		ptr.Elem().Set(val)
		val = ptr
	}
	return &ValueWrapper{
		value:    v,
		refValue: val,
		refType:  val.Type(),
	}
}

// Helper function to ensure the value is addressable
func (vw *ValueWrapper) getAddressableValue() reflect.Value {
	v := vw.refValue
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// For struct or struct pointer: Get field value by name
func (vw *ValueWrapper) GetField(fieldName string) (any, error) {
	v := vw.getAddressableValue()
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("value is not a struct or struct pointer")
	}
	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return nil, fmt.Errorf("no such field: %s", fieldName)
	}
	return f.Interface(), nil
}

// For struct or struct pointer: Set field value by name
func (vw *ValueWrapper) SetField(fieldName string, value any) error {
	v := vw.getAddressableValue()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("value is not a struct or struct pointer")
	}
	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return fmt.Errorf("no such field: %s", fieldName)
	}
	if !f.CanSet() {
		return fmt.Errorf("cannot set field: %s", fieldName)
	}

	val := reflect.ValueOf(value)
	if val.Type() != f.Type() {
		val = val.Convert(f.Type())
	}

	f.Set(val)
	return nil
}

// For map: Get value by key
func (vw *ValueWrapper) GetMapValue(key string) (any, error) {
	if vw.refValue.Kind() != reflect.Map {
		return nil, fmt.Errorf("value is not a map")
	}
	val := vw.refValue.MapIndex(reflect.ValueOf(key))
	if !val.IsValid() {
		return nil, fmt.Errorf("no such key: %s", key)
	}
	return val.Interface(), nil
}

// For map: Set value by key
func (vw *ValueWrapper) SetMapValue(key string, value any) error {
	if vw.refValue.Kind() != reflect.Map {
		return fmt.Errorf("value is not a map")
	}
	vw.refValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	return nil
}

// For slice or array: Get element by index
func (vw *ValueWrapper) GetElement(index int) (any, error) {
	if vw.refValue.Kind() != reflect.Slice && vw.refValue.Kind() != reflect.Array {
		return nil, fmt.Errorf("value is not a slice or array")
	}
	if index < 0 || index >= vw.refValue.Len() {
		return nil, fmt.Errorf("index out of range")
	}
	return vw.refValue.Index(index).Interface(), nil
}

// For slice or array: Set element by index
func (vw *ValueWrapper) SetElement(index int, value any) error {
	if vw.refValue.Kind() != reflect.Slice && vw.refValue.Kind() != reflect.Array {
		return fmt.Errorf("value is not a slice or array")
	}
	if index < 0 || index >= vw.refValue.Len() {
		return fmt.Errorf("index out of range")
	}
	vw.refValue.Index(index).Set(reflect.ValueOf(value))
	return nil
}

// For slice of structs or struct pointers: Get field value by index and field name
func (vw *ValueWrapper) GetElementField(index int, fieldName string) (any, error) {
	elem, err := vw.GetElement(index)
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(elem)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("element is not a struct or struct pointer")
	}
	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return nil, fmt.Errorf("no such field: %s", fieldName)
	}
	return f.Interface(), nil
}

// For slice of structs or struct pointers: Set field value by index and field name
func (vw *ValueWrapper) SetElementField(index int, fieldName string, value any) error {
	elem, err := vw.GetElement(index)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(elem)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("element is not a struct or struct pointer")
	}
	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return fmt.Errorf("no such field: %s", fieldName)
	}
	if !f.CanSet() {
		return fmt.Errorf("cannot set field: %s", fieldName)
	}

	val := reflect.ValueOf(value)
	if val.Type() != f.Type() {
		val = val.Convert(f.Type())
	}

	f.Set(val)
	return nil
}

func main() {
	// Example usage
	type Example struct {
		Name string
		Age  int
	}

	example := Example{Name: "John", Age: 30}
	vw := NewValueWrapper(&example)

	// Get field value
	name, err := vw.GetField("Name")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Name:", name)
	}

	// Set field value
	err = vw.SetField("Age", 31)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Updated Age:", example.Age)
	}

	// Example usage with map
	exampleMap := map[string]any{"Name": "John", "Age": 30}
	vwMap := NewValueWrapper(exampleMap)

	// Get map value
	name, err = vwMap.GetMapValue("Name")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Name:", name)
	}

	// Set map value
	err = vwMap.SetMapValue("Age", 31)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Updated Age in map:", exampleMap["Age"])
	}

	// Example usage with slice of structs
	exampleSlice := []Example{{Name: "Alice", Age: 25}, {Name: "Bob", Age: 35}}
	vwSlice := NewValueWrapper(exampleSlice)

	// Get element field value
	name, err = vwSlice.GetElementField(0, "Name")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("First element's Name:", name)
	}

	// Set element field value
	err = vwSlice.SetElementField(1, "Age", 36)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Updated Age in second element:", exampleSlice[1].Age)
	}
}
