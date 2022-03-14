package recording

import "errors"

type MediaProfile string

const (
	MediaVideoOnly MediaProfile = "video"
	MediaAudioOnly MediaProfile = "audio"
	MediaMuxedAV   MediaProfile = "av"
)

var ErrUnknownMediaProfile = errors.New("unknown media profile")

func ParseMediaProfile(p string) (MediaProfile, error) {
	var profile MediaProfile = ""
	var err error = nil

	switch p {
	case string(MediaVideoOnly):
		profile = MediaVideoOnly
	case string(MediaAudioOnly):
		profile = MediaAudioOnly
	case string(MediaMuxedAV):
		profile = MediaMuxedAV
	default:
		err = ErrUnknownMediaProfile
	}

	return profile, err
}
