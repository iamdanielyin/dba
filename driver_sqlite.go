package dba

import (
	"bytes"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"strings"
)

type sqliteDriver struct {
}

func (m *sqliteDriver) Name() string {
	return "sqlite"
}

func (m *sqliteDriver) Connect(config *ConnectConfig) (*sqlx.DB, error) {
	return sqlx.Connect(m.Name(), config.Dsn)
}

func (m *sqliteDriver) GenDDL(sortedNames []string, schs map[string]*Schema, ignoreComments ...bool) string {
	var ddls []string
	for _, name := range sortedNames {
		sch := schs[name]
		var (
			columns        []string
			primaryColumns []string
		)
		for _, field := range sch.Fields {
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
			buffer.WriteString(fmt.Sprintf("-- create \"%s\" table\n", sch.NativeName))
		}
		buffer.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n);", sch.NativeName, strings.Join(columns, ",\n")))
		ddls = append(ddls, buffer.String())
	}
	return strings.Join(ddls, "\n\n")
}

func (m *sqliteDriver) CreateClauses() string {
	return `INSERT {{if eq .ConflictKind "IGNORE"}}OR IGNORE {{end}}INTO {{.Table}} ({{.Columns}})
			VALUES
			{{.Rows}}
			{{if .ConflictUpdates}}
			ON CONFLICT {{if .ConflictColumns}}({{.ConflictColumns}}){{end}} DO UPDATE 
			SET {{.ConflictUpdates}}
			{{end}}`
}

func (m *sqliteDriver) DeleteClauses() string {
	return `DELETE FROM {{.Table}}
			WHERE {{.Where}}`
}

func (m *sqliteDriver) UpdateClauses() string {
	return `UPDATE {{.Table}}
			SET {{.Sets}}
			WHERE {{.Where}}`
}

func (m *sqliteDriver) QueryClauses() string {
	return `SELECT {{if .Columns}}{{.Columns}}{{else}}*{{end}}
			FROM {{.Table}}
			{{if .Where}}
			WHERE {{.Where}}
			{{end}}
			{{if .GroupBys}}
			GROUP BY {{.GroupBys}}
			{{end}}
			{{if .Having}}
			HAVING {{.Having}}
			{{end}}
			{{if .OrderBys}}
			ORDER BY {{.OrderBys}}
			{{end}}
			{{if .Limit}}
			LIMIT {{.Limit}}{{if .Offset}} OFFSET {{.Offset}}{{end}}
			{{end}}`
}
