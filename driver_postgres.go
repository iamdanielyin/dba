package dba

import (
	"bytes"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"strings"
)

type postgresDriver struct {
}

func (m *postgresDriver) Name() string {
	return "postgres"
}

func (m *postgresDriver) Connect(config *ConnectConfig) (*sqlx.DB, error) {
	return sqlx.Connect(m.Name(), config.Dsn)
}

func (m *postgresDriver) GenDDL(sortedNames []string, schs map[string]*Schema, ignoreComments ...bool) string {
	var ddls []string
	for _, name := range sortedNames {
		sch := schs[name]
		var (
			columns        []string
			primaryColumns []string
		)
		for _, field := range sch.Fields {
			var buffer bytes.Buffer
			buffer.WriteString(fmt.Sprintf(`"%s"	`, field.NativeName))
			nativeType := strings.TrimSpace(field.NativeType)
			if nativeType == "" {
				switch field.Type {
				case String:
					nativeType = "TEXT"
				case Integer:
					if field.IsAutoIncrement {
						nativeType = "BIGSERIAL"
					} else {
						nativeType = "BIGINT"
					}
				case Float:
					nativeType = "DOUBLE PRECISION"
				case Boolean:
					nativeType = "BOOLEAN"
				case Time:
					nativeType = "TIMESTAMP WITH TIME ZONE"
				}
			}
			if nativeType == "" {
				continue
			}
			buffer.WriteString(nativeType)
			if field.IsRequired() || field.IsPrimary {
				buffer.WriteString(" NOT NULL")
			} else {
				buffer.WriteString(" NULL")
			}
			if field.IsPrimary {
				primaryColumns = append(primaryColumns, fmt.Sprintf(`"%s"`, field.NativeName))
			}
			if field.Title != "" {
				buffer.WriteString(fmt.Sprintf(` COMMENT '%s'`, field.Title))
			}
			columns = append(columns, buffer.String())
		}
		if len(columns) == 0 {
			continue
		}
		if len(primaryColumns) > 0 {
			columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryColumns, ",")))
		}
		var buffer bytes.Buffer
		if len(ignoreComments) > 0 && !ignoreComments[0] {
			buffer.WriteString(fmt.Sprintf("-- create \"%s\" table\n", sch.NativeName))
		}
		buffer.WriteString(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s);`, sch.NativeName, strings.Join(columns, ",\n")))
		ddls = append(ddls, buffer.String())
	}
	return strings.Join(ddls, "\n\n")
}

func (m *postgresDriver) CreateClauses() string {
	return `INSERT INTO {{.Table}} ({{.Columns}})
			VALUES
			{{.Rows}}
			{{if .ConflictUpdates}}
			ON CONFLICT {{if .ConflictColumns}}({{.ConflictColumns}}){{end}} DO UPDATE 
			SET {{.ConflictUpdates}}
			{{else if eq .ConflictKind "IGNORE"}}
			ON CONFLICT {{if .ConflictColumns}}({{.ConflictColumns}}){{end}} DO NOTHING
			{{end}}`
}

func (m *postgresDriver) DeleteClauses() string {
	return `DELETE FROM {{.Table}}
			WHERE {{.Where}}`
}

func (m *postgresDriver) UpdateClauses() string {
	return `UPDATE {{.Table}}
			SET {{.Sets}}
			WHERE {{.Where}}`
}

func (m *postgresDriver) QueryClauses() string {
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
			LIMIT {{.Limit}}
			{{if .Offset}}OFFSET {{.Offset}}{{end}}
			{{end}}`
}
