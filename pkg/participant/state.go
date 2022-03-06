package participant

type state string

const (
	stateCreated   state = "created"
	stateRecording state = "recording"
	stateDone      state = "done"
)
