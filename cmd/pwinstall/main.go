package main

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

func main() {
	err := playwright.Install()
	fmt.Println(err)
}
