package recorder

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cloudgroundcontrol/livekit-egress/pkg/recorder/internal/static"
	"github.com/livekit/protocol/auth"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
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
	require.NotNil(t, rec.sb)

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
	require.Nil(t, rec.sb)

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

func getEnvOrFail(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("%s not set", key)
	}
	return val
}

func TestRecorderUsageScenario(t *testing.T) {
	url := getEnvOrFail("LIVEKIT_URL")
	apiKey := getEnvOrFail("LIVEKIT_API_KEY")
	apiSecret := getEnvOrFail("LIVEKIT_API_SECRET")
	testRoomName := "livekit-egress-test"
	TRUE := true
	FALSE := false
	participantID := "lk-participant"
	recorderID := "lk-recorder"

	// -----
	// Generate access tokens for participant and recorder
	// -----

	pAT := auth.NewAccessToken(apiKey, apiSecret)
	pGrant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         testRoomName,
		CanPublish:   &TRUE,
		CanSubscribe: &TRUE,
	}
	pAT.AddGrant(pGrant).SetIdentity(participantID).SetValidFor(time.Hour)
	pToken, err := pAT.ToJWT()
	require.NoError(t, err)

	rAT := auth.NewAccessToken(apiKey, apiSecret)
	rGrant := &auth.VideoGrant{
		RoomJoin:       true,
		Room:           testRoomName,
		CanPublish:     &FALSE,
		CanPublishData: &FALSE,
		CanSubscribe:   &TRUE,
		Hidden:         true,
		Recorder:       true,
	}
	rAT.AddGrant(rGrant).SetIdentity(recorderID).SetValidFor(time.Hour)
	rToken, err := rAT.ToJWT()
	require.NoError(t, err)

	// -----
	// Connect to room
	// -----

	var pRoom, rRoom *lksdk.Room

	pRoom, err = lksdk.ConnectToRoomWithToken(url, pToken)
	require.NoError(t, err)

	rRoom, err = lksdk.ConnectToRoomWithToken(url, rToken)
	require.NoError(t, err)

	// -----
	// Create track for participant to publish
	// -----

	preferredCodec := webrtc.MimeTypeVP8

	sampleTrack, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
		MimeType:  preferredCodec,
		ClockRate: 90000,
		Channels:  1,
	})
	require.NoError(t, err)

	pRoom.LocalParticipant.PublishTrack(sampleTrack, participantID+"-video")

	// -----
	// Publish static media packets until we receive stop signal
	// -----

	sampleProvider := static.NewProvider()
	sampleDone := make(chan struct{}, 1)
	go func() {
		var sample media.Sample
		for {
			select {
			case <-sampleDone:
				return
			default:
				sample, err = sampleProvider.NextSample()
				require.NoError(t, err)
				sampleTrack.WriteSample(sample, nil)
			}
		}
	}()

	// -----
	// Create recorder since we know the codec we'll be publishing
	// -----

	sink := NewBufferSink(participantID + "-video")
	rec, err := NewTrackRecorder(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8,
		},
	}, sink)
	require.NoError(t, err)

	rRoom.Callback.OnTrackSubscribed = func(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		rec.Start(track)
	}

	// -----
	// Let participant publish track for a while,
	// then stop publishing and recording
	// -----
	time.Sleep(time.Second * 2)
	sampleDone <- struct{}{}
	rec.Stop()
}
