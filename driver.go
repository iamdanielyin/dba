package dba

import "github.com/jmoiron/sqlx"

const (
	MySQL      = "mysql"
	PostgreSQL = "postgres"
	SQLite     = "sqlite"
)

var drivers = map[string]Driver{
	MySQL:      &mysqlDriver{},
	PostgreSQL: &postgresDriver{},
	SQLite:     &sqliteDriver{},
}

type Driver interface {
	Name() string
	Connect(config *ConnectConfig) (*sqlx.DB, error)
	GenDDL(sortedNames []string, schemas map[string]*Schema, ignoreComments ...bool) string
	CreateClauses() string
	DeleteClauses() string
	UpdateClauses() string
	QueryClauses() string
}
