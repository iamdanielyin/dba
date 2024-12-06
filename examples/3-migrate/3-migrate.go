package main

import (
	"fmt"
	"github.com/iamdanielyin/dba"
	_ "github.com/joho/godotenv/autoload"
	"os"
	"time"
)

func main() {
	schs := dba.SchemaBys()
	if ddl := dba.Session().GenDDL(schs); ddl != "" {
		_ = dba.EnsureDir("migrations")
		_ = os.WriteFile(fmt.Sprintf("migrations/%s.sql", time.Now().Format("20060102150405")), []byte(ddl), os.ModePerm)
	}
}
