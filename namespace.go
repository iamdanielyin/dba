package dba

import (
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

type Namespace struct {
	Name        string
	connections *sync.Map
	schemas     *sync.Map
}

func (ns *Namespace) Connect(name string, drv Driver, dsn string) (*Connection, error) {
	var dial gorm.Dialector
	switch drv {
	case MYSQL:
		dial = mysql.Open(dsn)
	default:
		return nil, errors.Errorf("dba: invalid driver: %s", drv)
	}
	gdb, err := gorm.Open(dial)
	if err != nil {
		return nil, errors.Wrap(err, "dba: connect failed")
	}
	if name == "" {
		count := 0
		ns.connections.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		name = fmt.Sprintf("%d", count)
	}
	conn := &Connection{
		driver: drv,
		dsn:    dsn,
		name:   name,
		gdb:    gdb,
	}
	ns.connections.Store(name, conn)
	return conn, nil
}

func (ns *Namespace) LookupConnection(name ...string) *Connection {
	key := "0"
	if len(name) > 0 && name[0] != "" {
		key = name[0]
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

func (ns *Namespace) RegisterSchemas(value ...any) error {
	ss, err := ParseSchemas(value...)
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
	for schemaName, schema := range schemas {
		var needUpdate bool
		for fieldName, field := range schema.Fields {
			if field.RelConfig != "" {
				rel := parseRel(field.RelConfig, schema, &field, schemas)
				if rel != nil {
					needUpdate = true
					field.Relationship = *rel
				}
				if field.ItemType != "" {
					if !allBasicTypeMap[SchemaType(field.ItemType)] && schemas[field.ItemType] == nil {
						field.ItemType = ""
					}
				}
				schema.Fields[fieldName] = field

			}
		}
		if needUpdate {
			ns.schemas.Store(schemaName, schema)
		}
	}
}
