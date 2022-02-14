package recordbot

import (
	"github.com/cloudgroundcontrol/livekit-recordbot/pkg/samplebuilder"
	"github.com/livekit/protocol/logger"
	"github.com/pion/rtp/codecs"
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
	sb     *samplebuilder.SampleBuilder
}

type RecorderHooks struct {
	OnFinishRecording func(filename string)
}

const (
	maxLate = 200
)

func NewTrackRecorder(track *webrtc.TrackRemote, sink RecorderSink, hooks *RecorderHooks) (Recorder, error) {
	done := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)

	var builder *samplebuilder.SampleBuilder
	switch track.Codec().MimeType {
	case webrtc.MimeTypeVP8:
		builder = samplebuilder.New(maxLate, &codecs.VP8Packet{}, track.Codec().ClockRate)
	case webrtc.MimeTypeVP9:
		builder = samplebuilder.New(maxLate, &codecs.VP9Packet{}, track.Codec().ClockRate)
	case webrtc.MimeTypeH264:
		builder = samplebuilder.New(maxLate, &codecs.H264Packet{}, track.Codec().ClockRate)
	case webrtc.MimeTypeOpus:
		builder = samplebuilder.New(maxLate, &codecs.OpusPacket{}, track.Codec().ClockRate)
	default:
		return nil, ErrMediaNotSupported
	}
	return &recorder{sink, hooks, done, closed, builder}, nil
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
	writer, err := createMediaWriter(r.sink, track.Codec())
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

			// Push packet to sample buffer
			r.sb.Push(packet)

			// Write the buffered packets to sink
			for _, p := range r.sb.PopPackets() {
				err = writer.WriteRTP(p)
				if err != nil {
					return
				}
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
