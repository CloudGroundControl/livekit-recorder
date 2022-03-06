package participant

import "time"

type ParticipantData struct {
	Identity string
	Start    time.Time
	End      time.Time
	Output   string
}
