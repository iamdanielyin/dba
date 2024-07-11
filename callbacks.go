package dba

import (
	"fmt"
	"gorm.io/gorm"
	"reflect"
)

func registerCallbacks(db *gorm.DB) {
	_ = db.Callback().Query().After("gorm:query").Register("dba:preload", preloadCallback)
}

func preloadCallback(db *gorm.DB) {
	var r *Result
	if v, ok := db.InstanceGet("DBA_RESULT"); ok {
		r = v.(*Result)
	}

	if r == nil || len(r.preload) == 0 {
		return
	}

	var dst any
	if v, ok := db.InstanceGet("DBA_DST"); ok {
		dst = v
	}

	loadRelationshipsBatch(db, dst, r.dm.schema, r.preload)
}

// Load all relationships for a given struct or slice of structs
func loadRelationshipsBatch(db *gorm.DB, value any, s *Schema, preloads map[string]*PreloadOptions) {
	reflectValue := reflect.ValueOf(value)
	reflectType := reflectValue.Type()

	if reflectType.Kind() == reflect.Ptr && reflectType.Elem().Kind() == reflect.Slice {
		reflectValue = reflectValue.Elem()
		reflectType = reflectValue.Type()
	}

	if reflectType.Kind() == reflect.Slice {
		for key, _ := range preloads {
			field := s.Fields[key]
			if !field.Valid() || field.IsScalarType() || field.Relationship == nil {
				continue
			}
			if field.Relationship != nil {
				switch field.Relationship.Kind {
				case HasOne:
					loadHasOneBatch(db, reflectValue, field)
				case HasMany:
					loadHasManyBatch(db, reflectValue, field)
				case RefOne:
					loadRefOneBatch(db, reflectValue, field)
				case RefMany:
					loadRefManyBatch(db, reflectValue, field)
				}
			}
		}
	}
}

func loadHasOneBatch(db *gorm.DB, reflectValue reflect.Value, field *Field) {
	srcIDs := getSrcIDs(reflectValue, field.Relationship.SrcSchemaField)

	dstMap := make(map[uint64]interface{})
	dstType := reflect.New(reflect.ValueOf(reflectValue.Index(0).Interface()).FieldByName(field.Name).Type().Elem()).Elem().Type()
	dstSlice := reflect.New(reflect.SliceOf(dstType)).Interface()
	db.Where(fmt.Sprintf("%s IN ?", field.Relationship.DstSchemaField), srcIDs).Find(dstSlice)

	dstValue := reflect.ValueOf(dstSlice).Elem()
	for i := 0; i < dstValue.Len(); i++ {
		dstElem := dstValue.Index(i).Interface()
		dstID := reflect.Indirect(reflect.ValueOf(dstElem)).FieldByName(field.Relationship.DstSchemaField).Uint()
		dstMap[dstID] = dstElem
	}

	for i := 0; i < reflectValue.Len(); i++ {
		srcID := reflect.Indirect(reflectValue.Index(i).FieldByName(field.Relationship.SrcSchemaField)).Uint()
		if dstElem, found := dstMap[srcID]; found {
			reflectValue.Index(i).FieldByName(field.Name).Set(reflect.ValueOf(dstElem))
		}
	}
}

func loadHasManyBatch(db *gorm.DB, reflectValue reflect.Value, field *Field) {
	srcIDs := getSrcIDs(reflectValue, field.Relationship.SrcSchemaField)

	dstMap := make(map[uint64][]interface{})
	dstType := reflect.New(reflect.ValueOf(reflectValue.Index(0).Interface()).FieldByName(field.Name).Type().Elem().Elem()).Elem().Type()
	dstSlice := reflect.New(reflect.SliceOf(dstType)).Interface()
	db.Where(fmt.Sprintf("%s IN ?", field.Relationship.DstSchemaField), srcIDs).Find(dstSlice)

	dstValue := reflect.ValueOf(dstSlice).Elem()
	for i := 0; i < dstValue.Len(); i++ {
		dstElem := dstValue.Index(i).Interface()
		dstID := reflect.Indirect(reflect.ValueOf(dstElem)).FieldByName(field.Relationship.DstSchemaField).Uint()
		dstMap[dstID] = append(dstMap[dstID], dstElem)
	}

	for i := 0; i < reflectValue.Len(); i++ {
		srcID := reflect.Indirect(reflectValue.Index(i).FieldByName(field.Relationship.SrcSchemaField)).Uint()
		if dstElems, found := dstMap[srcID]; found {
			sliceValue := reflect.MakeSlice(reflect.ValueOf(reflectValue.Index(i).Interface()).FieldByName(field.Name).Type(), len(dstElems), len(dstElems))
			for j, dstElem := range dstElems {
				sliceValue.Index(j).Set(reflect.ValueOf(dstElem))
			}
			reflectValue.Index(i).FieldByName(field.Name).Set(sliceValue)
		}
	}
}

func loadRefOneBatch(db *gorm.DB, reflectValue reflect.Value, field *Field) {
	srcIDs := getSrcIDs(reflectValue, field.Relationship.SrcSchemaField)

	dstMap := make(map[uint64]interface{})
	dstType := reflect.New(reflect.ValueOf(reflectValue.Index(0).Interface()).FieldByName(field.Name).Type().Elem()).Elem().Type()
	dstSlice := reflect.New(reflect.SliceOf(dstType)).Interface()
	db.Where(fmt.Sprintf("%s IN ?", field.Relationship.DstSchemaField), srcIDs).Find(dstSlice)

	dstValue := reflect.ValueOf(dstSlice).Elem()
	for i := 0; i < dstValue.Len(); i++ {
		dstElem := dstValue.Index(i).Interface()
		dstID := reflect.Indirect(reflect.ValueOf(dstElem)).FieldByName(field.Relationship.DstSchemaField).Uint()
		dstMap[dstID] = dstElem
	}

	for i := 0; i < reflectValue.Len(); i++ {
		srcID := reflect.Indirect(reflectValue.Index(i).FieldByName(field.Relationship.SrcSchemaField)).Uint()
		if dstElem, found := dstMap[srcID]; found {
			reflectValue.Index(i).FieldByName(field.Name).Set(reflect.ValueOf(dstElem))
		}
	}
}

func loadRefManyBatch(db *gorm.DB, reflectValue reflect.Value, field *Field) {
	srcIDs := getSrcIDs(reflectValue, field.Relationship.SrcSchemaField)

	dstMap := make(map[uint64][]interface{})
	dstType := reflect.New(reflect.ValueOf(reflectValue.Index(0).Interface()).FieldByName(field.Name).Type().Elem().Elem()).Elem().Type()
	dstSlice := reflect.New(reflect.SliceOf(dstType)).Interface()

	// 查询桥表数据
	var brgResults []map[string]interface{}
	db.Table(field.Relationship.BrgSchemaName).Where(fmt.Sprintf("%s IN ?", field.Relationship.BrgSchemaSrcField), srcIDs).Find(&brgResults)

	var dstIDs []uint64
	for _, result := range brgResults {
		dstID := result[field.Relationship.BrgSchemaDstField].(uint64)
		dstIDs = append(dstIDs, dstID)
	}

	// 查询目标表数据
	db.Where(fmt.Sprintf("%s IN ?", field.Relationship.DstSchemaField), dstIDs).Find(dstSlice)

	dstValue := reflect.ValueOf(dstSlice).Elem()
	for i := 0; i < dstValue.Len(); i++ {
		dstElem := dstValue.Index(i).Interface()
		dstID := reflect.Indirect(reflect.ValueOf(dstElem)).FieldByName(field.Relationship.DstSchemaField).Uint()
		dstMap[dstID] = append(dstMap[dstID], dstElem)
	}

	for i := 0; i < reflectValue.Len(); i++ {
		srcID := reflect.Indirect(reflectValue.Index(i).FieldByName(field.Relationship.SrcSchemaField)).Uint()
		var relatedItems []interface{}
		for _, result := range brgResults {
			if result[field.Relationship.BrgSchemaSrcField].(uint64) == srcID {
				dstID := result[field.Relationship.BrgSchemaDstField].(uint64)
				if dstElems, found := dstMap[dstID]; found {
					relatedItems = append(relatedItems, dstElems...)
				}
			}
		}
		if len(relatedItems) > 0 {
			sliceValue := reflect.MakeSlice(reflect.ValueOf(reflectValue.Index(i).Interface()).FieldByName(field.Name).Type(), len(relatedItems), len(relatedItems))
			for j, item := range relatedItems {
				sliceValue.Index(j).Set(reflect.ValueOf(item))
			}
			reflectValue.Index(i).FieldByName(field.Name).Set(sliceValue)
		}
	}
}

func getSrcIDs(reflectValue reflect.Value, fieldName string) []uint64 {
	srcIDs := make([]uint64, reflectValue.Len())
	for i := 0; i < reflectValue.Len(); i++ {
		srcIDs[i] = reflect.Indirect(reflectValue.Index(i).FieldByName(fieldName)).Uint()
	}
	return srcIDs
}
