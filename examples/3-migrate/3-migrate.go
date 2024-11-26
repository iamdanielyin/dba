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
	err := dba.RegisterSchema(
		&examples.Group{},
		&examples.Tag{},
	)
	if err != nil {
		log.Fatal(err)
	}

	schs := dba.LookupSchema()
	_ = os.WriteFile("schemas.json", []byte(dba.JSONStringify(schs, true)), os.ModePerm)

	if ddl := dba.Session().GenDDL(schs); ddl != "" {
		_ = dba.EnsureDir("migrations")
		_ = os.WriteFile(fmt.Sprintf("migrations/%s.sql", time.Now().Format("20060102150405")), []byte(ddl), os.ModePerm)
	}
}
