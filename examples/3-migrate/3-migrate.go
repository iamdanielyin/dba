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
	if err := dba.Connect(&dba.ConnectConfig{
		Driver: dba.MySQL,
		Dsn:    os.Getenv("MYSQL"),
	}); err != nil {
		log.Fatal(err)
	}

	err := dba.RegisterSchema(
		&examples.Role{},
		&examples.Permission{},
	)
	if err != nil {
		log.Fatal(err)
	}

	schs := dba.LookupSchema()
	_ = os.WriteFile("schs.json", []byte(dba.JSONStringify(schs, true)), os.ModePerm)

	if ddl := dba.Session().GenDDL(schs); ddl != "" {
		_ = dba.EnsureDir("migrations")
		_ = os.WriteFile(fmt.Sprintf("migrations/%s.sql", time.Now().Format("20060102150405")), []byte(ddl), os.ModePerm)
	}
}
