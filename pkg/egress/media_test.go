package egress

import (
	"testing"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

func TestGetTrackFilenameSuccess(t *testing.T) {
	mimeType := webrtc.MimeTypeVP8
	fileID := "test"
	expected := "test.ivf"
	actual, err := getTrackFilename(fileID, mimeType)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestGetTrackFilenameFailEmptyFileID(t *testing.T) {
	mimeType := webrtc.MimeTypeH264
	fileID := ""
	_, err := getTrackFilename(fileID, mimeType)
	require.ErrorIs(t, err, ErrEmptyFileID)
}

func TestGetTrackFilenameFailFileIDContainsExtension(t *testing.T) {
	mimeType := webrtc.MimeTypeAV1
	fileID := "test.ivf"
	_, err := getTrackFilename(fileID, mimeType)
	require.ErrorIs(t, err, ErrExtensionInFileID)
}

func TestGetTrackFilenameFailUnsupportedMedia(t *testing.T) {
	mimeType := webrtc.MimeTypeAV1
	fileID := "test"
	_, err := getTrackFilename(fileID, mimeType)
	require.ErrorIs(t, err, recorder.ErrMediaNotSupported)
}
