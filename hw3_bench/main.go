package main

import (
	"bytes"
	"fmt"
)

func main() {
	slowOut := new(bytes.Buffer)
	FastSearch(slowOut)
	fmt.Println(slowOut.String())
}