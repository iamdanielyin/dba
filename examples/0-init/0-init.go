package __init

import (
	"github.com/iamdanielyin/dba"
	"github.com/iamdanielyin/dba/examples"
	"log"
)

func init() {
	if _, err := dba.Connect(&dba.ConnectConfig{
		Driver: "mysql",
		Dsn:    "root:yHD9xA4uXfGJ5v4d@tcp(127.0.0.1:3306)/dba?tls=skip-verify&charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai",
	}); err != nil {
		log.Fatal(err)
	}
	// 注册元数据
	_ = dba.RegisterSchema(
		&examples.Org{},
		&examples.User{},
		&examples.UserSocialMediaAccounts{},
		&examples.UserProfile{},
		&examples.Tag{},
	)
}
