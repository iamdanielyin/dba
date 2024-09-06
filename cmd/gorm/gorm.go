package main

import (
	"fmt"
	"strings"
)

func modifySlice(s []string) {
	for _, item := range s {
		item = "modified__" + strings.TrimSpace(item)
	}
}

func main() {
	strSlice := []string{"original", "value"}
	fmt.Println("Before:", strSlice)

	modifySlice(strSlice)
	fmt.Println("After:", strSlice)
}
