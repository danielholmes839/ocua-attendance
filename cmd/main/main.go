package main

import (
	"bytes"
	"fmt"

	"github.com/danielholmes839/ocua-attendance-bot/internal/ocua"
	"github.com/spf13/afero"
)

func temp() error {
	fs := afero.NewOsFs()

	data, err := afero.ReadFile(fs, "./data/team.html")
	if err != nil {
		return err
	}

	team, err := ocua.ParseTeamPage(bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	data, err = afero.ReadFile(fs, "./data/attendance.html")
	if err != nil {
		return err
	}

	attendance, err := ocua.ParseAttendancePage(bytes.NewBuffer(data))

	week := attendance[0]

	for id, player := range team {
		fmt.Println(player, week.Players[id])
	}

	return err
}

func main() {
	err := temp()
	if err != nil {
		panic(err)
	}

}
