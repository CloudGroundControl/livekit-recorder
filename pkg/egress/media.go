package egress

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
	"github.com/pion/webrtc/v3"
)

var (
	ErrEmptyFileID       = errors.New("empty file ID")
	ErrExtensionInFileID = errors.New("extension in file ID")
)

func getTrackFilename(fileID string, mimeType string) (string, error) {
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

// ------
// Containerisation
// ------

func isContainerisable(codec webrtc.RTPCodecParameters) bool {
	switch recorder.GetMediaExtension(codec.MimeType) {
	case recorder.MediaIVF:
		return true
	case recorder.MediaH264:
		return true
	default:
		return false
	}
}

var (
	ErrContainerTypeNotSupported    = errors.New("container type not supporated")
	ErrInputFilenameHasMultipleDots = errors.New("input file name has multiple dots")
)

func getContainerFilename(filename string) (string, error) {
	// Split into file ID and extension
	tokens := strings.Split(filename, ".")
	if len(tokens) != 2 {
		return "", ErrInputFilenameHasMultipleDots
	}
	fileID := tokens[0]
	fileExt := tokens[1]

	// Detect file extension and return the appropriate container file name
	switch fileExt {
	case string(recorder.MediaH264):
		return fmt.Sprintf("%s.%s", fileID, "mp4"), nil
	case string(recorder.MediaIVF):
		return fmt.Sprintf("%s.%s", fileID, "webm"), nil
	default:
		return "", ErrContainerTypeNotSupported
	}
}

func containeriseFile(input string) error {
	output, err := getContainerFilename(input)
	if err != nil {
		return err
	}

	// Containerise file (only)
	cmd := exec.Command("ffmpeg",
		"-i", input,
		"-c:v", "copy",
		"-loglevel", "error", "-y",
		output,
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
