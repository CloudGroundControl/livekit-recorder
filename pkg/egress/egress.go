package egress

import (
	"context"
)

type Service interface {
	StartRecording(ctx context.Context, req StartRecordingRequest) error
	StopRecording(ctx context.Context, req StopRecordingRequest) error
}

type service struct {
}

func NewService() Service {
	return &service{}
}

func (s *service) StartRecording(ctx context.Context, req StartRecordingRequest) error {
	return nil
}

func (s *service) StopRecording(ctx context.Context, req StopRecordingRequest) error {
	return nil
}
