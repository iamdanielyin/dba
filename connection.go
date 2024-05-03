package dba

import (
	"gorm.io/gorm"
	"sync"
)

var DefaultNamespace = &Namespace{
	connections: new(sync.Map),
	schemas:     new(sync.Map),
}

type Driver string

const (
	MYSQL Driver = "MYSQL"
)

type Connection struct {
	driver Driver
	dsn    string
	name   string
	gdb    *gorm.DB
}
