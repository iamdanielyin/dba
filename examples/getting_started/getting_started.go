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
		&examples.Role{},
		&examples.Permission{},
	)
	if err != nil {
		log.Fatal(err)
	}
	RoleModel := dba.Model("Role")
	RoleModel.Create(&examples.Role{
		Model:       examples.Model{},
		Name:        "",
		Permissions: nil,
		Others:      nil,
	})
}
