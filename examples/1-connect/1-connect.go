package main

import (
	"github.com/iamdanielyin/dba"
	"log"
	"os"
)

func main() {
	if err := dba.Connect(dba.MYSQL, os.Getenv("MYSQL")); err != nil {
		log.Fatal(err)
	}
}
