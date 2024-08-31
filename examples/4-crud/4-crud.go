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

	// populate
	User := dba.Model("User")
	var users []*examples.User
	if err := User.Find().Populate("Profile", "Org", "Roles").Populate("Roles.Permissions").All(&users); err != nil {
		log.Fatal(err)
	}

	// TODO 手工联查

	// ---------------------
	// ------ HAS_ONE ------
	// ---------------------
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

	// ----------------------
	// ------ HAS_MANY ------
	// ----------------------
	// 1.收集id（同上）
	// 2.统一查询
	Account := dba.Model("Account")
	var accounts []*examples.Account
	if err := Account.Find("UserID $IN", userIds).All(&accounts); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	userAccountsMap := make(map[uint][]*examples.Account)
	for _, account := range accounts {
		userAccountsMap[account.UserID] = append(userAccountsMap[account.UserID], account)
	}
	// 4.回写字段
	for i, user := range users {
		user.Accounts = userAccountsMap[user.ID]
		users[i] = user
	}

	// ---------------------
	// ------ REF_ONE ------
	// ---------------------
	// 1.收集id
	var orgIds []uint
	for _, user := range users {
		orgIds = append(orgIds, user.OrgID)
	}
	// 2.统一查询
	Org := dba.Model("Org")
	var orgs []*examples.Org
	if err := Org.Find("OrgID $IN", orgIds).All(&orgs); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	orgMap := make(map[uint]*examples.Org)
	for _, org := range orgs {
		orgMap[org.ID] = org
	}
	// 4.回写字段
	for i, user := range users {
		user.Org = orgMap[user.OrgID]
		users[i] = user
	}

	// --------------------------------
	// ------ REF_MANY [Usage 1] ------
	// --------------------------------
	// 1.收集id（同上）
	// 2.统一查询
	var allBrgData []map[string]any
	if err := dba.Query(&allBrgData, `SELECT * FROM user_role_ref WHERE user_id IN (?)`, userIds); err != nil {
		log.Fatal(err)
	}
	var allRoleIds []uint
	for _, brg := range allBrgData {
		allRoleIds = append(allRoleIds, brg["role_id"].(uint))
	}
	Role := dba.Model("Role")
	var allRoles []*examples.Role
	if err := Role.Find("ID $IN", allRoleIds).All(&allRoles); err != nil {
		log.Fatal(err)
	}
	allRoleMap := make(map[uint]*examples.Role)
	for _, role := range allRoles {
		allRoleMap[role.ID] = role
	}
	// 3.建立映射
	userRolesMap := make(map[uint][]*examples.Role)
	for _, brg := range allBrgData {
		userId := brg["user_id"].(uint)
		roleId := brg["role_id"].(uint)
		role := allRoleMap[roleId]
		if role == nil {
			continue
		}
		userRolesMap[userId] = append(userRolesMap[userId], role)
	}
	// 4.回写字段
	for i, user := range users {
		user.Roles = userRolesMap[user.ID]
		users[i] = user
	}

	// --------------------------------
	// ------ REF_MANY [Usage 2] ------
	// --------------------------------
	// 1.收集id（同上）
	// 2.统一查询
	var allBrgSchData []*examples.UserDept
	UserDept := dba.Model("UserDept")
	if err := UserDept.Find("UserID $IN", userIds).All(&allBrgSchData); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	userDeptsMap := make(map[uint][]*examples.UserDept)
	for _, brg := range allBrgSchData {
		userDeptsMap[brg.UserID] = append(userDeptsMap[brg.UserID], brg)
	}
	// 4.回写字段
	for i, user := range users {
		user.Departments = userDeptsMap[user.ID]
		users[i] = user
	}
}
