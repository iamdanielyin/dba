package dba

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"math"
	"sync"
)

func (r *Result) One(dst any) error {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(ONE, dst)
	if err := gdb.First(dst).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (r *Result) All(dst any) error {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(ALL, dst)
	return gdb.Find(dst).Error
}

func (r *Result) Count() (int, error) {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(COUNT)
	var count int64
	err := gdb.Count(&count).Error
	return int(count), err
}

func (r *Result) Paginate(pageNum int, pageSize int, dst any) (totalRecords int, totalPages int, err error) {
	// FINAL
	defer r.reset()

	r.offset = (pageNum - 1) * pageSize
	r.limit = pageSize
	if err = r.All(dst); err != nil {
		return
	}

	gdb := r.dm.gdb
	var count int64
	if err = gdb.Limit(-1).Offset(-1).Count(&count).Error; err == nil {
		totalRecords = int(count)
		totalPages = int(math.Ceil(float64(count) / float64(pageSize)))
	}
	return
}

func (r *Result) Update(doc any) (int, error) {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(UPDATE)
	values := r.dm.schema.ParseValue(doc, true)
	ret := gdb.Updates(values)
	return int(ret.RowsAffected), ret.Error
}

func (r *Result) Delete() (int, error) {
	// FINAL
	defer r.reset()

	gdb := r.beforeQuery(DELETE)
	ret := gdb.Delete(nil)
	return int(ret.RowsAffected), ret.Error
}

func (r *Result) reset() {
	gdb := r.dm.conn.NewDB()
	gdb = gdb.Table(r.dm.schema.NativeName)
	r.dm.gdb = gdb
	r.filters = nil
	r.orderBys = make(map[string]bool)
	r.limit = 0
	r.offset = 0
	r.cache = new(sync.Map)
	r.preload = make(map[string]*PreloadOptions)
}
