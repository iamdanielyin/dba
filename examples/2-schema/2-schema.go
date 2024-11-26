package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	_ "github.com/joho/godotenv/autoload"
	"log"
)

func main() {
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
	schemaNames := []string{
		"Tenant",
		"User",
		"UserProfile",
		"Address",
		"Group",
		"Tag",
		"UserGroup",
	}
	for _, name := range schemaNames {
		ModelSchema := dba.SchemaBy(name)
		log.Println(ModelSchema.Name, ModelSchema.NativeName)
	}

}
