package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
)

func main() {
	if err := dba.Connect(&dba.ConnectConfig{
		Driver: dba.MySQL,
		Dsn:    os.Getenv("MYSQL"),
	}); err != nil {
		log.Fatal(err)
	}

	err := dba.RegisterSchema(
		&examples.Org{},
		&examples.User{},
		&examples.Profile{},
		&examples.Account{},
		&examples.Dept{},
		&examples.UserDept{},
		&examples.Role{},
		&examples.Permission{},
	)
	if err != nil {
		log.Fatal(err)
	}
	schemaNames := []string{
		"Org",
		"User",
		"Profile",
		"Account",
		"Dept",
		"UserDept",
		"Permission",
	}
	for _, name := range schemaNames {
		ModelSchema := dba.SchemaBy(name)
		log.Println(ModelSchema.Name, ModelSchema.NativeName)
	}
}
