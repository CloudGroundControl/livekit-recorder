package recordbot

import (
	"github.com/livekit/protocol/logger"
	"github.com/pion/webrtc/v3"
)

type Recorder interface {
	Start(track *webrtc.TrackRemote)
	Stop()
}

type recorder struct {
	sink   RecorderSink
	hooks  *RecorderHooks
	done   chan struct{}
	closed chan struct{}
}

type RecorderHooks struct {
	OnFinishRecording func(filename string)
}

func NewTrackRecorder(sink RecorderSink, hooks *RecorderHooks) (Recorder, error) {
	done := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)
	return &recorder{sink, hooks, done, closed}, nil
}

func (r *recorder) Start(track *webrtc.TrackRemote) {
	go r.startRecording(track)
}

func (r *recorder) Stop() {
	go r.stopRecording()
}

func (r *recorder) startRecording(track *webrtc.TrackRemote) {
	var err error

	// If we can't instantiate media writer, stop
	writer, err := createMediaWriter(r.sink, track.Codec().MimeType)
	if err != nil {
		logger.Warnw("cannot create media writer", err, "codec", track.Codec().MimeType)
		return
	}

	// Clean-up process
	defer func() {
		// Log error during recording
		if err != nil {
			logger.Warnw("error while recording", err)
		}

		// Close sink
		err = r.sink.Close()
		if err != nil {
			logger.Warnw("cannot close sink", err)
		}

		// Signal that sink has been closed
		close(r.closed)
	}()

	for {
		select {
		case <-r.done:
			return
		default:
			// Read RTP stream
			packet, _, err := track.ReadRTP()
			if err != nil {
				return
			}

			// Write to sink
			err = writer.WriteRTP(packet)
			if err != nil {
				return
			}
		}
	}

}

func (r *recorder) stopRecording() {
	// Signal to startRecording() goroutine to end
	close(r.done)

	// Wait for signal from startRecording() after clean-up is done.
	// This function must be called in a goroutine or it'll block main thread
	<-r.closed
}
