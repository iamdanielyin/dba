package main

import (
	"fmt"
	"github.com/iamdanielyin/dba"
)

type Inner struct {
	C int
	D float64
}

type Outer struct {
	A     int
	B     string
	Inner     // Anonymous field (embedded struct)
	e     int // Unexported field
}

func main() {
	outer := Outer{
		A: 1,
		B: "hello",
		Inner: Inner{
			C: 2,
			D: 3.14,
		},
		e: 5,
	}

	parsed := dba.ParseStruct(outer)
	fmt.Printf("%v\n", parsed)
}
