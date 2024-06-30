package dba

import (
	"gorm.io/gorm"
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
	//var dst any
	//if v, ok := db.InstanceGet("DBA_DST"); ok {
	//	dst = v
	//}
	//
	//for key, opts := range r.preload {
	//}
}
