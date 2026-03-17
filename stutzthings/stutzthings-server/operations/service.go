package operations

import (
	"context"
	"encoding/json"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
)

type QueryStore interface {
	QueryRows(ctx context.Context, query string) ([]map[string]any, error)
}

type TopicPublisher interface {
	Publish(ctx context.Context, topic string, qos byte, retained bool, payload []byte) error
}

type Service struct {
	store     QueryStore
	publisher TopicPublisher
	signer    *auth.Signer
}

func NewService(store QueryStore, publisher TopicPublisher, signer *auth.Signer) *Service {
	return &Service{store: store, publisher: publisher, signer: signer}
}

func (s *Service) ensureReadAuthorized(claims *auth.Claims, prefix []string) bool {
	if claims == nil {
		return false
	}
	return auth.AnyFilterCanMatchPrefix(claims.Publ, prefix, telemetryTopicSegments) || auth.AnyFilterCanMatchPrefix(claims.Subs, prefix, telemetryTopicSegments)
}

func (s *Service) ensureCommandAuthorized(claims *auth.Claims, topic string) bool {
	if claims == nil {
		return false
	}
	return auth.AnyMatch(claims.Subs, topic)
}

func (s *Service) ensureRegistrationAuthorized(claims *auth.Claims, accountID string, deviceID string) bool {
	if claims == nil {
		return false
	}
	prefix := []string{accountID, deviceID}
	return auth.AnyFilterCanMatchPrefix(claims.Publ, prefix, telemetryTopicSegments) && auth.AnyFilterCanMatchPrefix(claims.Subs, prefix, commandTopicSegments)
}

func marshalDesiredStatePayload(value any) ([]byte, error) {
	if value == nil {
		return nil, invalidInput("value is required", nil)
	}
	payload, err := json.Marshal(map[string]any{"value": value})
	if err != nil {
		return nil, invalidInput("value must be JSON serializable", err)
	}
	return payload, nil
}
