package egress

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
)

var (
	ErrEmptyFileID       = errors.New("empty file ID")
	ErrExtensionInFileID = errors.New("extension in file ID")
)

func getMediaFilename(fileID string, mimeType string) (string, error) {
	if fileID == "" {
		return "", ErrEmptyFileID
	} else if strings.Contains(fileID, ".") {
		return "", ErrExtensionInFileID
	}

	ext := recorder.GetMediaExtension(mimeType)
	if ext == "" {
		return "", recorder.ErrMediaNotSupported
	}

	return fmt.Sprintf("%s.%s", fileID, ext), nil
}
