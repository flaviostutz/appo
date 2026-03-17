package operations

import (
	"context"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
)

func (s *Service) SetDesiredState(ctx context.Context, claims *auth.Claims, identity DeviceIdentity, request DesiredStateRequest) (*DesiredStateResponse, error) {
	if err := ValidateDeviceIdentity(identity); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, unauthorized("missing auth context", nil)
	}
	payload, err := marshalDesiredStatePayload(request.Value)
	if err != nil {
		return nil, err
	}
	topic := identity.CommandTopic()
	if !s.ensureCommandAuthorized(claims, topic) {
		return nil, forbidden("token does not authorize the requested desired-state topic")
	}
	if err := s.publisher.Publish(ctx, topic, 1, false, payload); err != nil {
		return nil, backendFailure("publish desired state", err)
	}
	return &DesiredStateResponse{Accepted: true, Topic: topic}, nil
}
