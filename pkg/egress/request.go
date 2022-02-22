package egress

import "github.com/pion/webrtc/v3"

type StartRecordingRequest struct {
	Room        string        `json:"room"`
	Participant string        `json:"participant"`
	File        string        `json:"file"`
	Channel     OutputChannel `json:"channel"`
}

type StopRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
}

type OutputChannel string

var (
	OutputChannelAudio OutputChannel = OutputChannel(webrtc.RTPCodecTypeAudio.String())
	OutputChannelVideo OutputChannel = OutputChannel(webrtc.RTPCodecTypeVideo.String())
	OutputChannelAV    OutputChannel = "av"
)
