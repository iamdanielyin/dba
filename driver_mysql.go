package dba

import (
	"bytes"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"strings"
)

type mysqlDriver struct {
}

func (m *mysqlDriver) Name() string {
	return "mysql"
}

func (m *mysqlDriver) Connect(config *ConnectConfig) (*sqlx.DB, error) {
	return sqlx.Connect(m.Name(), config.Dsn)
}

func (m *mysqlDriver) GenDDL(sortedNames []string, schemas map[string]*Schema, ignoreComments ...bool) string {
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

func (m *mysqlDriver) CreateClauses() string {
	//INSERT INTO table_name (column1, column2, column3, ...)
	//VALUES
	//(value1a, value2a, value3a, ...),
	//(value1b, value2b, value3b, ...),
	//(value1c, value2c, value3c, ...)
	//ON DUPLICATE KEY UPDATE column1 = VALUES(column1), column2 = VALUES(column2), ...;

	return `INSERT {{if eq .ConflictResolution "IGNORE"}}IGNORE {{end}}INTO {{.TableName}} ({{.Columns}})
			VALUES
			{{.Rows}}
			{{if .ConflictUpdateValues}}
			ON DUPLICATE KEY UPDATE {{.ConflictUpdateValues}}
			{{end}}`
}

func (m *mysqlDriver) DeleteClauses() string {
	//DELETE FROM students
	//WHERE student_id = 1;
	return `DELETE FROM {{.TableName}}
			WHERE {{.Conditions}}`
}

func (m *mysqlDriver) UpdateClauses() string {
	//UPDATE students
	//SET name = 'Alice', age = 21
	//WHERE student_id = 1;
	return `UPDATE {{.TableName}}
			SET {{.Sets}}
			WHERE {{.Conditions}}`
}

func (m *mysqlDriver) QueryClauses() string {
	//SELECT column1, column2, ...
	//FROM table_name
	//WHERE condition
	//GROUP BY column1, column2, ...
	//HAVING condition
	//ORDER BY column1, column2, ...
	//LIMIT offset, count;
	//假设每页显示 N 条记录，要查询第 P 页的数据：
	//offset = (P - 1) * N
	//count = N
	return `SELECT {{if .Columns}}{{.Columns}}{{else}}*{{end}}
			FROM {{.TableName}}
			{{if .Conditions}}
			WHERE {{.Conditions}}
			{{end}
			{{if .GroupBys}}
			GROUP BY {{.GroupBys}}
			{{end}}
			{{if .Having}}
			HAVING {{.Having}}
			{{end}}
			{{if .OrderBys}}
			ORDER BY {{.OrderBys}}
			{{end}}
			{{if .Paginate}}
			LIMIT {{.Paginate.Offset}}, {{.Paginate.Limit}}
			{{end}}`
}
