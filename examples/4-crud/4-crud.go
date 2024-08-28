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
		&examples.Org{},
		&examples.User{},
		&examples.Profile{},
		&examples.Account{},
		&examples.Dept{},
		&examples.UserDept{},
		&examples.Role{},
		&examples.Permission{},
	)
	if err := dba.Session().Init(); err != nil {
		log.Fatal(err)
	}

	Permission := dba.Model("Permission")

	// create
	doc := examples.Permission{
		Code: time.Now().Format(time.DateTime),
		Name: "Hello " + time.Now().Format(time.DateTime),
	}
	if err = Permission.Create(&doc); err != nil {
		log.Fatal(err)
	}

	// query
	var list []*examples.Permission
	if err := Permission.Find().Select("ID", "Code").Or("ID >", 1, "Code $PREFIX", "2023").All(&list); err != nil {
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

	// preload
	User := dba.Model("User")
	var users []*examples.User
	if err := User.Find().Preload("Profile", "Org", "Roles").Preload("Roles.Permissions").All(&users); err != nil {
		log.Fatal(err)
	}
	// TODO 手工联查
	// 1.收集id
	var userIds []uint
	for _, user := range users {
		userIds = append(userIds, user.ID)
	}
	// 2.统一查询
	Profile := dba.Model("Profile")
	var profiles []*examples.Profile
	if err := Profile.Find("UserID $IN", userIds).All(&profiles); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	userProfileMap := make(map[uint]*examples.Profile)
	for _, profile := range profiles {
		userProfileMap[profile.UserID] = profile
	}
	// 4.回写字段
	for i, user := range users {
		user.Profile = userProfileMap[user.ID]
		users[i] = user
	}
	log.Println(len(users))
}
