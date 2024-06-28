package main

import (
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/danielholmes839/ocua-attendance-bot/internal/bot"
	"github.com/danielholmes839/ocua-attendance-bot/internal/ocua"
	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
	"gopkg.in/yaml.v2"
)

func launch() error {
	godotenv.Load()

	// ocua environment variables
	username := os.Getenv("ocua_username")
	password := os.Getenv("ocua_password")
	teamID := os.Getenv("ocua_team_id")
	baseURL := os.Getenv("ocua_base_url")

	// discord environment variables
	guildID := os.Getenv("discord_guild_id")
	applicationID := os.Getenv("discord_application_id")
	token := os.Getenv("discord_bot_token")

	data, err := os.ReadFile("./data/players.yaml")
	if err != nil {
		return err
	}

	players := map[string]string{}
	yaml.Unmarshal(data, players)

	// setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	// setup playwright browser
	startup := time.Now()
	pw, err := playwright.Run()
	if err != nil {
		return err
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{})
	if err != nil {
		return err
	}

	dur := time.Since(startup)
	logger.Info("launched playwright browser", "dur", dur.String())

	// setup client browser context
	contextOpts := playwright.BrowserNewContextOptions{
		BaseURL: playwright.String(baseURL),
	}

	context, err := browser.NewContext(contextOpts)
	if err != nil {
		return err
	}
	defer context.Close()

	// setup client
	client := &ocua.Client{
		RWMutex:        sync.RWMutex{},
		BrowserContext: context,
	}

	refresher := &ocua.ClientSessionRefresher{
		Browser:                  browser,
		BrowserNewContextOptions: contextOpts,
		Client:                   client,
		Username:                 username,
		Password:                 password,
		Logger:                   logger,
	}

	refresher.RunBackground()

	// setup the discord bot
	b := &bot.Bot{
		TeamID:        teamID,
		Client:        client,
		ApplicationID: applicationID,
		GuildID:       guildID,
		Players:       players,
	}

	b.Run(token)
	return nil
}

func main() {
	err := launch()
	if err != nil {
		panic(err)
	}
}
