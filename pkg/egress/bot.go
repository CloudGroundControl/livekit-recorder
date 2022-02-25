package egress

import (
	"errors"
	"os"
	"sync"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
	"github.com/cloudgroundcontrol/livekit-recorder/pkg/upload"
	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/webrtc/v3"
)

type bot struct {
	lock      sync.Mutex
	room      *lksdk.Room
	pending   map[string]trackRequest
	recorders map[string]*recorderInstance
	uploader  upload.Uploader
}

type trackRequest struct {
	sid    string
	output OutputDescription
}

type recorderInstance struct {
	output   OutputDescription
	codec    webrtc.RTPCodecParameters
	instance recorder.Recorder
}

var ErrRecorderNotFound = errors.New("recorder not found")

func createBot(url string, token string, uploader upload.Uploader) (*bot, error) {
	b := &bot{
		lock:      sync.Mutex{},
		room:      nil,
		pending:   make(map[string]trackRequest),
		recorders: make(map[string]*recorderInstance),
		uploader:  uploader,
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

func (b *bot) pushTrackRequests(reqs []trackRequest) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, req := range reqs {
		b.pending[req.sid] = req
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

	// If recorder exists, remember to remove it at the end
	defer delete(b.recorders, trackSid)

	// Stop the recorder
	r := b.recorders[trackSid]
	r.instance.Stop()

	// Check if we should containerise the file (support video only for now)
	var err error
	if isContainerisable(r.codec) {
		err = containeriseFile(r.instance.Sink().Name())
		if err != nil {
			return err
		}

		// If there are no errors, remove the raw file
		err = os.Remove(r.instance.Sink().Name())
	}
	return err
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
	filename, err := getTrackFilename(req.output.LocalID, track.Codec().MimeType)
	if err != nil {
		logger.Warnw("cannot get file name", err, "SID", publication.SID(), "Output", req.output, "Codec", track.Codec().MimeType)
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
		logger.Warnw("cannot create recorder", err, "SID", publication.SID(), "Output", req.output, "Codec", track.Codec().MimeType)
	}
	rec.Start(track)

	// Attach recorder and remove track request from pending list
	b.recorders[publication.SID()] = &recorderInstance{
		output:   req.output,
		codec:    track.Codec(),
		instance: rec,
	}
	delete(b.pending, publication.SID())
}

func (b *bot) OnTrackUnsubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	// Stop recording
	err := b.stopRecording(publication.SID())
	if err != nil {
		logger.Warnw("cannot stop recorder", err, "SID", publication.SID())
	}
}
