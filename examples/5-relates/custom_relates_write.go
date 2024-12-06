package main

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	_ "github.com/iamdanielyin/dba/examples/0-init"
)

func main() {
	User := dba.Model("User")

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
				Birthday: "2003-02-02",
				Gender:   "女",
			},
		},
	})

	_, _ = User.Find("ID > 2").Update(&examples.User{
		Username: "Bar",
		Profile: &examples.UserProfile{
			Birthday: "2003-02-02",
			Gender:   "女",
		},
	})
}
