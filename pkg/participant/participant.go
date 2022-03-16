package participant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/upload"
	"github.com/labstack/gommon/log"
	"github.com/lithammer/shortuuid/v4"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/livekit/server-sdk-go/pkg/samplebuilder"
	"github.com/pion/webrtc/v3"
)

const RecordingsDir = "recordings"

type Participant interface {
	GetData() ParticipantData
	IsVideoRecordable() bool
	IsAudioRecordable() bool

	RegisterVideo(track *webrtc.TrackRemote) error
	RegisterAudio(track *webrtc.TrackRemote) error

	Start()
	Stop()
}

type participant struct {
	ctx      context.Context
	data     ParticipantData
	state    state
	uploader upload.Uploader
	pli      lksdk.PLIWriter

	// Filenames
	vf string
	af string

	// Tracks
	vt *webrtc.TrackRemote
	at *webrtc.TrackRemote

	// Recorders
	vr recorder.Recorder
	ar recorder.Recorder
}

func NewParticipant(identity string, uploader upload.Uploader, pli lksdk.PLIWriter) Participant {
	return &participant{
		ctx: context.TODO(),
		data: ParticipantData{
			Identity: identity,
		},
		state:    stateCreated,
		uploader: uploader,
		pli:      pli,
	}
}

func (p *participant) createRecorder(track *webrtc.TrackRemote) (recorder.Recorder, error) {
	fileID := shortuuid.New()
	if fileID == "" {
		return nil, errors.New("empty file ID")
	}

	fileExt := recorder.GetMediaExtension(track.Codec().MimeType)
	if fileExt == "" {
		return nil, errors.New("unsupported media")
	}

	fileName := fmt.Sprintf("%s/%s.%s", RecordingsDir, fileID, fileExt)

	return recorder.New(track.Codec(), fileName, samplebuilder.WithPacketDroppedHandler(func() {
		p.pli(track.SSRC())
	}))
}

func (p *participant) GetData() ParticipantData {
	return p.data
}

func (p *participant) IsVideoRecordable() bool {
	return p.vr != nil
}

func (p *participant) IsAudioRecordable() bool {
	return p.ar != nil
}

func (p *participant) RegisterVideo(track *webrtc.TrackRemote) error {
	r, err := p.createRecorder(track)
	if err != nil {
		return err
	}
	p.vf = r.Sink().Name()
	p.vt = track
	p.vr = r
	return nil
}

func (p *participant) RegisterAudio(track *webrtc.TrackRemote) error {
	r, err := p.createRecorder(track)
	if err != nil {
		return err
	}
	p.af = r.Sink().Name()
	p.at = track
	p.ar = r
	return nil
}

func (p *participant) Start() {
	if p.state != stateCreated {
		return
	}
	if p.vt != nil && p.vr != nil {
		p.vr.Start(p.ctx, p.vt)
	}
	if p.at != nil && p.ar != nil {
		p.ar.Start(p.ctx, p.at)
	}
	p.state = stateRecording
	p.data.Start = time.Now()
}

func (p *participant) Stop() {
	if p.state == stateDone {
		return
	}
	if p.vr != nil {
		p.vr.Stop()
	}
	if p.ar != nil {
		p.ar.Stop()
	}
	p.state = stateDone
	p.data.End = time.Now()

	// Do post processing
	// While it is synchronous, the containerisation should be almost instantenous.
	// Meanwhile, uploading is done in the background via goroutine
	err := p.process()
	if err != nil {
		log.Errorf("error in post processing | error: %v, participant: %s", err, p.data.Identity)
	}
}
