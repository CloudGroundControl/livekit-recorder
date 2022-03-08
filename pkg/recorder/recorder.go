package recorder

import (
	"context"
	"errors"
	"time"

	"github.com/livekit/protocol/logger"
	"github.com/livekit/server-sdk-go/pkg/samplebuilder"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type Recorder interface {
	Start(context.Context, *webrtc.TrackRemote)
	Stop() error
	Sink() Sink
}

type recorder struct {
	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}

	sink Sink
	mw   media.Writer
	sb   *samplebuilder.SampleBuilder
}

func New(codec webrtc.RTPCodecParameters, filename string) (Recorder, error) {
	sink, err := NewFileSink(filename)
	if err != nil {
		return nil, err
	}
	return NewWith(codec, sink)
}

func NewWith(codec webrtc.RTPCodecParameters, sink Sink) (Recorder, error) {
	mw, err := createMediaWriter(sink, codec)
	if err != nil {
		return nil, err
	}
	return &recorder{
		closed: make(chan struct{}),
		sink:   sink,
		mw:     mw,
		sb:     createSampleBuilder(codec),
	}, nil
}

func (r *recorder) Start(ctx context.Context, track *webrtc.TrackRemote) {
	// Copy context since it's a good practice
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Start recording in a goroutine
	go r.startRecording(track)
}

const maxWaitDuration = time.Second * 10

var ErrRecorderStopTimeout = errors.New("recorder stop timeout")

func (r *recorder) Stop() error {
	// Signal goroutine to stop
	r.cancel()

	// Wait for goroutine to finish gracefully or timeouts with error
	for {
		select {
		case <-r.closed:
			return nil
		case <-time.After(maxWaitDuration):
			return ErrRecorderStopTimeout
		}
	}
}

func (r *recorder) Sink() Sink {
	return r.sink
}

func (r *recorder) startRecording(track *webrtc.TrackRemote) {
	var err error
	defer func() {
		// Log any errors
		if err != nil {
			logger.Warnw("recorder error", err)
		}

		// Close sink
		err = r.sink.Close()
		if err != nil {
			logger.Warnw("error closing sink", err)
		}

		// Signal recorder has finished cleaning up
		close(r.closed)
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
