package recording

import (
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/pipeline"
	"sync"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/participant"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/upload"
	"github.com/labstack/gommon/log"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/webrtc/v3"
)

type bot struct {
	// ID is mainly for internal use
	id string

	// States
	lock     sync.Mutex
	room     *lksdk.Room
	uploader upload.Uploader

	// Key: identity
	pending map[string]ParticipantRequest

	// Key: identity
	pipelines map[string]pipeline.Pipeline

	callback botCallback
}

type botCallback struct {
	SendRecordingData func(p participant.ParticipantData)
}

func createBot(id string, url string, token string, callback botCallback) (*bot, error) {
	b := &bot{
		id:        id,
		lock:      sync.Mutex{},
		pending:   make(map[string]ParticipantRequest),
		pipelines: make(map[string]pipeline.Pipeline),
		callback:  callback,
	}

	room, err := lksdk.ConnectToRoomWithToken(url, token, lksdk.WithAutoSubscribe(false))
	if err != nil {
		return nil, err
	}

	room.Callback.OnTrackSubscribed = b.OnTrackSubscribed
	room.Callback.OnTrackUnsubscribed = b.OnTrackUnsubscribed
	b.room = room

	return b, nil
}

type ParticipantRequest struct {
	Identity string
	Profile  MediaProfile
}

func (b *bot) pushParticipantRequest(identity string, profile MediaProfile) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.pending[identity] = ParticipantRequest{
		Identity: identity,
		Profile:  profile,
	}
	log.Debugf("pushed participant request | participant: %s, profile: %v", identity, profile)
}

func (b *bot) SetUploader(uploader upload.Uploader) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.uploader = uploader
}

func (b *bot) OnTrackSubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Check if recorder needs to handle this participant
	_, found := b.pending[rp.Identity()]
	if !found {
		log.Warnf("request not found for participant | participant: %s, codec: %s", rp.Identity(), track.Codec().MimeType)
		return
	}
	req := b.pending[rp.Identity()]

	p, err := pipeline.NewTrackPipeline(track)
	if err != nil {
		log.Errorf("cannot create track pipeline: %s", err)
		return
	}

	b.pipelines[req.Identity] = p
	p.Start()
}

func (b *bot) OnTrackUnsubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	// Stop recording: ignore recording as the only error is from updating subscription.
	// In this method, it will always be true as participant has left
	b.stopRecording(rp.Identity())
	log.Debugf("stopped recording | participant: %s, type: %s, codec: %v", rp.Identity(), track.Kind().String(), track.Codec().MimeType)
}

func (b *bot) stopRecording(identity string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Check that the participant exists
	_, found := b.pipelines[identity]
	if !found {
		return
	}

	// Retrieve the participant and stop recording
	p := b.pipelines[identity]
	p.Stop()

	// Send data via webhook (in background)
	//go b.callback.SendRecordingData(p.GetData())

	// Remove participant before returning
	delete(b.pipelines, identity)
}

func (b *bot) disconnect() {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, p := range b.pipelines {
		p.Stop()
	}
	b.room.Disconnect()
}
