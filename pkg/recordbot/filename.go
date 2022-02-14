package recordbot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pion/webrtc/v3"
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
