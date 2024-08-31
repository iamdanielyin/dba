package dba

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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
	//TODO implement me
	panic("implement me")
}

func (m *postgresDriver) CreateClauses() string {
	//TODO implement me
	panic("implement me")
}

func (m *postgresDriver) DeleteClauses() string {
	//TODO implement me
	panic("implement me")
}

func (m *postgresDriver) UpdateClauses() string {
	//TODO implement me
	panic("implement me")
}

func (m *postgresDriver) QueryClauses() string {
	//TODO implement me
	panic("implement me")
}
