package dba

import "github.com/jmoiron/sqlx"

var drivers = map[string]Driver{
	"mysql":    &mysqlDriver{},
	"postgres": &postgresDriver{},
	"sqlite":   &sqliteDriver{},
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
