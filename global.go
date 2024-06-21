package dba

func Connect(drv Driver, dsn string, config ...*ConnectConfig) error {
	var c *ConnectConfig
	if len(config) > 0 && config[0] != nil {
		c = config[0]
	} else {
		c = new(ConnectConfig)
	}
	c.Driver = drv
	c.Dsn = dsn
	_, err := DefaultNamespace.Connect(c)
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
