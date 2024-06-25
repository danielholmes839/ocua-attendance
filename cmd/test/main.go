package main

import (
	"fmt"
	"os"

	"github.com/danielholmes839/ocua-attendance-bot/internal/ocua"
	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
)

func temp() error {
	godotenv.Load()

	username := os.Getenv("ocua_username")
	password := os.Getenv("ocua_password")

	pw, err := playwright.Run()
	if err != nil {
		return err
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Devtools: playwright.Bool(true),
	})
	if err != nil {
		return err
	}

	fmt.Println("launched browser...")

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		BaseURL: playwright.String("https://www.ocua.ca"),
	})
	if err != nil {
		return err
	}
	defer context.Close()

	err = ocua.Login(username, password, context)
	if err != nil {
		return err
	}

	attendance, _ := ocua.GetAttendance("13313", context)
	fmt.Println(attendance)

	team, _ := ocua.GetTeam("13313", context)
	fmt.Println(team)
	return nil
}

func main() {
	err := temp()
	if err != nil {
		panic(err)
	}
}
