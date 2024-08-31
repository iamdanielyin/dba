package dba

import "sync"

var DefaultNamespace = &Namespace{
	connections: new(sync.Map),
	schemas:     new(sync.Map),
}

func Connect(config *ConnectConfig) error {
	_, err := DefaultNamespace.Connect(config)
	return err
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

func LookupSchema(name ...string) map[string]*Schema {
	return DefaultNamespace.LookupSchema(name...)
}

func Model(schemaName string) *DataModel {
	return DefaultNamespace.Model(schemaName)
}

func ModelBy(connectionName, schemaName string) *DataModel {
	return DefaultNamespace.ModelBy(connectionName, schemaName)
}

func Exec(query string, args ...any) (int, error) {
	return ExecBy("", query, args)
}
func BatchExec(query string, args ...any) (int, error) {
	return BatchExecBy("", query, args)
}

func Query(dst any, query string, args ...any) error {
	return QueryBy("", dst, query, args)
}

func ExecBy(connectionName string, query string, args ...any) (int, error) {
	sess := DefaultNamespace.Session(connectionName)
	return sess.Exec(query, args...)
}
func BatchExecBy(connectionName string, query string, args ...any) (int, error) {
	sess := DefaultNamespace.Session(connectionName)
	return sess.BatchExec(query, args...)
}

func QueryBy(connectionName string, dst any, query string, args ...any) error {
	sess := DefaultNamespace.Session(connectionName)
	return sess.Query(dst, query, args...)
}
