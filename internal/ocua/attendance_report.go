package ocua

type AttendanceReport struct {
	Open    []Player
	Woman   []Player
	Unknown []Player
	Invited []Player
}

func GetAttendanceReport(week Attendance, team map[string]Player) AttendanceReport {
	w := []Player{}
	o := []Player{}
	invited := []Player{}
	unknown := []Player{}

	for playerID, status := range week.Players {
		player, ok := team[playerID]
		if !ok {
			continue
		}

		// add player to report
		if status == ATTENDING {
			if player.Gender == "W" {
				w = append(w, player)
			} else {
				o = append(o, player)
			}
		} else if status == UNKNOWN && player.Role != "Substitute player" {
			unknown = append(unknown, player)
		} else if status == INVITED && player.Role == "Substitute player" {
			invited = append(invited, player)
		}
	}

	return AttendanceReport{
		Open:    o,
		Woman:   w,
		Unknown: unknown,
		Invited: invited,
	}
}
