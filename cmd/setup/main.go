package main

import (
	"fmt"
	"os"

	"github.com/danielholmes839/ocua-attendance-bot/internal/ocua"
	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
)

func setup() error {
	godotenv.Load()

	// ocua environment variables
	username := os.Getenv("ocua_username")
	password := os.Getenv("ocua_password")
	teamID := os.Getenv("ocua_team_id")
	baseURL := os.Getenv("ocua_base_url")

	// setup browser
	pw, err := playwright.Run()
	if err != nil {
		return err
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{})
	if err != nil {
		return err
	}

	// setup client browser context
	contextOpts := playwright.BrowserNewContextOptions{
		BaseURL: playwright.String(baseURL),
	}

	context, err := browser.NewContext(contextOpts)
	if err != nil {
		return err
	}
	defer context.Close()

	err = ocua.Login(username, password, context)
	if err != nil {
		return err
	}

	buf, err := ocua.GetTeamPage(teamID, context)
	if err != nil {
		return err
	}

	players, err := ocua.ParseTeamPage(buf)
	if err != nil {
		return err
	}

	for _, player := range players {
		fmt.Printf("# %s\n", player.Name)
		fmt.Printf("%q: %q\n\n", player.ID, "")
	}

	return nil
}

func main() {
	setup()
}
