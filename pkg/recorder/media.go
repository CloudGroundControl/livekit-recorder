package recorder

import (
	"errors"
	"io"
	"strings"

	"github.com/livekit/server-sdk-go/pkg/samplebuilder"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

type MediaExtension string

const (
	MediaOGG  MediaExtension = "ogg"
	MediaIVF  MediaExtension = "ivf"
	MediaH264 MediaExtension = "h264"
)

func GetMediaExtension(mimeType string) MediaExtension {
	if strings.EqualFold(mimeType, webrtc.MimeTypeVP8) ||
		strings.EqualFold(mimeType, webrtc.MimeTypeVP9) {
		return MediaIVF
	}
	if strings.EqualFold(mimeType, webrtc.MimeTypeH264) {
		return MediaH264
	}
	if strings.EqualFold(mimeType, webrtc.MimeTypeOpus) {
		return MediaOGG
	}
	return ""
}

var ErrMediaNotSupported = errors.New("media not supported")

func createMediaWriter(out io.Writer, codec webrtc.RTPCodecParameters) (media.Writer, error) {
	switch GetMediaExtension(codec.MimeType) {
	case MediaIVF:
		return ivfwriter.NewWith(out)
	case MediaH264:
		return h264writer.NewWith(out), nil
	case MediaOGG:
		return oggwriter.NewWith(out, 48000, codec.Channels)
	default:
		return nil, ErrMediaNotSupported
	}
}

const sampleMaxLate = 200

func createSampleBuilder(codec webrtc.RTPCodecParameters, opts ...samplebuilder.Option) *samplebuilder.SampleBuilder {
	var depacketizer rtp.Depacketizer
	switch codec.MimeType {
	case webrtc.MimeTypeVP8:
		depacketizer = &codecs.VP8Packet{}
	case webrtc.MimeTypeVP9:
		depacketizer = &codecs.VP9Packet{}
	case webrtc.MimeTypeH264:
		depacketizer = &codecs.H264Packet{}
	case webrtc.MimeTypeOpus:
		depacketizer = &codecs.OpusPacket{}
	default:
		return nil
	}
	return samplebuilder.New(sampleMaxLate, depacketizer, codec.ClockRate, opts...)
}
