package dba

import (
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	gschema "gorm.io/gorm/schema"
	"log"
	"os"
	"sync"
	"time"
)

type Namespace struct {
	Name        string
	connections *sync.Map
	schemas     *sync.Map
}

type ConnectConfig struct {
	Name        string
	Driver      Driver
	Dsn         string
	ShowSQL     bool
	TablePrefix string
}

func (ns *Namespace) Connect(config *ConnectConfig) (*Connection, error) {
	var dial gorm.Dialector
	switch config.Driver {
	case MYSQL:
		dial = mysql.Open(config.Dsn)
	case SQLITE:
		dial = sqlite.Open(config.Dsn)
	default:
		return nil, errors.Errorf("dba: invalid driver: %s", config.Driver)
	}
	gConf := gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		}),
		FullSaveAssociations:                     false,
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: gschema.NamingStrategy{
			TablePrefix: "",
		},
	}
	if config.ShowSQL {
		gConf.Logger = logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		})
	}
	if config.TablePrefix != "" {
		gConf.NamingStrategy = gschema.NamingStrategy{
			TablePrefix: config.TablePrefix,
		}
	}
	gdb, err := gorm.Open(dial, &gConf)
	if err != nil {
		return nil, errors.Wrap(err, "dba: connect failed")
	}
	if config.Name == "" {
		count := 0
		ns.connections.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		config.Name = fmt.Sprintf("%d", count)
	}
	registerCallbacks(gdb)
	conn := &Connection{
		ns:     ns,
		driver: config.Driver,
		dsn:    config.Dsn,
		name:   config.Name,
		gdb:    gdb,
	}
	ns.connections.Store(config.Name, conn)
	return conn, nil
}

func (ns *Namespace) Session(connectionName ...string) *Connection {
	key := "0"
	if len(connectionName) > 0 && connectionName[0] != "" {
		key = connectionName[0]
	}
	conn, ok := ns.connections.Load(key)
	if !ok {
		return nil
	}
	return conn.(*Connection)
}

func (ns *Namespace) ConnectionNames() []string {
	var names []string
	ns.connections.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})
	return names
}

func (ns *Namespace) Disconnect(name ...string) {
	for _, item := range name {
		ns.connections.Delete(item)
	}
}

func (ns *Namespace) DisconnectAll() {
	ns.Disconnect(ns.ConnectionNames()...)
}

func (ns *Namespace) RegisterSchema(value ...any) error {
	ss, err := ParseSchema(value...)
	if err != nil {
		return err
	}
	for _, item := range ss {
		ns.schemas.Store(item.Name, item)
	}
	// 所有模型注册完成后，再统一修复引用关系
	ns.RepairRelationships()
	return nil
}

func (ns *Namespace) LookupSchema(name string) *Schema {
	s, ok := ns.schemas.Load(name)
	if !ok {
		return nil
	}
	original := s.(*Schema)
	return original.Clone()
}

func (ns *Namespace) Schemas(names ...string) map[string]*Schema {
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}
	schemas := make(map[string]*Schema)
	ns.schemas.Range(func(key, value any) bool {
		copied := value.(*Schema).Clone()
		name := key.(string)
		if len(nameMap) == 0 || nameMap[name] {
			schemas[name] = copied
		}
		return true
	})
	return schemas
}

func (ns *Namespace) RepairRelationships() {
	schemas := ns.Schemas()
	for schemaName, s := range schemas {
		var needUpdate bool
		for fieldName, field := range s.Fields {
			if field.RelConfig != "" {
				rel := parseRel(field.RelConfig, s, field, schemas)
				if rel != nil {
					needUpdate = true
					field.Relationship = rel
				}
				if field.ItemType != "" {
					if !scalarTypeMap[SchemaType(field.ItemType)] && schemas[field.ItemType] == nil {
						field.ItemType = ""
					}
				}
				s.Fields[fieldName] = field

			}
		}
		if needUpdate {
			ns.schemas.Store(schemaName, s)
		}
	}
}

func (ns *Namespace) Model(schemaName string) *DataModel {
	return ns.ModelBySession("", schemaName)
}

func (ns *Namespace) Init(connectionName ...string) error {
	schemas := ns.Schemas()
	if len(connectionName) == 0 {
		connectionName = append(connectionName, "")
	}
	for _, name := range connectionName {
		conn := ns.Session(name)
		ddl := conn.GenDDL(schemas, true)
		if _, err := conn.Exec(ddl); err != nil {
			return err
		}
	}
	return nil
}

func (ns *Namespace) ModelBySession(connectionName, schemaName string) *DataModel {
	conn := ns.Session(connectionName)
	if conn == nil {
		panic(fmt.Errorf("connection not exists: %s", connectionName))
	}

	s := ns.LookupSchema(schemaName)
	if s == nil {
		panic(fmt.Errorf("schema not exists: %s", schemaName))
	}

	gdb := conn.NewDB()
	gdb = gdb.Table(s.NativeName)
	return &DataModel{
		conn:   conn,
		schema: s,
		gdb:    gdb,
	}
}
