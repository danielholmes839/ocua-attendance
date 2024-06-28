package ocua

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

const (
	ABSENT    = AttendanceStatus("ABSENT")
	ATTENDING = AttendanceStatus("ATTENDING")
	AVAILABLE = AttendanceStatus("AVAILABLE")
	INVITED   = AttendanceStatus("INVITED")
	UNKNOWN   = AttendanceStatus("UNKNOWN")
)

type Attendance struct {
	Gametime time.Time
	Players  map[string]AttendanceStatus
}

type AttendanceStatus string

type attendanceTableColumns struct {
	Gametime time.Time
	Valid    bool
}

type attendanceTableRow struct {
	PlayerID   string
	PlayerName string
	Status     []string
}

func parseAttendanceGametime(text string) (time.Time, error) {
	formats := []string{
		"Jan 2, 2006 3:04PM", // Format for "May 20, 2024 6:45PM"
		"Jan 2, 2006",        // Format for "Jul 1, 2024"
	}

	for _, format := range formats {
		gametime, err := time.ParseInLocation(format, text, time.Local)
		if err == nil {
			return gametime, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse game time: %q", text)
}

func parseAttendanceHeaders(table *goquery.Selection) []attendanceTableColumns {
	headers := []attendanceTableColumns{}

	ths := table.Find("thead > tr > th")
	ths = ths.Slice(1, ths.Length()-2)

	ths.Each(func(i int, s *goquery.Selection) {
		t, err := parseAttendanceGametime(s.Text())

		headers = append(headers, attendanceTableColumns{
			Gametime: t,
			Valid:    err == nil,
		})
	})

	return headers
}

func parseAttendanceBody(table *goquery.Selection) []attendanceTableRow {
	rows := table.Find("tbody > tr")

	nodes := rows.Nodes

	attendanceRows := []attendanceTableRow{}

	for _, node := range nodes {
		row := goquery.NewDocumentFromNode(node)
		player := row.Find("td").First()
		playerName := player.Text()

		if playerName == "" {
			break
		}

		playerHref, _ := player.Find("a").Attr("href")

		u, err := url.Parse(playerHref)
		if err != nil {
			continue
		}

		playerId := u.Query().Get("person")

		attendanceStatus := []string{}

		attendance := row.Find("td")
		attendance = attendance.Slice(1, attendance.Length()-2)

		attendance.Each(func(i int, s *goquery.Selection) {
			status, ok := s.Find("img").Attr("title")

			if !ok {
				attendanceStatus = append(attendanceStatus, "N/A")
				return
			}

			status = strings.TrimPrefix(status, "Current attendance: ")
			status = strings.ToUpper(status)
			attendanceStatus = append(attendanceStatus, status)
		})

		attendanceRows = append(attendanceRows, attendanceTableRow{
			PlayerID:   playerId,
			PlayerName: playerName,
			Status:     attendanceStatus,
		})
	}

	return attendanceRows
}

func GetAttendancePage(teamID string, context playwright.BrowserContext) (*bytes.Buffer, error) {
	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}

	page.Goto(fmt.Sprintf("/zuluru/teams/attendance?team=%s", teamID))
	defer page.Close()

	content, err := page.Content()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte(content))
	return buf, nil
}

func ParseAttendancePage(page io.Reader) ([]Attendance, error) {
	doc, err := goquery.NewDocumentFromReader(page)
	if err != nil {
		return nil, err
	}

	// find the table element
	table := doc.Find("div.teams.attendance").Find("table").First()

	headers := parseAttendanceHeaders(table)
	rows := parseAttendanceBody(table)

	// organize attendance data by week
	weeks := []Attendance{}

	for week, header := range headers {
		// get each players status for the week
		players := map[string]AttendanceStatus{}
		for _, player := range rows {
			players[player.PlayerID] = AttendanceStatus(player.Status[week])
		}

		weeks = append(weeks, Attendance{
			Gametime: header.Gametime,
			Players:  players,
		})
	}

	return weeks, nil
}
