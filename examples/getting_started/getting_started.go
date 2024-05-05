package main

import (
	_ "ariga.io/atlas-provider-gorm/gormschema"
	"fmt"
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples/schemas"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
	"time"
)

func main() {
	if err := dba.Connect(dba.MYSQL, os.Getenv("MYSQL")); err != nil {
		log.Fatal(err)
	}
	err := dba.RegisterSchemas(
		//&schemas.Org{},
		//&schemas.User{},
		//&schemas.Profile{},
		//&schemas.Account{},
		//&schemas.Dept{},
		//&schemas.UserDept{},
		&schemas.Role{},
		&schemas.Permission{},
	)
	if err != nil {
		log.Fatal(err)
	}
	allSchemas := dba.Schemas()
	_ = os.WriteFile("schemas.json", []byte(dba.JSONStringify(allSchemas, true)), os.ModePerm)

	if ddl := dba.GenDDL(allSchemas); ddl != "" {
		_ = dba.EnsureDir("migrations")
		_ = os.WriteFile(fmt.Sprintf("migrations/%s.sql", time.Now().Format("20060102150405")), []byte(ddl), os.ModePerm)
	}
	//testSchemaNames := []string{
	//	"Org",
	//	"User",
	//	"Profile",
	//	"Account",
	//	"Dept",
	//	"UserDept",
	//	"Permission",
	//}
	//for _, name := range testSchemaNames {
	//	ModelSchema := dba.LookupSchema(name)
	//	log.Println(ModelSchema.Name, ModelSchema.NativeName)
	//}
}
