package recording

import (
	"context"

	"github.com/livekit/protocol/livekit"
)

type UpdateTrackSubscriptionsRequest struct {
	Room     string
	Identity string
	SIDs     []string
	Subcribe bool
}

func (s *service) updateTrackSubscriptions(ctx context.Context, req UpdateTrackSubscriptionsRequest) error {
	_, err := s.lksvc.UpdateSubscriptions(ctx, &livekit.UpdateSubscriptionsRequest{
		Room:      req.Room,
		Identity:  req.Identity,
		TrackSids: req.SIDs,
		Subscribe: req.Subcribe,
	})
	return err
}
