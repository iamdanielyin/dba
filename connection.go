package dba

import (
	"github.com/jmoiron/sqlx"
	"sort"
	"sync"
)

var DefaultNamespace = &Namespace{
	connections: new(sync.Map),
	schemas:     new(sync.Map),
}

type Connection struct {
	ns      *Namespace
	adapter Adapter
	dsn     string
	name    string
	xdb     *sqlx.DB
}

func (c *Connection) Init(schemas ...map[string]*Schema) error {
	var ss map[string]*Schema
	if len(schemas) > 0 {
		ss = schemas[0]
	}
	if ss == nil {
		ss = c.ns.Schemas()
	}
	ddl := c.GenDDL(ss)
	if ddl != "" {
		_, err := c.Exec(ddl)
		return err
	}
	return nil
}

func (c *Connection) GenDDL(schemas map[string]*Schema, ignoreComments ...bool) string {
	var sortedNames []string
	for name := range schemas {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)
	return c.adapter.GenDDL(sortedNames, schemas, ignoreComments...)
}

func (c *Connection) Exec(sql string, values ...any) (int, error) {
	r, err := c.xdb.Exec(sql, values...)
	if err != nil {
		return 0, err
	}
	var n int64
	n, err = r.RowsAffected()
	return int(n), err
}
