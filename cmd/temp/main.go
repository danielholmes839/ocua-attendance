package main

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/danielholmes839/ocua-attendance-bot/internal/ocua"
	"github.com/joho/godotenv"
	"github.com/spf13/afero"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	logger.Info("this is my message", slog.Any("err", errors.New("err")))

	godotenv.Load()

	fs := afero.NewOsFs()

	data, err := afero.ReadFile(fs, "./data/attendance.html")
	if err != nil {
		panic(err)
	}

	attendance, err := ocua.ParseAttendancePage(bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	data, err = afero.ReadFile(fs, "./data/team.html")
	if err != nil {
		panic(err)
	}

	team, err := ocua.ParseTeamPage(bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	fmt.Println(attendance, team)
}
