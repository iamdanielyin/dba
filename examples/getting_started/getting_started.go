package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples/schemas"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
)

func main() {
	if err := dba.Connect(dba.MYSQL, os.Getenv("MYSQL")); err != nil {
		log.Fatal(err)
	}
	err := dba.RegisterSchemas(
		&schemas.Org{},
		&schemas.User{},
		&schemas.Profile{},
		&schemas.Account{},
		&schemas.Dept{},
		&schemas.UserDept{},
		&schemas.Role{},
		&schemas.Permission{},
	)
	if err != nil {
		log.Fatal(err)
	}
	allSchemas := dba.Schemas()
	log.Println(dba.JSONStringify(allSchemas))
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
