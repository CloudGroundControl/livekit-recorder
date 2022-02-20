package egress

import "github.com/pion/webrtc/v3"

type StartRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
	Channel     string `json:"channel"`
	File        string `json:"file"`
}

type StopRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
	Sink        string `json:"sink"`
}

type OutputChannel string

var (
	OutputChannelAudio OutputChannel = OutputChannel(webrtc.RTPCodecTypeAudio.String())
	OutputChannelVideo OutputChannel = OutputChannel(webrtc.RTPCodecTypeVideo.String())
	OutputChannelAV    OutputChannel = "av"
)
