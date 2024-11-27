package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
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

	schs := dba.SchemaBys()
	for name, sch := range schs {
		log.Printf("Schema %s → %s\n", name, sch.NativeName)
	}
	_ = os.WriteFile(
		"schemas.json",
		[]byte(dba.JSONStringify(schs, true)),
		os.ModePerm,
	)
}
