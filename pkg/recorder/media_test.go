package recorder

import (
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/stretchr/testify/require"
)

func TestExtensionG722(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypeG722)
	require.Equal(t, mediaOGG, ext)
}

func TestExtensionOpus(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypeOpus)
	require.Equal(t, mediaOGG, ext)
}

func TestExtensionPCMU(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypePCMU)
	require.EqualValues(t, mediaOGG, ext)
}

func TestExtensionPCMA(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypePCMA)
	require.Equal(t, mediaOGG, ext)
}

func TestExtensionVP8(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypeVP8)
	require.Equal(t, mediaIVF, ext)
}

func TestExtensionVP9(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypeVP9)
	require.Equal(t, mediaIVF, ext)
}

func TestExtensionH264(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypeH264)
	require.Equal(t, mediaH264, ext)
}

func TestExtensionH265GetEmptyString(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypeH265)
	require.Equal(t, mediaExtension(""), ext)
}

func TestExtensionAV1GetEmptyString(t *testing.T) {
	ext := getMediaExtension(webrtc.MimeTypeAV1)
	require.Equal(t, mediaExtension(""), ext)
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
	require.ErrorIs(t, err, ErrExtensionInFileID)
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
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeH264,
			Channels: 1,
		},
	}
	w, err := createMediaWriter(&mockSink{}, codec)
	require.NoError(t, err)
	require.Implements(t, (*media.Writer)(nil), w)
}

func TestGetIVFWriter(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8,
			Channels: 1,
		},
	}
	w, err := createMediaWriter(&mockSink{}, codec)
	require.NoError(t, err)
	require.Implements(t, (*media.Writer)(nil), w)
}

func TestGetOGGWriter(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeOpus,
			Channels: 2,
		},
	}
	w, err := createMediaWriter(&mockSink{}, codec)
	require.NoError(t, err)
	require.Implements(t, (*media.Writer)(nil), w)
}

func TestGetUnsupportedWriter(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeAV1,
			Channels: 1,
		},
	}
	_, err := createMediaWriter(&mockSink{}, codec)
	require.ErrorIs(t, ErrMediaNotSupported, err)
}

func TestVP8SampleBuilder(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8,
			Channels: 1,
		},
	}
	sb := createSampleBuilder(codec)
	require.NotNil(t, sb)
}

func TestVP9SampleBuilder(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP9,
			Channels: 1,
		},
	}
	sb := createSampleBuilder(codec)
	require.NotNil(t, sb)
}

func TestH264SampleBuilder(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeH264,
			Channels: 1,
		},
	}
	sb := createSampleBuilder(codec)
	require.NotNil(t, sb)
}

func TestOpusSampleBuilder(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeOpus,
			Channels: 1,
		},
	}
	sb := createSampleBuilder(codec)
	require.NotNil(t, sb)
}

func TestUnsupportedCodecSampleBuilder(t *testing.T) {
	codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeAV1,
			Channels: 1,
		},
	}
	sb := createSampleBuilder(codec)
	require.Nil(t, sb)
}
