package dba

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type postgresAdapter struct {
}

func (m *postgresAdapter) Name() string {
	return "postgres"
}

func (m *postgresAdapter) Connect(config *ConnectConfig) (*sqlx.DB, error) {
	return sqlx.Connect(m.Name(), config.Dsn)
}

func (m *postgresAdapter) GenDDL(sortedNames []string, schemas map[string]*Schema, ignoreComments ...bool) string {
	//TODO implement me
	panic("implement me")
}

func (m *postgresAdapter) CreateClauses() string {
	//TODO implement me
	panic("implement me")
}

func (m *postgresAdapter) DeleteClauses() string {
	//TODO implement me
	panic("implement me")
}

func (m *postgresAdapter) UpdateClauses() string {
	//TODO implement me
	panic("implement me")
}

func (m *postgresAdapter) QueryClauses() string {
	//TODO implement me
	panic("implement me")
}
