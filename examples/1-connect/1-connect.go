package main

import (
	"github.com/iamdanielyin/dba"
	"log"
)

func main() {
	config := dba.ConnectConfig{
		Driver: "mysql",
		Dsn:    "root:yHD9xA4uXfGJ5v4d@tcp(127.0.0.1:3306)/dba?tls=skip-verify&charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai",
	}

	sess, err := dba.Connect(&config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Connected to %q with DSN:\n\t%q", sess.Name(), sess.DSN())
}
