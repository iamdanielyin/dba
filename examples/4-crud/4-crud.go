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
	if err = Permission.Create(&examples.Permission{
		Code: time.Now().Format(time.DateTime),
		Name: "Hello " + time.Now().Format(time.DateTime),
	}); err != nil {
		log.Fatal(err)
	}
	var list []*examples.Permission
	if err := Permission.Find("ID >", 16).And("Code LIKE ?", "2024%").All(&list); err != nil {
		log.Fatal(err)
	}
	log.Println(list)
}
