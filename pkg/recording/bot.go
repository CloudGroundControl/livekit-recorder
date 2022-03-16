package recording

import (
	"sync"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/participant"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/upload"
	"github.com/labstack/gommon/log"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/rtcp"
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
	participants map[string]participant.Participant

	callback botCallback
}

type botCallback struct {
	RemoveSubscription func(bot string, room string, participant string, req MediaProfile) error
	SendRecordingData  func(p participant.ParticipantData)
}

func createBot(id string, url string, token string, callback botCallback) (*bot, error) {
	b := &bot{
		id:           id,
		lock:         sync.Mutex{},
		pending:      make(map[string]ParticipantRequest),
		participants: make(map[string]participant.Participant),
		callback:     callback,
	}

	room, err := lksdk.ConnectToRoomWithToken(url, token)
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

	// Send feedback to the server for synchronisation
	_, err := publication.Receiver().Transport().WriteRTCP([]rtcp.Packet{
		&rtcp.TransportLayerNack{
			MediaSSRC: uint32(track.SSRC()),
		},
		&rtcp.PictureLossIndication{
			MediaSSRC: uint32(track.SSRC()),
		},
	})
	if err != nil {
		log.Errorf("error writing RTCP | error: %v, participant: %s", err, rp.Identity())
	}

	// Retrieve the participant. If they don't exist yet, create a new entry
	_, found = b.participants[req.Identity]
	if !found {
		b.participants[req.Identity] = participant.NewParticipant(req.Identity, b.uploader, rp.WritePLI)
	}
	p := b.participants[req.Identity]

	// Register tracks
	if track.Kind() == webrtc.RTPCodecTypeVideo {
		if err := p.RegisterVideo(track); err != nil {
			log.Errorf("cannot register video | error: %v, participant: %s, track: %s, type: %s, codec: %s", err, rp.Identity(), publication.SID(), track.Kind().String(), track.Codec().MimeType)
			return
		}
		log.Infof("registered video | participant: %s, track: %s, type: %s, codec: %s", rp.Identity(), publication.SID(), track.Kind().String(), track.Codec().MimeType)
	}
	if track.Kind() == webrtc.RTPCodecTypeAudio {
		if err := p.RegisterAudio(track); err != nil {
			log.Errorf("cannot register audio | error: %v, participant: %s, track: %s, type: %s, codec: %s", err, rp.Identity(), publication.SID(), track.Kind().String(), track.Codec().MimeType)
			return
		}
		log.Infof("registered audio | participant: %s, track: %s, type: %s, codec: %s", rp.Identity(), publication.SID(), track.Kind().String(), track.Codec().MimeType)
	}

	// Decide if we need to start recording or wait.
	var canStartRecording = false
	switch req.Profile {
	case MediaAudioOnly:
		if p.IsAudioRecordable() {
			canStartRecording = true
		}
	case MediaVideoOnly:
		if p.IsVideoRecordable() {
			canStartRecording = true
		}
	case MediaMuxedAV:
		if p.IsVideoRecordable() && p.IsAudioRecordable() {
			canStartRecording = true
		}
	}

	// Start recording if allowed
	if canStartRecording {
		p.Start()
		log.Debugf("started recording | participant: %s", rp.Identity())

		delete(b.pending, req.Identity)
		log.Debugf("removed participant request | participant: %s", rp.Identity())
	}
}

func (b *bot) OnTrackUnsubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	// Stop recording: ignore recording as the only error is from updating subscription.
	// In this method, it will always be true as participant has left
	b.stopRecording(rp.Identity())
	log.Debugf("stopped recording | participant: %s, type: %s, codec: %v", rp.Identity(), track.Kind().String(), track.Codec().MimeType)
}

func (b *bot) stopRecording(identity string) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Check that the participant exists
	_, found := b.participants[identity]
	if !found {
		return nil
	}

	// Retrieve the participant and stop recording
	p := b.participants[identity]
	p.Stop()

	// Send data via webhook (in background)
	go b.callback.SendRecordingData(p.GetData())

	// Remove participant before returning
	defer delete(b.participants, identity)

	// Find out media profile
	var profile MediaProfile
	switch {
	case p.IsVideoRecordable() && p.IsAudioRecordable():
		profile = MediaMuxedAV
	case p.IsVideoRecordable():
		profile = MediaVideoOnly
	case p.IsAudioRecordable():
		profile = MediaAudioOnly
	default:
		return ErrUnknownMediaProfile
	}

	// Remove subscription
	return b.callback.RemoveSubscription(b.id, b.room.Name, identity, profile)
}
