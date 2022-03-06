package recorder

import (
	"context"
	"io"

	"github.com/livekit/protocol/logger"
	"github.com/livekit/server-sdk-go/pkg/samplebuilder"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type Recorder interface {
	Start(track *webrtc.TrackRemote)
	Stop() error
	Sink() Sink
}

type recorder struct {
	ctx    context.Context
	cancel context.CancelFunc
	sink   Sink
	sb     *samplebuilder.SampleBuilder
	mw     media.Writer
}

func NewRecorder(codec webrtc.RTPCodecParameters, sink Sink) (Recorder, error) {
	sb := createSampleBuilder(codec)
	mw, err := createMediaWriter(sink, codec)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.TODO())
	return &recorder{ctx, cancel, sink, sb, mw}, nil
}

func (r *recorder) Start(track *webrtc.TrackRemote) {
	go r.startRecording(track)
}

func (r *recorder) Stop() error {
	// Signal goroutine to stop
	r.cancel()

	// Close the file synchronously.
	return r.sink.Close()
}

func (r *recorder) Sink() Sink {
	return r.sink
}

func (r *recorder) startRecording(track *webrtc.TrackRemote) {
	var err error
	defer func() {
		// Ignore EOF and sink closed errors
		if err == io.EOF || err == ErrSinkClosed {
			err = nil
		}

		// Log any errors
		if err != nil {
			logger.Warnw("recorder error", err)
		}
	}()

	// Process RTP packets forever until stopped
	var packet *rtp.Packet
	for {
		select {
		case <-r.ctx.Done():
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
	if packets := r.sb.PopPackets(); packets != nil {
		for _, p := range packets {
			err = r.mw.WriteRTP(p)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
