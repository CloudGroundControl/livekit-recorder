package egress

import (
	"errors"
	"sync"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/webrtc/v3"
)

type bot struct {
	lock      sync.Mutex
	room      *lksdk.Room
	pending   map[string]TrackRequest
	recorders map[string]recorder.Recorder
}

type TrackRequest struct {
	SID    string
	Output string
}

var ErrRecorderNotFound = errors.New("recorder not found")

func createBot(url string, token string) (*bot, error) {
	b := &bot{
		lock:      sync.Mutex{},
		room:      nil,
		pending:   make(map[string]TrackRequest),
		recorders: make(map[string]recorder.Recorder),
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

func (b *bot) pushTrackRequests(reqs []TrackRequest) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, req := range reqs {
		b.pending[req.SID] = req
	}
}

func (b *bot) stopRecording(trackSid string) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Check that recorder exists
	_, found := b.recorders[trackSid]
	if !found {
		return ErrRecorderNotFound
	}

	// Stop recorder and remove from list
	r := b.recorders[trackSid]
	r.Stop()
	delete(b.recorders, trackSid)
	return nil
}

func (b *bot) OnTrackSubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Check if track is in pending list. If not, we don't need to record it
	req, found := b.pending[publication.SID()]
	if !found {
		return
	}

	// Guess file extension from codec and generate file name
	filename, err := getMediaFilename(req.Output, track.Codec().MimeType)
	if err != nil {
		logger.Warnw("cannot generate file name", err, "SID", publication.SID(), "Output", req.Output, "Codec", track.Codec().MimeType)
		return
	}

	// Create file sink
	sink, err := recorder.NewFileSink(filename)
	if err != nil {
		logger.Warnw("cannot generate file sink", err, "SID", publication.SID())
		return
	}

	// Create recorder and start recording
	rec, err := recorder.NewTrackRecorder(track.Codec(), sink)
	if err != nil {
		logger.Warnw("cannot create recorder", err, "SID", publication.SID(), "Output", req.Output, "Codec", track.Codec().MimeType)
	}
	rec.Start(track)

	// Attach recorder and remove track request from pending list
	b.recorders[publication.SID()] = rec
	delete(b.pending, publication.SID())
}

func (b *bot) OnTrackUnsubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	// Stop recording
	err := b.stopRecording(publication.SID())
	if err != nil {
		logger.Warnw("cannot stop recorder", err, "track SID", publication.SID())
	}
}
