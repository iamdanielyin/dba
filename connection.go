package dba

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

type Connection struct {
	ns             *Namespace
	driver         Driver
	dsn            string
	name           string
	xdb            *sqlx.DB
	CreateTemplate *template.Template
	UpdateTemplate *template.Template
	DeleteTemplate *template.Template
	QueryTemplate  *template.Template
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
		_, err := c.BatchExec(ddl)
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
	return c.driver.GenDDL(sortedNames, schemas, ignoreComments...)
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

func (c *Connection) BatchExec(query string, args ...any) (int, error) {
	tx, err := c.xdb.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			err = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	var res sql.Result
	sqlStatements := strings.Split(query, ";")
	paramIndex := 0
	for _, stmt := range sqlStatements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			stmtParams, newIndex, err := extractParams(stmt, args, paramIndex)
			if err != nil {
				return 0, err
			}

			res, err = tx.Exec(stmt, stmtParams...)
			if err != nil {
				return 0, err
			}
			paramIndex = newIndex
		}
	}
	if res != nil {
		n, _ := res.RowsAffected()
		return int(n), err
	}
	return 0, nil
}

func extractParams(stmt string, params []any, paramIndex int) ([]any, int, error) {
	placeholderPattern := regexp.MustCompile(`\?`)
	placeholders := placeholderPattern.FindAllStringIndex(stmt, -1)
	numPlaceholders := len(placeholders)

	// 检查参数是否足够
	if paramIndex+numPlaceholders > len(params) {
		return nil, paramIndex, fmt.Errorf("insufficient parameters for statement: %s", stmt)
	}

	// 提取匹配的参数
	stmtParams := params[paramIndex : paramIndex+numPlaceholders]
	return stmtParams, paramIndex + numPlaceholders, nil
}
