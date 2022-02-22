package egress

import (
	"time"

	"github.com/livekit/protocol/auth"
)

type authProvider struct {
	APIKey    string
	APISecret string
}

func (p *authProvider) buildEmptyToken(room string, identity string) (string, error) {
	at := auth.NewAccessToken(p.APIKey, p.APISecret)
	f := false
	grant := &auth.VideoGrant{
		Room:           room,
		RoomJoin:       true,
		CanPublish:     &f,
		CanPublishData: &f,
		CanSubscribe:   &f,
		Hidden:         true,
		Recorder:       true,
	}
	return at.
		AddGrant(grant).
		SetIdentity(identity).
		SetValidFor(time.Hour).
		ToJWT()
}
