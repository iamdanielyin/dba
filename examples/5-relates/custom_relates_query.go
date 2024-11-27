package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	"log"
)

var (
	User  = dba.Model("User")
	users []*examples.User
)

func main() {
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
	var profiles []*examples.UserProfile
	if err := Profile.Find("UserID $IN", userIds).All(&profiles); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	userProfileMap := make(map[uint]*examples.UserProfile)
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
	Address := dba.Model("Address")
	var accounts []*examples.Address
	if err := Address.Find("UserID $IN", userIds).All(&accounts); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	userAddressesMap := make(map[uint][]*examples.Address)
	for _, account := range accounts {
		userAddressesMap[account.UserID] = append(userAddressesMap[account.UserID], account)
	}
	// 4.回写字段
	for i, user := range users {
		user.Addresses = userAddressesMap[user.ID]
		users[i] = user
	}

	// ---------------------
	// ------ REF_ONE ------
	// ---------------------
	// 1.收集id
	var tenantIDs []uint
	for _, user := range users {
		tenantIDs = append(tenantIDs, user.TenantID)
	}
	// 2.统一查询
	Tenant := dba.Model("Tenant")
	var tenants []*examples.Tenant
	if err := Tenant.Find("TenantID $IN", tenantIDs).All(&tenants); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	tenantMap := make(map[uint]*examples.Tenant)
	for _, tenant := range tenants {
		tenantMap[tenant.ID] = tenant
	}
	// 4.回写字段
	for i, user := range users {
		user.Tenant = tenantMap[user.TenantID]
		users[i] = user
	}

	// --------------------------------
	// ------ REF_MANY [Usage 1] ------
	// --------------------------------
	// 1.收集id（同上）
	// 2.统一查询
	var allBrgData []map[string]any
	if err := dba.Query(&allBrgData, `SELECT * FROM user_tag_refers WHERE user_id IN (?)`, userIds); err != nil {
		log.Fatal(err)
	}
	var allTagIds []uint
	for _, brg := range allBrgData {
		allTagIds = append(allTagIds, brg["tag_id"].(uint))
	}
	Tag := dba.Model("Tag")
	var allTags []*examples.Tag
	if err := Tag.Find("ID $IN", allTagIds).All(&allTags); err != nil {
		log.Fatal(err)
	}
	allTagMap := make(map[uint]*examples.Tag)
	for _, tag := range allTags {
		allTagMap[tag.ID] = tag
	}
	// 3.建立映射
	userTagsMap := make(map[uint][]*examples.Tag)
	for _, brg := range allBrgData {
		userId := brg["user_id"].(uint)
		tagId := brg["tag_id"].(uint)
		tag := allTagMap[tagId]
		if tag == nil {
			continue
		}
		userTagsMap[userId] = append(userTagsMap[userId], tag)
	}
	// 4.回写字段
	for i, user := range users {
		user.Tags = userTagsMap[user.ID]
		users[i] = user
	}

	// --------------------------------
	// ------ REF_MANY [Usage 2] ------
	// --------------------------------
	// 1.收集id（同上）
	// 2.统一查询
	var allBrgSchData []*examples.UserGroup
	UserGroup := dba.Model("UserGroup")
	if err := UserGroup.Find("UserID $IN", userIds).All(&allBrgSchData); err != nil {
		log.Fatal(err)
	}
	// 3.建立映射
	userGroupsMap := make(map[uint][]*examples.UserGroup)
	for _, brg := range allBrgSchData {
		userGroupsMap[brg.UserID] = append(userGroupsMap[brg.UserID], brg)
	}
	// 4.回写字段
	for i, user := range users {
		user.Groups = userGroupsMap[user.ID]
		users[i] = user
	}

	docs := []*examples.User{
		{
			Username: "测试1",
			Profile: &examples.UserProfile{
				Gender: "男",
			},
		},
		{
			Username: "测试2",
			Profile: &examples.UserProfile{
				Gender:   "女",
				Birthday: "2000-01-20",
			},
		},
	}
	_ = User.Create(&docs)
}
