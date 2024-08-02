package dba

import (
	"bytes"
	"fmt"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
	"sort"
	"strings"
	"sync"
)

var DefaultNamespace = &Namespace{
	connections: new(sync.Map),
	schemas:     new(sync.Map),
}

type Driver string

const (
	MYSQL    Driver = "MYSQL"
	SQLITE   Driver = "SQLITE"
	POSTGRES Driver = "POSTGRES"
)

type Connection struct {
	ns     *Namespace
	driver Driver
	dsn    string
	name   string
	xdb    *sqlx.DB
}

func (c *Connection) AutoMigrate(values ...any) error {
	return c.NewDB().AutoMigrate(values...)
}

func (c *Connection) NewDB() *gorm.DB {
	return c.gdb.Session(&gorm.Session{
		NewDB: true,
	})
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

func (c *Connection) genDDLWithMySQL(sortedNames []string, schemas map[string]*Schema, ignoreComments ...bool) string {
	var ddls []string
	for _, name := range sortedNames {
		schema := schemas[name]
		var (
			columns        []string
			primaryColumns []string
		)
		for _, field := range schema.Fields {
			var buffer bytes.Buffer
			buffer.WriteString(fmt.Sprintf("`%s`\t", field.NativeName))
			nativeType := strings.TrimSpace(field.NativeType)
			if nativeType == "" {
				switch field.Type {
				case String:
					nativeType = "TEXT"
				case Integer:
					nativeType = "BIGINT"
				case Float:
					nativeType = "DOUBLE"
				case Boolean:
					nativeType = "TINYINT(1)"
				case Time:
					nativeType = "DATETIME(3)"
				}
			}
			if nativeType == "" {
				continue
			}
			buffer.WriteString(nativeType)
			if field.IsUnsigned {
				buffer.WriteString(" UNSIGNED")
			}
			if field.IsRequired {
				buffer.WriteString(" NOT NULL")
			} else {
				buffer.WriteString(" NULL")
			}
			if field.IsAutoIncrement {
				buffer.WriteString(" AUTO_INCREMENT")
			}
			if field.IsPrimary {
				primaryColumns = append(primaryColumns, fmt.Sprintf("`%s`", field.NativeName))
			}
			if field.Title != "" {
				buffer.WriteString(fmt.Sprintf(" COMMENT '%s'", field.Title))
			}
			columns = append(columns, buffer.String())
		}
		if len(columns) == 0 {
			continue
		}
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryColumns, ",")))
		var buffer bytes.Buffer
		if len(ignoreComments) > 0 && !ignoreComments[0] {
			buffer.WriteString(fmt.Sprintf("-- create \"%s\" table\n", schema.NativeName))
		}
		buffer.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n);", schema.NativeName, strings.Join(columns, ",\n")))
		ddls = append(ddls, buffer.String())
	}
	return strings.Join(ddls, "\n\n")
}

func (c *Connection) genDDLWithSQLite(sortedNames []string, schemas map[string]*Schema, ignoreComments ...bool) string {
	var ddls []string
	for _, name := range sortedNames {
		schema := schemas[name]
		var (
			columns        []string
			primaryColumns []string
		)
		for _, field := range schema.Fields {
			var buffer bytes.Buffer
			buffer.WriteString(fmt.Sprintf("`%s`\t", field.NativeName))
			nativeType := strings.TrimSpace(field.NativeType)
			if nativeType == "" {
				switch field.Type {
				case String:
					nativeType = "TEXT"
				case Integer:
					nativeType = "INTEGER"
				case Float:
					nativeType = "REAL"
				case Boolean:
					nativeType = "INTEGER"
				case Time:
					nativeType = "DATETIME"
				}
			}
			if nativeType == "" {
				continue
			}
			buffer.WriteString(nativeType)
			if field.IsPrimary {
				if field.IsPrimary {
					primaryColumns = append(primaryColumns, field.NativeName)
				}
			} else {
				if field.IsRequired {
					buffer.WriteString(" NOT NULL")
				} else {
					buffer.WriteString(" NULL")
				}
			}
			columns = append(columns, buffer.String())
		}
		if len(columns) == 0 {
			continue
		}
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryColumns, ",")))
		var buffer bytes.Buffer
		if len(ignoreComments) > 0 && !ignoreComments[0] {
			buffer.WriteString(fmt.Sprintf("-- create \"%s\" table\n", schema.NativeName))
		}
		buffer.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n);", schema.NativeName, strings.Join(columns, ",\n")))
		ddls = append(ddls, buffer.String())
	}
	return strings.Join(ddls, "\n\n")
}

func (c *Connection) GenDDL(schemas map[string]*Schema, ignoreComments ...bool) string {
	var sortedNames []string
	for name := range schemas {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)
	var ddl string
	switch c.driver {
	case MYSQL:
		ddl = c.genDDLWithMySQL(sortedNames, schemas, ignoreComments...)
	case SQLITE:
		ddl = c.genDDLWithSQLite(sortedNames, schemas, ignoreComments...)
	}
	return ddl
}

func (c *Connection) Exec(sql string, values ...any) (int, error) {
	gdb := c.NewDB().Exec(sql, values...)
	return int(gdb.RowsAffected), gdb.Error
}
