package main

import (
	"fmt"
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
	"time"
)

func main() {
	if err := dba.Connect(dba.MYSQL, os.Getenv("MYSQL")); err != nil {
		log.Fatal(err)
	}

	err := dba.RegisterSchema(
		&examples.Role{},
		&examples.Permission{},
	)
	if err != nil {
		log.Fatal(err)
	}

	allSchemas := dba.Schemas()
	_ = os.WriteFile("schemas.json", []byte(dba.JSONStringify(allSchemas, true)), os.ModePerm)

	if ddl := dba.Session().GenDDL(allSchemas); ddl != "" {
		_ = dba.EnsureDir("migrations")
		_ = os.WriteFile(fmt.Sprintf("migrations/%s.sql", time.Now().Format("20060102150405")), []byte(ddl), os.ModePerm)
	}
}
