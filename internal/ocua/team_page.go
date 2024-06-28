package ocua

import (
	"bytes"
	"fmt"
	"io"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

type Player struct {
	ID     string
	Name   string
	Role   string
	Gender string
}

func GetTeamPage(teamID string, context playwright.BrowserContext) (*bytes.Buffer, error) {
	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}

	page.Goto(fmt.Sprintf("/zuluru/teams/view?team=%s", teamID))
	defer page.Close()

	content, err := page.Content()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte(content))
	return buf, nil
}

func ParseTeamPage(page io.Reader) (map[string]Player, error) {
	doc, err := goquery.NewDocumentFromReader(page)
	if err != nil {
		return nil, err
	}

	// find the table element
	body := doc.Find("div.related.row").Find("table.table-striped.table-hover > tbody").First()

	rows := body.Find("tr")
	rows = rows.Slice(1, rows.Length()-1)

	players := map[string]Player{}

	rows.Each(func(i int, s *goquery.Selection) {
		cells := s.Find("td")

		playerCell := goquery.NewDocumentFromNode(cells.Get(0))
		playerName := playerCell.Find("a").Text()
		playerHref, ok := playerCell.Find("a").Attr("href")
		if !ok {
			return
		}

		playerUrl, err := url.Parse(playerHref)
		if err != nil {
			return
		}

		playerID := playerUrl.Query().Get("person")

		roleCell := goquery.NewDocumentFromNode(cells.Get(1))
		role := roleCell.Find("a").Text()

		genderCell := goquery.NewDocumentFromNode(cells.Get(2))
		gender := genderCell.Text()
		gender = string([]rune(gender)[0])

		players[playerID] = Player{
			ID:     playerID,
			Name:   playerName,
			Role:   role,
			Gender: gender,
		}
	})

	return players, nil
}
