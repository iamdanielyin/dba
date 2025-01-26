package dba

import (
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	globalLogger = logrus.New()
)

func init() {
	globalLogger.SetFormatter(&logrus.JSONFormatter{})
	globalLogger.SetOutput(&lumberjack.Logger{
		Filename:   "./logs/dba-" + time.Now().Format("20060102") + ".log", // 使用当前日期作为日志文件名
		MaxSize:    100,                                                    // 每个日志文件的最大大小 (MB)
		MaxBackups: 7,                                                      // 保留最多的旧日志文件
		MaxAge:     30,                                                     // 保留旧日志的最大天数
		Compress:   true,                                                   // 启用压缩旧日志文件
	})
}

type Namespace struct {
	Name        string
	connections *sync.Map
	schemas     *sync.Map
}

type ConnectConfig struct {
	Driver        string         `json:"driver,omitempty"`
	Dsn           string         `json:"dsn,omitempty"`
	Name          string         `json:"name,omitempty"`
	CreateClauses string         `json:"create_clauses,omitempty"`
	DeleteClauses string         `json:"delete_clauses,omitempty"`
	UpdateClauses string         `json:"update_clauses,omitempty"`
	QueryClauses  string         `json:"query_clauses,omitempty"`
	Logger        *logrus.Logger `json:"-"`
}

func (ns *Namespace) Connect(config *ConnectConfig) (*Connection, error) {
	driver := drivers[config.Driver]
	if driver == nil {
		return nil, errors.Errorf("dba: invalid driver: %s", config.Driver)
	}
	xdb, err := driver.Connect(config)
	if err != nil {
		return nil, errors.Wrap(err, "dba: connect failed")
	}
	err = xdb.Ping()
	if err != nil {
		return nil, errors.Wrap(err, "dba: connect failed")
	}
	if config.Name == "" {
		count := 0
		ns.connections.Range(func(key, value any) bool {
			count++
			return true
		})
		config.Name = fmt.Sprintf("%d", count)
	}
	logger := config.Logger
	if logger == nil {
		logger = globalLogger
	}
	conn := &Connection{
		ns:     ns,
		driver: driver,
		dsn:    config.Dsn,
		name:   config.Name,
		xdb:    xdb,
		logger: logger,
	}
	var (
		createClauses = config.CreateClauses
		deleteClauses = config.DeleteClauses
		updateClauses = config.UpdateClauses
		queryClauses  = config.QueryClauses
	)
	if createClauses == "" {
		createClauses = driver.CreateClauses()
	}
	if deleteClauses == "" {
		deleteClauses = driver.DeleteClauses()
	}
	if updateClauses == "" {
		updateClauses = driver.UpdateClauses()
	}
	if queryClauses == "" {
		queryClauses = driver.QueryClauses()
	}
	conn.CreateTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(createClauses))
	conn.DeleteTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(deleteClauses))
	conn.UpdateTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(updateClauses))
	conn.QueryTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(queryClauses))
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
	// 所有模型注册完成后，再统一设置引用关系
	ns.setRelations()
	return nil
}

func (ns *Namespace) UnregisterSchema(names ...string) error {
	for _, name := range names {
		ns.schemas.Delete(name)
	}
	ns.setRelations()
	return nil
}

func (ns *Namespace) SchemaBy(name string) *Schema {
	s, ok := ns.schemas.Load(name)
	if !ok {
		return nil
	}
	original := s.(*Schema)
	return original.Clone()
}

func (ns *Namespace) Schemas() []*Schema {
	var result []*Schema
	for _, item := range ns.SchemaBys() {
		result = append(result, item)
	}
	return result
}

func (ns *Namespace) SchemaBys(names ...string) map[string]*Schema {
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}
	schs := make(map[string]*Schema)
	ns.schemas.Range(func(key, value any) bool {
		copied := value.(*Schema).Clone()
		name := key.(string)
		if len(nameMap) == 0 || nameMap[name] {
			schs[name] = copied
		}
		return true
	})
	return schs
}

func (ns *Namespace) setRelations() {
	schs := ns.SchemaBys()
	for schemaName, s := range schs {
		var needUpdate bool
		for fieldName, field := range s.Fields {
			if field.RelationConfig != "" {
				rel := parseRelation(field.RelationConfig, s, field, schs)
				if rel != nil {
					needUpdate = true
					field.Relation = rel
					if rel.BrgIsNative && rel.BrgSchema != "" {
						srcField := s.Fields[rel.SrcField]
						dstField := schs[rel.DstSchema].Fields[rel.DstField]

						mockNativeSchema := Schema{
							Name:       rel.BrgSchema,
							NativeName: rel.BrgSchema,
							Fields: map[string]*Field{
								rel.BrgSrcField: {
									Name:           rel.BrgSrcField,
									NativeName:     rel.BrgSrcField,
									RequiredConfig: "true",
									Type:           srcField.Type,
									NativeType:     srcField.NativeType,
								},
								rel.BrgDstField: {
									Name:           rel.BrgDstField,
									NativeName:     rel.BrgDstField,
									RequiredConfig: "true",
									Type:           dstField.Type,
									NativeType:     dstField.NativeType,
								},
							},
						}
						_ = ns.RegisterSchema(&mockNativeSchema)
					}
				}
				if field.ItemType != "" {
					if !scalarTypeMap[SchemaType(field.ItemType)] && schs[field.ItemType] == nil {
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

type ModelOptions struct {
	ConnectionName string
	Tx             *sqlx.Tx
}

func (ns *Namespace) Init(connectionName ...string) error {
	schs := ns.SchemaBys()
	if len(connectionName) == 0 {
		connectionName = append(connectionName, "")
	}
	for _, name := range connectionName {
		conn := ns.Session(name)
		ddl := conn.GenDDL(schs, true)
		if _, err := conn.Exec(ddl); err != nil {
			return err
		}
	}
	return nil
}

func (ns *Namespace) Model(schemaName string, options ...*ModelOptions) *DataModel {
	opts := new(ModelOptions)
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}
	connectionName := opts.ConnectionName
	conn := ns.Session(connectionName)
	if conn == nil {
		panic(fmt.Errorf("connection not exists: %s", connectionName))
	}

	s := ns.SchemaBy(schemaName)
	if s == nil {
		panic(fmt.Errorf("schema not exists: %s", schemaName))
	}

	var (
		createTemplate = conn.CreateTemplate
		deleteTemplate = conn.DeleteTemplate
		updateTemplate = conn.UpdateTemplate
		queryTemplate  = conn.QueryTemplate
	)
	if s.CreateClauses != "" {
		createTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(s.CreateClauses))
	}
	if s.DeleteClauses != "" {
		deleteTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(s.DeleteClauses))
	}
	if s.UpdateClauses != "" {
		updateTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(s.UpdateClauses))
	}
	if s.QueryClauses != "" {
		queryTemplate = template.Must(template.New("").Funcs(sprig.FuncMap()).Parse(s.QueryClauses))
	}

	return &DataModel{
		conn:           conn,
		schema:         s,
		xdb:            conn.xdb,
		xtx:            opts.Tx,
		createTemplate: createTemplate,
		deleteTemplate: deleteTemplate,
		updateTemplate: updateTemplate,
		queryTemplate:  queryTemplate,
	}
}
