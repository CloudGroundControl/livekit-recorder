package recorder

import (
	"fmt"
	"testing"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

func mockPacket(id uint16, p []byte) *rtp.Packet {
	return &rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: id,
		},
		Payload: p,
	}
}

func promoteRecorder(r Recorder) *recorder {
	rec, ok := r.(*recorder)
	if !ok {
		panic("cannot promote Recorder to *recorder")
	}
	return rec
}

func TestCreateRecorderForVideo(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8,
		},
	}
	sink := NewBufferSink("test")

	tr, err := NewTrackRecorder(codec, sink)
	require.NoError(t, err)
	require.NotNil(t, tr)

	rec := promoteRecorder(tr)
	require.NotNil(t, rec.sb)
	require.NotNil(t, rec.mw)
}

func TestCreateRecorderForAudio(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeOpus,
			Channels: 2,
		},
	}
	sink := NewBufferSink("test")

	tr, err := NewTrackRecorder(codec, sink)
	require.NoError(t, err)
	require.NotNil(t, tr)

	rec := promoteRecorder(tr)
	require.NotNil(t, rec.sb)
	require.NotNil(t, rec.mw)
}

func TestFailCreateRecorderForUnsupportedCodec(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeAV1,
		},
	}
	sink := NewBufferSink("test")
	_, err := NewTrackRecorder(codec, sink)
	require.ErrorIs(t, err, ErrMediaNotSupported)
}

func TestWritePacketsWithSampleBuffer(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			// Choose between VP8, VP9, H264 and Opus
			MimeType: webrtc.MimeTypeH264,
			Channels: 2,
		},
	}
	sink := NewBufferSink("test")
	tr, _ := NewTrackRecorder(codec, sink)
	rec := promoteRecorder(tr)

	// Write multiple packets
	for i := 0; i < 10; i++ {
		payload := []byte(fmt.Sprintf("Hello World %d\n!", i))
		packet := mockPacket(uint16(i), payload)
		err := rec.writeToSink(packet)
		require.NoError(t, err)
	}
}

func TestWritePacketsWithoutSampleBuffer(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			// G722 does not have sample buffer depacketizer,
			// so it won't spawn sample buffer
			MimeType: webrtc.MimeTypeG722,
			Channels: 1,
		},
	}
	sink := NewBufferSink("test")
	tr, _ := NewTrackRecorder(codec, sink)
	rec := promoteRecorder(tr)

	// Write multiple packets
	for i := 0; i < 10; i++ {
		payload := []byte(fmt.Sprintf("Hello World %d\n!", i))
		packet := mockPacket(uint16(i), payload)
		err := rec.writeToSink(packet)
		require.NoError(t, err)
	}
}

func TestStopRecordingWithoutStart(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8,
		},
	}
	sink := NewBufferSink("test")
	tr, _ := NewTrackRecorder(codec, sink)
	rec := promoteRecorder(tr)

	go func() {
		// Trigger stop signal
		rec.Stop()

		// Expect `done` to be closed
		_, ok := (<-rec.done)
		require.False(t, ok)

		// Expect `closed` to still be open since we're stopping recording
		// without starting, so the goroutine to close `rec.closed` is not called
		_, ok = (<-rec.closed)
		require.True(t, ok)
	}()
}
