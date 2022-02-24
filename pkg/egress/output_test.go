package egress

import (
	"testing"

	"github.com/cloudgroundcontrol/livekit-recorder/pkg/recorder"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

func TestGetFilenameSuccess(t *testing.T) {
	mimeType := webrtc.MimeTypeVP8
	fileID := "test"
	expected := "test.ivf"
	actual, err := getMediaFilename(fileID, mimeType)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestGetFilenameFailEmptyFileID(t *testing.T) {
	mimeType := webrtc.MimeTypeH264
	fileID := ""
	_, err := getMediaFilename(fileID, mimeType)
	require.ErrorIs(t, err, ErrEmptyFileID)
}

func TestGetFilenameFailFileIDContainsExtension(t *testing.T) {
	mimeType := webrtc.MimeTypeAV1
	fileID := "test.ivf"
	_, err := getMediaFilename(fileID, mimeType)
	require.ErrorIs(t, err, ErrExtensionInFileID)
}

func TestGetFilenameFailUnsupportedMedia(t *testing.T) {
	mimeType := webrtc.MimeTypeAV1
	fileID := "test"
	_, err := getMediaFilename(fileID, mimeType)
	require.ErrorIs(t, err, recorder.ErrMediaNotSupported)
}
