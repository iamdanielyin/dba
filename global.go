package dba

func Connect(drv Driver, dsn string, name ...string) error {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	_, err := DefaultNamespace.Connect(n, drv, dsn)
	return err
}

func LookupConnection(name string) *Connection {
	return DefaultNamespace.LookupConnection(name)
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

func RegisterSchemas(values ...any) error {
	return DefaultNamespace.RegisterSchemas(values...)
}

func LookupSchema(name string) *Schema {
	return DefaultNamespace.LookupSchema(name)
}

func Schemas() map[string]*Schema {
	return DefaultNamespace.Schemas()
}
