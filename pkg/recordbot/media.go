package recordbot

import (
	"errors"
	"fmt"
	"io"
	"strings"

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

var ErrMediaNotSupported = errors.New("media not supported")

func getMediaExtension(mimeType string) (mediaExtension, error) {
	if strings.EqualFold(mimeType, webrtc.MimeTypeVP8) ||
		strings.EqualFold(mimeType, webrtc.MimeTypeVP9) {
		return mediaIVF, nil
	}
	if strings.EqualFold(mimeType, webrtc.MimeTypeH264) {
		return mediaH264, nil
	}
	if strings.EqualFold(mimeType, webrtc.MimeTypeG722) ||
		strings.EqualFold(mimeType, webrtc.MimeTypeOpus) ||
		strings.EqualFold(mimeType, webrtc.MimeTypePCMA) ||
		strings.EqualFold(mimeType, webrtc.MimeTypePCMU) {
		return mediaOGG, nil
	}
	return "", ErrMediaNotSupported
}

var ErrEmptyFileID = errors.New("empty file ID")
var ErrFileIDContainsExtension = errors.New("file ID contains extension")

func getMediaFilename(fileID string, mimeType string) (string, error) {
	if fileID == "" {
		return "", ErrEmptyFileID
	} else if strings.Contains(fileID, ".") {
		return "", ErrFileIDContainsExtension
	}

	ext, err := getMediaExtension(mimeType)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", fileID, ext), nil
}

func createMediaWriter(out io.Writer, codec webrtc.RTPCodecParameters) (media.Writer, error) {
	ext, err := getMediaExtension(codec.MimeType)
	if err != nil {
		return nil, err
	}
	switch ext {
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
