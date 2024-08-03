package dba

import "github.com/jmoiron/sqlx"

var adapters = map[string]Adapter{
	"mysql":    &mysqlAdapter{},
	"postgres": &postgresAdapter{},
	"sqlite":   &sqliteAdapter{},
}

type Adapter interface {
	Name() string
	Connect(config *ConnectConfig) (*sqlx.DB, error)
	GenDDL(sortedNames []string, schemas map[string]*Schema, ignoreComments ...bool) string
	CreateClauses() string
	DeleteClauses() string
	UpdateClauses() string
	QueryClauses() string
}
