package recorder

import (
	"errors"
	"io"
	"strings"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/samplebuilder"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

type mediaExtension string

const (
	mediaOGG  mediaExtension = "ogg"
	mediaIVF  mediaExtension = "ivf"
	mediaH264 mediaExtension = "h264"
)

func GetMediaExtension(mimeType string) mediaExtension {
	if strings.EqualFold(mimeType, webrtc.MimeTypeVP8) ||
		strings.EqualFold(mimeType, webrtc.MimeTypeVP9) {
		return mediaIVF
	}
	if strings.EqualFold(mimeType, webrtc.MimeTypeH264) {
		return mediaH264
	}
	if strings.EqualFold(mimeType, webrtc.MimeTypeG722) ||
		strings.EqualFold(mimeType, webrtc.MimeTypeOpus) ||
		strings.EqualFold(mimeType, webrtc.MimeTypePCMA) ||
		strings.EqualFold(mimeType, webrtc.MimeTypePCMU) {
		return mediaOGG
	}
	return ""
}

var ErrMediaNotSupported = errors.New("media not supported")

func createMediaWriter(out io.Writer, codec webrtc.RTPCodecParameters) (media.Writer, error) {
	switch GetMediaExtension(codec.MimeType) {
	case mediaIVF:
		return ivfwriter.NewWith(out)
	case mediaH264:
		return h264writer.NewWith(out), nil
	case mediaOGG:
		return oggwriter.NewWith(out, 48000, codec.Channels)
	default:
		return nil, ErrMediaNotSupported
	}
}

const sampleMaxLate = 200

func createSampleBuilder(codec webrtc.RTPCodecParameters) *samplebuilder.SampleBuilder {
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
	return samplebuilder.New(sampleMaxLate, depacketizer, codec.ClockRate)
}
