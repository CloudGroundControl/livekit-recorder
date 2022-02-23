package egress

type StartRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
	Output      string `json:"output"`
}

type StopRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
}
