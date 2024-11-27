package main

import "github.com/iamdanielyin/dba/examples"

func main() {
	// ---------------------
	// ------ HAS_ONE ------
	// ---------------------
	// 新增
	_ = User.Create([]*examples.User{
		{
			Username: "Foo",
			Profile: &examples.UserProfile{
				Birthday: "2000-01-20",
				Gender:   "男",
			},
		},
		{
			Username: "Bar",
			Profile: &examples.UserProfile{
				ID:       100,
				Birthday: "2003-02-02",
				Gender:   "女",
			},
		},
	})
	// 修改
	_, _ = User.Find("ID > 2").Update(&examples.User{
		Username: "Bar",
		Profile: &examples.UserProfile{
			//ID:       100,
			Birthday: "2003-02-02",
			Gender:   "女",
		},
	})
	// 删除
	_, _ = User.Find("ID > 2").Delete()

	// ----------------------
	// ------ HAS_MANY ------
	// ----------------------
	// 新增
	// 修改
	// 删除

	// ---------------------
	// ------ REF_ONE ------
	// ---------------------
	// 新增
	// 修改
	// 删除

	// --------------------------------
	// ------ REF_MANY [Usage 1] ------
	// --------------------------------
	// 新增
	// 修改
	// 删除

	// --------------------------------
	// ------ REF_MANY [Usage 2] ------
	// --------------------------------
	// 新增
	// 修改
	// 删除
}
