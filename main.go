package main

import (
	"fmt"
	"strings"
)

func cleanInput(text string) []string {
	str := strings.ToLower(text)
	trimmed := strings.TrimSpace(str)
	return strings.Split(trimmed, " ")
}

func main() {
	fmt.Println("Hello, World!")
}
