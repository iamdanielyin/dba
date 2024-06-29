package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	"log"
	"time"
)

func main() {
	if err := dba.Connect(dba.SQLITE, "./test.db", &dba.ConnectConfig{
		ShowSQL: true,
	}); err != nil {
		log.Fatal(err)
	}

	err := dba.RegisterSchema(
		&examples.Role{},
		&examples.Permission{},
	)
	if err := dba.Session().Init(); err != nil {
		log.Fatal(err)
	}

	Permission := dba.Model("Permission")

	// create
	if err = Permission.Create(&examples.Permission{
		Code: time.Now().Format(time.DateTime),
		Name: "Hello " + time.Now().Format(time.DateTime),
	}); err != nil {
		log.Fatal(err)
	}

	// query
	var list []*examples.Permission
	if err := Permission.Find().Select("ID", "Code").Or("ID >", 16, "Code $PREFIX", "2023").All(&list); err != nil {
		log.Fatal(err)
	}

	// update
	if _, err := Permission.Find("ID >", 16).Update(&examples.Permission{
		Code: "UPDATED",
	}); err != nil {
		log.Fatal(err)
	}

	// delete
	if _, err := Permission.Find("ID =", list[0].ID).Delete(); err != nil {
		log.Fatal(err)
	}
}
