package dba

func Connect(drv Driver, dsn string, name ...string) error {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	_, err := DefaultNamespace.Connect(n, drv, dsn)
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
