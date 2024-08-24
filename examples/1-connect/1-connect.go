package main

import (
	"github.com/iamdanielyin/dba"
	"log"
	"os"
)

func main() {
	if err := dba.Connect(&dba.ConnectConfig{
		Driver: dba.MySQL,
		Dsn:    os.Getenv("MYSQL"),
	}); err != nil {
		log.Fatal(err)
	}
}
