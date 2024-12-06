package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	_ "github.com/iamdanielyin/dba/examples/0-init"
	"log"
)

func main() {
	User := dba.Model("User")

	var user *examples.User
	if err := User.Find("Username", "DanielYin").
		Populate("Profile").
		PopulateBy(&dba.PopulateOptions{
			Path:  "Addresses",
			Match: dba.And("PostalCode", "510000"),
			Fields: []string{
				"Name",
				"Phone",
			},
		}).
		Populate("DefaultAddress").
		Populate("Tenant").
		Populate("Tags").
		Populate("Groups").
		One(&user); err != nil {
		log.Fatal(err)
	}
}
