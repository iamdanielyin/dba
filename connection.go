package dba

import (
	"bytes"
	"fmt"
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
	MYSQL Driver = "MYSQL"
)

type Connection struct {
	driver Driver
	dsn    string
	name   string
	gdb    *gorm.DB
}

func (c *Connection) GenDDL(schemas map[string]*Schema) string {
	var sortedNames []string
	for name := range schemas {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

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
					nativeType = "VARCHAR(255)"
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
		buffer.WriteString(fmt.Sprintf("-- create \"%s\" table\n", schema.NativeName))
		buffer.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n);", schema.NativeName, strings.Join(columns, ",\n")))
		ddls = append(ddls, buffer.String())
	}
	return strings.Join(ddls, "\n\n")
}
