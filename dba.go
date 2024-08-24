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

func LookupSchema(name string) *Schema {
	return DefaultNamespace.LookupSchema(name)
}

func Schemas(name ...string) map[string]*Schema {
	return DefaultNamespace.Schemas(name...)
}

func Model(schemaName string) *DataModel {
	return DefaultNamespace.Model(schemaName)
}

func ModelBySession(connectionName, schemaName string) *DataModel {
	return DefaultNamespace.ModelBySession(connectionName, schemaName)
}
