package bot

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/danielholmes839/ocua-attendance-bot/internal/ocua"
)

func formatPlayers(players []ocua.Player, discordIds map[string]string) string {
	sorted := make([]ocua.Player, len(players))
	copy(sorted, players)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	names := make([]string, len(players))
	for i, player := range sorted {
		discordId, ok := discordIds[player.ID]
		if ok && discordId != "" {
			names[i] = fmt.Sprintf("<@%s>", discordId)
		} else {
			names[i] = player.Name
		}
	}

	return strings.Join(names, ", ")
}

func formatAttendanceReport(report ocua.AttendanceReport, gametime time.Time, players map[string]string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Current attendance for %s: %dO, %dW\n\n", gametime.Format("Monday Jan 2"), len(report.Open), len(report.Woman)))

	if len(report.Unknown) > 0 {
		sb.WriteString(fmt.Sprintf("Reminder to please update your attendance: %s\n\n", formatPlayers(report.Unknown, players)))
	}

	if len(report.Invited) > 0 {
		sb.WriteString(fmt.Sprintf("The following subs have been invited: %s\n\n", formatPlayers(report.Invited, players)))
	}

	sb.WriteString("[Click here to view game attendance on OCUA](https://www.ocua.ca/zuluru/teams/attendance?team=13313)")

	return sb.String()
}

func generateAutocomplete(attendance []ocua.Attendance, t time.Time) []*discordgo.ApplicationCommandOptionChoice {
	choices := []*discordgo.ApplicationCommandOptionChoice{}

	for _, week := range attendance {
		if t.After(week.Gametime) {
			continue
		}

		name := week.Gametime.Format("Jan 2")
		val := week.Gametime.Format("2006-01-02")

		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  name,
			Value: val,
		})
	}

	return choices
}

type Client interface {
	GetTeam(teamID string) (map[string]ocua.Player, error)
	GetAttendance(teamID string) ([]ocua.Attendance, error)
}

type Bot struct {
	TeamID        string
	Client        Client
	ApplicationID string
	GuildID       string
	Players       map[string]string // map of ocua id -> discord id

	sync.RWMutex
	cachedAttendance []ocua.Attendance
}

func (b *Bot) getCachedAttendance() []ocua.Attendance {
	b.RLock()
	defer b.RUnlock()
	return b.cachedAttendance

}

func (b *Bot) setCachedAttendance(attendance []ocua.Attendance) {
	b.Lock()
	defer b.Unlock()
	b.cachedAttendance = attendance
}

func (b *Bot) HandleAttendanceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "generating report...",
			Flags:   4,
		},
	})

	wg := sync.WaitGroup{}
	wg.Add(2)

	var (
		team          map[string]ocua.Player
		teamErr       error
		attendance    []ocua.Attendance
		attendanceErr error
	)

	// get team, attendance in parallel
	go func() {
		team, teamErr = b.Client.GetTeam(b.TeamID)
		wg.Done()
	}()

	go func() {
		attendance, attendanceErr = b.Client.GetAttendance(b.TeamID)
		wg.Done()
	}()

	wg.Wait()

	// handle errors getting attendance data
	if attendanceErr != nil {
		msg := "failed to get team data"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})

		slog.Error(msg, "err", attendanceErr)
		return
	}

	b.setCachedAttendance(attendance)

	// handle errors getting team data
	if teamErr != nil {
		msg := "failed to get team data"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})
		slog.Error(msg, "err", teamErr)
		return
	}

	// find attendance for the requested date
	cmd := i.ApplicationCommandData()
	date := cmd.Options[0].StringValue() // week in "YYYY-mm-dd"
	index := -1
	for i, week := range attendance {
		if week.Gametime.Format("2006-01-02") == date {
			index = i
			break
		}
	}

	// no matching week
	if index == -1 {
		msg := "failed to find matching week"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})
		slog.Error(msg, "date", date)
		return
	}

	// get report info
	report := ocua.GetAttendanceReport(attendance[index], team)
	gametime := attendance[index].Gametime

	content := formatAttendanceReport(report, gametime, b.Players)

	// respond with report info
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	slog.Info("successfully handled attendance command")
}

func (b *Bot) HandleAttendanceAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	attendance := b.getCachedAttendance()
	choices := generateAutocomplete(attendance, time.Now().Local())

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices, // This is basically the whole purpose of autocomplete interaction - return custom options to the user.
		},
	})

	if err != nil {
		slog.Error("failed to send autocomplete data", "err", err)
		return
	}

	slog.Info("successfully handled autocomplete")
}

func (b *Bot) HandleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		command := i.ApplicationCommandData()
		if command.Name == "attendance" {
			b.HandleAttendanceCommand(s, i)
		}
	}

	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		b.HandleAttendanceAutocomplete(s, i)
	}
}

func (b *Bot) RegisterAttendanceCommand(dg *discordgo.Session) error {
	_, err := dg.ApplicationCommandCreate(b.ApplicationID, "", &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "Check the attendance for, @ the people who haven't entered their attendance",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "week",
				Description:  "The week to check attendance for",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
		},
	})
	return err
}

func (b *Bot) Run(token string) error {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	b.RegisterAttendanceCommand(dg)

	attendance, err := b.Client.GetAttendance(b.TeamID)
	if err != nil {
		return err
	}

	b.setCachedAttendance(attendance)

	dg.AddHandler(b.HandleInteractionCreate)

	err = dg.Open()
	if err != nil {
		return err
	}

	slog.Info("the bot is running!")
	select {}
}
