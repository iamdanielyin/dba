package dba

import (
	"reflect"
	"testing"
)

type SampleStruct struct {
	Code string
}

func TestReflectUtils(t *testing.T) {
	// 测试结构体
	var a1 = SampleStruct{Code: "test1"}
	utils1, err := NewReflectUtils(a1)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if utils1.TypeCategory() != CategoryStruct {
		t.Fatalf("Expected CategoryStruct, got %v", utils1.TypeCategory())
	}

	fieldValue, err := utils1.GetFieldOrKey(a1, "Code")
	if err != nil || fieldValue != "test1" {
		t.Fatalf("Error: %v, fieldValue: %v", err, fieldValue)
	}

	err = utils1.SetFieldOrKey(&a1, "Code", "newTest1")
	if err != nil || a1.Code != "newTest1" {
		t.Fatalf("Error: %v, updated value: %v", err, a1.Code)
	}

	fieldNames, err := utils1.GetAllFieldNamesOrKeys(a1)
	if err != nil || len(fieldNames) != 1 || fieldNames[0] != "Code" {
		t.Fatalf("Error: %v, fieldNames: %v", err, fieldNames)
	}

	fieldValues, err := utils1.GetAllFieldValuesOrValues(a1)
	if err != nil || len(fieldValues) != 1 || fieldValues[0] != "newTest1" {
		t.Fatalf("Error: %v, fieldValues: %v", err, fieldValues)
	}

	fieldsAndValues, err := utils1.GetAllFieldsOrKeysAndValues(a1)
	if err != nil || len(fieldsAndValues) != 1 || fieldsAndValues["Code"] != "newTest1" {
		t.Fatalf("Error: %v, fieldsAndValues: %v", err, fieldsAndValues)
	}

	// 测试结构体指针
	var a2 = &SampleStruct{Code: "test2"}
	utils2, err := NewReflectUtils(a2)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if utils2.TypeCategory() != CategoryStructPointer {
		t.Fatalf("Expected CategoryStructPointer, got %v", utils2.TypeCategory())
	}

	fieldValue, err = utils2.GetFieldOrKey(a2, "Code")
	if err != nil || fieldValue != "test2" {
		t.Fatalf("Error: %v, fieldValue: %v", err, fieldValue)
	}

	err = utils2.SetFieldOrKey(a2, "Code", "newTest2")
	if err != nil || a2.Code != "newTest2" {
		t.Fatalf("Error: %v, updated value: %v", err, a2.Code)
	}

	fieldNames, err = utils2.GetAllFieldNamesOrKeys(a2)
	if err != nil || len(fieldNames) != 1 || fieldNames[0] != "Code" {
		t.Fatalf("Error: %v, fieldNames: %v", err, fieldNames)
	}

	fieldValues, err = utils2.GetAllFieldValuesOrValues(a2)
	if err != nil || len(fieldValues) != 1 || fieldValues[0] != "newTest2" {
		t.Fatalf("Error: %v, fieldValues: %v", err, fieldValues)
	}

	fieldsAndValues, err = utils2.GetAllFieldsOrKeysAndValues(a2)
	if err != nil || len(fieldsAndValues) != 1 || fieldsAndValues["Code"] != "newTest2" {
		t.Fatalf("Error: %v, fieldsAndValues: %v", err, fieldsAndValues)
	}

	// 测试 map[string]any
	var a3 = map[string]any{"key": "value"}
	utils3, err := NewReflectUtils(a3)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if utils3.TypeCategory() != CategoryMapStringAny {
		t.Fatalf("Expected CategoryMapStringAny, got %v", utils3.TypeCategory())
	}

	keyValue, err := utils3.GetFieldOrKey(a3, "key")
	if err != nil || keyValue != "value" {
		t.Fatalf("Error: %v, keyValue: %v", err, keyValue)
	}

	err = utils3.SetFieldOrKey(a3, "key", "newValue")
	if err != nil || a3["key"] != "newValue" {
		t.Fatalf("Error: %v, updated value: %v", err, a3["key"])
	}

	fieldNames, err = utils3.GetAllFieldNamesOrKeys(a3)
	if err != nil || len(fieldNames) != 1 || fieldNames[0] != "key" {
		t.Fatalf("Error: %v, fieldNames: %v", err, fieldNames)
	}

	fieldValues, err = utils3.GetAllFieldValuesOrValues(a3)
	if err != nil || len(fieldValues) != 1 || fieldValues[0] != "newValue" {
		t.Fatalf("Error: %v, fieldValues: %v", err, fieldValues)
	}

	fieldsAndValues, err = utils3.GetAllFieldsOrKeysAndValues(a3)
	if err != nil || len(fieldsAndValues) != 1 || fieldsAndValues["key"] != "newValue" {
		t.Fatalf("Error: %v, fieldsAndValues: %v", err, fieldsAndValues)
	}

	// 测试结构体切片
	var a4 = []SampleStruct{{Code: "test3"}}
	utils4, err := NewReflectUtils(a4)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if utils4.TypeCategory() != CategoryStructSliceOrArray {
		t.Fatalf("Expected CategoryStructSliceOrArray, got %v", utils4.TypeCategory())
	}

	elem4, err := utils4.GetElement(0)
	if err != nil || reflect.ValueOf(elem4).FieldByName("Code").String() != "test3" {
		t.Fatalf("Error: %v, elem4: %v", err, elem4)
	}

	fieldValue, err = utils4.GetFieldOrKey(elem4, "Code")
	if err != nil || fieldValue != "test3" {
		t.Fatalf("Error: %v, fieldValue: %v", err, fieldValue)
	}

	err = utils4.SetFieldOrKey(&a4[0], "Code", "newTest3")
	if err != nil || a4[0].Code != "newTest3" {
		t.Fatalf("Error: %v, updated value: %v", err, a4[0].Code)
	}

	fieldNames, err = utils4.GetAllFieldNamesOrKeys(elem4)
	if err != nil || len(fieldNames) != 1 || fieldNames[0] != "Code" {
		t.Fatalf("Error: %v, fieldNames: %v", err, fieldNames)
	}

	fieldValues, err = utils4.GetAllFieldValuesOrValues(elem4)
	if err != nil || len(fieldValues) != 1 || fieldValues[0] != "newTest3" {
		t.Fatalf("Error: %v, fieldValues: %v", err, fieldValues)
	}

	fieldsAndValues, err = utils4.GetAllFieldsOrKeysAndValues(elem4)
	if err != nil || len(fieldsAndValues) != 1 || fieldsAndValues["Code"] != "newTest3" {
		t.Fatalf("Error: %v, fieldsAndValues: %v", err, fieldsAndValues)
	}

	// 测试结构体指针切片
	var a5 = []*SampleStruct{{Code: "test4"}}
	utils5, err := NewReflectUtils(a5)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if utils5.TypeCategory() != CategoryStructPointerSliceOrArray {
		t.Fatalf("Expected CategoryStructPointerSliceOrArray, got %v", utils5.TypeCategory())
	}

	elem5, err := utils5.GetElement(0)
	if err != nil || reflect.ValueOf(elem5).Elem().FieldByName("Code").String() != "test4" {
		t.Fatalf("Error: %v, elem5: %v", err, elem5)
	}

	fieldValue, err = utils5.GetFieldOrKey(elem5, "Code")
	if err != nil || fieldValue != "test4" {
		t.Fatalf("Error: %v, fieldValue: %v", err, fieldValue)
	}

	err = utils5.SetFieldOrKey(a5[0], "Code", "newTest4")
	if err != nil || a5[0].Code != "newTest4" {
		t.Fatalf("Error: %v, updated value: %v", err, a5[0].Code)
	}

	fieldNames, err = utils5.GetAllFieldNamesOrKeys(elem5)
	if err != nil || len(fieldNames) != 1 || fieldNames[0] != "Code" {
		t.Fatalf("Error: %v, fieldNames: %v", err, fieldNames)
	}

	fieldValues, err = utils5.GetAllFieldValuesOrValues(elem5)
	if err != nil || len(fieldValues) != 1 || fieldValues[0] != "newTest4" {
		t.Fatalf("Error: %v, fieldValues: %v", err, fieldValues)
	}

	fieldsAndValues, err = utils5.GetAllFieldsOrKeysAndValues(elem5)
	if err != nil || len(fieldsAndValues) != 1 || fieldsAndValues["Code"] != "newTest4" {
		t.Fatalf("Error: %v, fieldsAndValues: %v", err, fieldsAndValues)
	}

	// 测试 map[string]any 指针切片
	var a6 = []*map[string]any{{"key": "value"}}
	utils6, err := NewReflectUtils(a6)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if utils6.TypeCategory() != CategoryMapStringAnyPointerSliceOrArray {
		t.Fatalf("Expected CategoryMapStringAnyPointerSliceOrArray, got %v", utils6.TypeCategory())
	}

	elem6, err := utils6.GetElement(0)
	if err != nil || (*elem6.(*map[string]any))["key"] != "value" {
		t.Fatalf("Error: %v, elem6: %v", err, elem6)
	}

	keyValue, err = utils6.GetFieldOrKey(elem6, "key")
	if err != nil || keyValue != "value" {
		t.Fatalf("Error: %v, keyValue: %v", err, keyValue)
	}

	err = utils6.SetFieldOrKey(a6[0], "key", "newValue")
	if err != nil || (*a6[0])["key"] != "newValue" {
		t.Fatalf("Error: %v, updated value: %v", err, (*a6[0])["key"])
	}

	fieldNames, err = utils6.GetAllFieldNamesOrKeys(elem6)
	if err != nil || len(fieldNames) != 1 || fieldNames[0] != "key" {
		t.Fatalf("Error: %v, fieldNames: %v", err, fieldNames)
	}

	fieldValues, err = utils6.GetAllFieldValuesOrValues(elem6)
	if err != nil || len(fieldValues) != 1 || fieldValues[0] != "newValue" {
		t.Fatalf("Error: %v, fieldValues: %v", err, fieldValues)
	}

	fieldsAndValues, err = utils6.GetAllFieldsOrKeysAndValues(elem6)
	if err != nil || len(fieldsAndValues) != 1 || fieldsAndValues["key"] != "newValue" {
		t.Fatalf("Error: %v, fieldsAndValues: %v", err, fieldsAndValues)
	}
}
