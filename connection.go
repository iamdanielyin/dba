package dba

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Connection struct {
	ns             *Namespace
	driver         Driver
	dsn            string
	name           string
	xdb            *sqlx.DB
	logger         *logrus.Logger
	CreateTemplate *template.Template
	UpdateTemplate *template.Template
	DeleteTemplate *template.Template
	QueryTemplate  *template.Template
}

func (c *Connection) Name() string {
	return c.name
}

func (c *Connection) Driver() string {
	return c.driver.Name()
}

func (c *Connection) DSN() string {
	return c.dsn
}

func (c *Connection) Init(schs ...map[string]*Schema) error {
	var ss map[string]*Schema
	if len(schs) > 0 {
		ss = schs[0]
	}
	if ss == nil {
		ss = c.ns.SchemaBys()
	}
	ddl := c.GenDDL(ss)
	if ddl != "" {
		_, err := c.BatchExec(ddl)
		return err
	}
	return nil
}
func (c *Connection) Namespace() *Namespace {
	return c.ns
}

func (c *Connection) GenDDL(schs map[string]*Schema, ignoreComments ...bool) string {
	var sortedNames []string
	for name := range schs {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)
	return c.driver.GenDDL(sortedNames, schs, ignoreComments...)
}

func (c *Connection) Query(dst any, query string, args ...any) error {
	query = formatSQL(query)
	return autoScan(dst, c.xdb, query, args)
}

func (c *Connection) Exec(query string, args ...any) (int, error) {
	query = formatSQL(query)
	r, err := c.xdb.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	var n int64
	n, err = r.RowsAffected()
	return int(n), err
}

func (c *Connection) BatchExec(query string, args ...any) ([]int, error) {
	query = formatSQL(query)
	tx, err := c.xdb.Begin()

	var results []int
	if err != nil {
		return results, err
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
				return results, err
			}

			res, err = tx.Exec(stmt, stmtParams...)
			if err != nil {
				return results, err
			}
			paramIndex = newIndex
			n, _ := res.RowsAffected()
			results = append(results, int(n))
		}
	}
	return results, nil
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

func formatSQL(sql string) string {
	// 正则表达式匹配所有字符串字面量（包括单引号和双引号）
	re := regexp.MustCompile(`'[^']*'|"[^"]*"`)

	// 用于存储被提取的字符串字面量
	stringLiterals := re.FindAllString(sql, -1)

	// 替换字符串字面量为占位符
	placeholderSQL := re.ReplaceAllString(sql, "##STRING_LITERAL##")

	// 对占位符后的 SQL 进行标准化（去除多余空白）
	standardizedSQL := regexp.MustCompile(`\s+`).ReplaceAllString(placeholderSQL, " ")

	// 依次将原本的字符串字面量替换回去
	for _, literal := range stringLiterals {
		standardizedSQL = strings.Replace(standardizedSQL, "##STRING_LITERAL##", literal, 1)
	}

	// 去除前后的空白字符
	return strings.TrimSpace(standardizedSQL)
}
