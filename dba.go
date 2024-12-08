package dba

import "sync"

var DefaultNamespace = &Namespace{
	connections: new(sync.Map),
	schemas:     new(sync.Map),
}

func Connect(config *ConnectConfig) (*Connection, error) {
	return DefaultNamespace.Connect(config)
}

func Session(name ...string) *Connection {
	return DefaultNamespace.Session(name...)
}

func ConnectionNames() []string {
	return DefaultNamespace.ConnectionNames()
}

func Disconnect(name ...string) {
	DefaultNamespace.Disconnect(name...)
}

func DisconnectAll() {
	DefaultNamespace.DisconnectAll()
}

func RegisterSchema(values ...any) error {
	return DefaultNamespace.RegisterSchema(values...)
}

func SchemaBy(name string) *Schema {
	return DefaultNamespace.SchemaBy(name)
}

func SchemaBys(name ...string) map[string]*Schema {
	return DefaultNamespace.SchemaBys(name...)
}

func Model(schemaName string, options ...*ModelOptions) *DataModel {
	return DefaultNamespace.Model(schemaName, options...)
}

func Exec(query string, args ...any) (int, error) {
	return ExecBy("", query, args)
}

func ExecBatch(query string, args ...any) (int, error) {
	return ExecByBatch("", query, args)
}

func ExecBy(connectionName string, query string, args ...any) (int, error) {
	sess := DefaultNamespace.Session(connectionName)
	return sess.Exec(query, args...)
}

func ExecByBatch(connectionName string, query string, args ...any) (int, error) {
	sess := DefaultNamespace.Session(connectionName)
	return sess.BatchExec(query, args...)
}

func Query(dst any, query string, args ...any) error {
	return QueryBy("", dst, query, args)
}

func QueryBy(connectionName string, dst any, query string, args ...any) error {
	sess := DefaultNamespace.Session(connectionName)
	return sess.Query(dst, query, args...)
}
