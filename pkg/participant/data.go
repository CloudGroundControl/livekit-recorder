package participant

import "time"

type ParticipantData struct {
	Identity string    `json:"identity"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Output   string    `json:"output"`
}
