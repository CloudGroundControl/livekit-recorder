package recordbot

import (
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/stretchr/testify/require"
)

func TestExtensionG722(t *testing.T) {
	ext, err := getMediaExtension(webrtc.MimeTypeG722)
	require.NoError(t, err)
	require.Equal(t, mediaOGG, ext)
}

func TestExtensionOpus(t *testing.T) {
	ext, err := getMediaExtension(webrtc.MimeTypeOpus)
	require.NoError(t, err)
	require.Equal(t, mediaOGG, ext)
}

func TestExtensionPCMU(t *testing.T) {
	ext, err := getMediaExtension(webrtc.MimeTypePCMU)
	require.NoError(t, err)
	require.Equal(t, mediaOGG, ext)
}

func TestExtensionPCMA(t *testing.T) {
	ext, err := getMediaExtension(webrtc.MimeTypePCMA)
	require.NoError(t, err)
	require.Equal(t, mediaOGG, ext)
}

func TestExtensionVP8(t *testing.T) {
	ext, err := getMediaExtension(webrtc.MimeTypeVP8)
	require.NoError(t, err)
	require.Equal(t, mediaIVF, ext)
}

func TestExtensionVP9(t *testing.T) {
	ext, err := getMediaExtension(webrtc.MimeTypeVP9)
	require.NoError(t, err)
	require.Equal(t, mediaIVF, ext)
}

func TestExtensionH264(t *testing.T) {
	ext, err := getMediaExtension(webrtc.MimeTypeH264)
	require.NoError(t, err)
	require.Equal(t, mediaH264, ext)
}

func TestExtensionH265Fail(t *testing.T) {
	_, err := getMediaExtension(webrtc.MimeTypeH265)
	require.ErrorIs(t, err, ErrMediaNotSupported)
}

func TestExtensionAV1Fail(t *testing.T) {
	_, err := getMediaExtension(webrtc.MimeTypeAV1)
	require.ErrorIs(t, err, ErrMediaNotSupported)
}

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
	require.ErrorIs(t, err, ErrFileIDContainsExtension)
}

func TestGetFilenameFailUnsupportedMedia(t *testing.T) {
	mimeType := webrtc.MimeTypeAV1
	fileID := "test"
	_, err := getMediaFilename(fileID, mimeType)
	require.ErrorIs(t, err, ErrMediaNotSupported)
}

type mockSink struct{}

func (m *mockSink) Name() string {
	return "mock"
}

func (m *mockSink) Write(data []byte) (int, error) {
	return 1, nil
}

func (m *mockSink) Close() error {
	return nil
}

func TestGetH264Writer(t *testing.T) {
	mimeType := webrtc.MimeTypeH264
	w, err := createMediaWriter(&mockSink{}, mimeType)
	require.NoError(t, err)
	require.Implements(t, (*media.Writer)(nil), w)
}

func TestGetIVFWriter(t *testing.T) {
	mimeType := webrtc.MimeTypeVP8
	w, err := createMediaWriter(&mockSink{}, mimeType)
	require.NoError(t, err)
	require.Implements(t, (*media.Writer)(nil), w)
}

func TestGetOGGWriter(t *testing.T) {
	mimeType := webrtc.MimeTypeOpus
	w, err := createMediaWriter(&mockSink{}, mimeType)
	require.NoError(t, err)
	require.Implements(t, (*media.Writer)(nil), w)
}

func TestGetUnsupportedWriter(t *testing.T) {
	mimeType := webrtc.MimeTypeAV1
	_, err := createMediaWriter(&mockSink{}, mimeType)
	require.ErrorIs(t, ErrMediaNotSupported, err)
}
