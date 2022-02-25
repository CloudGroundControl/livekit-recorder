package recorder

import (
	"time"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/samplebuilder"
	"github.com/livekit/protocol/logger"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type Recorder interface {
	Start(track *webrtc.TrackRemote)
	Stop()
	Sink() RecorderSink
}

type recorder struct {
	sink   RecorderSink
	done   chan struct{}
	closed chan struct{}
	sb     *samplebuilder.SampleBuilder
	mw     media.Writer
}

func NewTrackRecorder(codec webrtc.RTPCodecParameters, sink RecorderSink) (Recorder, error) {
	done := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)
	sb := createSampleBuilder(codec)
	mw, err := createMediaWriter(sink, codec)
	if err != nil {
		return nil, err
	}
	return &recorder{sink, done, closed, sb, mw}, nil
}

func (r *recorder) Start(track *webrtc.TrackRemote) {
	go r.startRecording(track)
}

func (r *recorder) Stop() {
	go r.stopRecording()
}

func (r *recorder) Sink() RecorderSink {
	return r.sink
}

func (r *recorder) startRecording(track *webrtc.TrackRemote) {
	// Clean-up process
	var err error
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

	// Process RTP packets forever until stopped
	var packet *rtp.Packet
	for {
		select {
		case <-r.done:
			return
		default:
			// Read RTP stream
			packet, _, err = track.ReadRTP()
			if err != nil {
				return
			}

			// Write packet to sink
			err = r.writeToSink(packet)
			if err != nil {
				return
			}
		}
	}
}

func (r *recorder) writeToSink(p *rtp.Packet) (err error) {
	// If no sample buffer is used, write directly to sink
	if r.sb == nil {
		return r.mw.WriteRTP(p)
	}

	// If sample buffer is used, write to buffer first
	r.sb.Push(p)

	// And from the buffered packets, write to sink
	for _, p := range r.sb.PopPackets() {
		err = r.mw.WriteRTP(p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *recorder) stopRecording() {
	// Signal to startRecording() goroutine to end
	close(r.done)

	// Wait for signal from startRecording() after clean-up is done.
	// This function must be called in a goroutine or it'll block main thread
	<-r.closed

	// Introduce a small delay otherwise the end frame of a video track will look chopped on the sides
	time.Sleep(time.Millisecond * 10)
}
