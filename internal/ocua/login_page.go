package ocua

import (
	"github.com/playwright-community/playwright-go"
)

func Login(email, password string, context playwright.BrowserContext) error {
	page, err := context.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	_, err = page.Goto("/user/login")
	if err != nil {
		return err
	}

	page.Locator("#edit-name").First().Fill(email)
	page.Locator("#edit-pass").First().Fill(password)

	err = page.Locator("#edit-submit").Click()
	if err != nil {
		return err
	}

	return nil
}
