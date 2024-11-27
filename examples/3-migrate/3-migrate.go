package main

import (
	"fmt"
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	_ "github.com/iamdanielyin/dba/examples/0-init"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
	"time"
)

func main() {
	// 注册元数据
	err := dba.RegisterSchema(
		&examples.Tenant{},
		&examples.User{},
		&examples.UserProfile{},
		&examples.Address{},
		&examples.Group{},
		&examples.Tag{},
		&examples.UserGroup{},
	)
	if err != nil {
		log.Fatal(err)
	}

	schs := dba.SchemaBys()
	if ddl := dba.Session().GenDDL(schs); ddl != "" {
		_ = dba.EnsureDir("migrations")
		_ = os.WriteFile(fmt.Sprintf("migrations/%s.sql", time.Now().Format("20060102150405")), []byte(ddl), os.ModePerm)
	}
}
