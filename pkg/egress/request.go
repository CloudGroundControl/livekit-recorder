package egress

import (
	"strings"
)

type StartRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
	Output      string `json:"output"`
}

type StopRecordingRequest struct {
	Room        string `json:"room"`
	Participant string `json:"participant"`
}

type OutputDescription struct {
	// RemoteID refers to the final ID requested by the user. This can
	// include subdirectory prefix such as `livekit/my-video`. If we use
	// this ID for local storage, the video will fail silently as we need to
	// create a local directory called `livekit` as well.
	RemoteID string

	// "Temporary" ID for local storage if RemoteID is a file in directory (contains "/")
	LocalID string
}

func getOutputDescription(output string) OutputDescription {
	var localID = output
	if strings.Contains(output, "/") {
		// Strip away the directories (described by "/") to get the root file ID
		tokens := strings.SplitAfter(output, "/")
		localID = tokens[len(tokens)-1]
	}
	return OutputDescription{
		LocalID:  localID,
		RemoteID: output,
	}
}
