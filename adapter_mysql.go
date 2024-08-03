package dba

import (
	"bytes"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"strings"
)

type mysqlAdapter struct {
}

func (m *mysqlAdapter) Name() string {
	return "mysql"
}

func (m *mysqlAdapter) Connect(config *ConnectConfig) (*sqlx.DB, error) {
	return sqlx.Connect(m.Name(), config.Dsn)
}

func (m *mysqlAdapter) GenDDL(sortedNames []string, schemas map[string]*Schema, ignoreComments ...bool) string {
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

func (m *mysqlAdapter) CreateClauses() string {
	return `INSERT INTO {{.TableName}} ({{.Columns}})
			VALUES
			{{.Rows}}
			{{if .OnConflict}}
			ON DUPLICATE KEY UPDATE {{.OnConflictUpdate}}
			{{end}}`
}

func (m *mysqlAdapter) DeleteClauses() string {
	//TODO implement me
	panic("implement me")
	//deleteClauses = []string{"DELETE", "FROM", "WHERE"}
}

func (m *mysqlAdapter) UpdateClauses() string {
	//TODO implement me
	panic("implement me")
	//updateClauses = []string{"UPDATE", "SET", "WHERE"}
}

func (m *mysqlAdapter) QueryClauses() string {
	//TODO implement me
	panic("implement me")
	//queryClauses  = []string{"SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR"}
}
