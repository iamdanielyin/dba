package main

import (
	"github.com/guregu/null/v5"
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	"log"
)

// Main function to demonstrate the logic
func main() {
	if err := dba.Connect(dba.SQLITE, "./test.db"); err != nil {
		log.Fatal(err)
	}
	_ = dba.RegisterSchema(
		&examples.Org{},
		&examples.User{},
		&examples.Profile{},
		&examples.Account{},
		&examples.Dept{},
		&examples.UserDept{},
		&examples.Role{},
		&examples.Permission{},
	)

	UserModel := dba.Model("User")
	// Create example data
	org := examples.Org{Code: "001", Name: "Organization 1"}
	role := examples.Role{Name: "Admin"}
	dept := examples.Dept{Name: "Dept 1"}
	user := examples.User{
		Nickname: "John",
		Profile:  &examples.Profile{RealName: "John Doe"},
		Accounts: []*examples.Account{
			{Type: "GitHub", Username: "john123", Password: "secret"},
			{Type: "Email", Username: "john@example.com", Password: "password"},
		},
		Org:   &org,
		Roles: []*examples.Role{&role},
		Departments: []*examples.UserDept{
			{Dept: &dept, IsMain: null.BoolFrom(true)},
		},
	}
	if err := UserModel.Create(&user); err != nil {
		log.Fatal(err)
	}

	// Find users
	var users []*examples.User
	if err := UserModel.Find().Preload("Profile", "Accounts", "Org", "Roles", "Departments").All(&users); err != nil {
		log.Fatal(err)
	}

	// Output results
	for _, item := range users {
		log.Println("User:", item)
		log.Println("User Profile:", item.Profile)
		log.Println("User Accounts:", item.Accounts)
		log.Println("User Org:", item.Org)
		log.Println("User Roles:", item.Roles)
		log.Println("User Departments:", item.Departments)
	}
}
