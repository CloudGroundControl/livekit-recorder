package static

import (
	"time"

	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/webrtc/v3/pkg/media"
)

type provider struct{}

func NewProvider() lksdk.SampleProvider {
	return &provider{}
}

const stringSample = "hello world"

func (p *provider) NextSample() (media.Sample, error) {
	return media.Sample{
		Data:      []byte(stringSample),
		Timestamp: time.Now().Add(-time.Millisecond),
		Duration:  time.Millisecond,
	}, nil
}

func (p *provider) OnBind() error {
	return nil
}

func (p *provider) OnUnbind() error {
	return nil
}
