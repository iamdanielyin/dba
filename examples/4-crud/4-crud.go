package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
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
		&examples.Tenant{},
		&examples.User{},
		&examples.UserProfile{},
		&examples.Address{},
		&examples.Tenant{},
		&examples.Tag{},
		&examples.UserGroup{},
	)
	if err := dba.Session().Init(); err != nil {
		log.Fatal(err)
	}

	Tenant := dba.Model("Tenant")

	// create
	doc := examples.Tenant{
		Name: time.Now().Format(time.DateTime),
	}
	if err = Tenant.Create(&doc); err != nil {
		log.Fatal(err)
	}

	// query
	var list []*examples.Tenant
	if err := Tenant.Find().Select("ID", "Name").AddOr("ID >", 1, "Name $PREFIX", "2023").All(&list); err != nil {
		log.Fatal(err)
	}

	// update
	if _, err := Tenant.Find("ID >", 16).Update(&examples.Tenant{
		Name: "UPDATED",
	}); err != nil {
		log.Fatal(err)
	}

	// delete
	if _, err := Tenant.Find("ID =", list[0].ID).Delete(); err != nil {
		log.Fatal(err)
	}

	// populate
	User := dba.Model("User")
	var users []*examples.User
	if err := User.Find().Populate("Profile", "DefaultAddress", "Addresses").Populate("Roles.Permissions").All(&users); err != nil {
		log.Fatal(err)
	}
}
