package operations

import (
	"context"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
)

func (s *Service) GetAttributeState(ctx context.Context, claims *auth.Claims, identity DeviceIdentity) (*AttributeStateResponse, error) {
	if err := ValidateDeviceIdentity(identity); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, unauthorized("missing auth context", nil)
	}
	if !auth.AnyMatch(claims.Publ, identity.TelemetryTopic()) && !auth.AnyMatch(claims.Subs, identity.TelemetryTopic()) {
		return nil, forbidden("token does not authorize the requested attribute")
	}
	rows, err := s.store.QueryRows(ctx, buildLatestAttributeQuery(identity))
	if err != nil {
		return nil, backendFailure("query latest attribute state", err)
	}
	if len(rows) == 0 {
		return &AttributeStateResponse{Found: false, Observation: nil}, nil
	}
	observation, err := decodeObservationRow(rows[0])
	if err != nil {
		return nil, backendFailure("decode latest attribute state", err)
	}
	return &AttributeStateResponse{Found: true, Observation: &observation}, nil
}

func (s *Service) GetDeviceState(ctx context.Context, claims *auth.Claims, scope DeviceScope) (*DeviceStateResponse, error) {
	if err := ValidateDeviceScope(scope); err != nil {
		return nil, err
	}
	if !s.ensureReadAuthorized(claims, scope.TelemetryPrefix()) {
		return nil, forbidden("token does not authorize the requested device scope")
	}
	rows, err := s.store.QueryRows(ctx, buildLatestDeviceQuery(scope))
	if err != nil {
		return nil, backendFailure("query device snapshot", err)
	}
	response := &DeviceStateResponse{
		AccountID:        scope.AccountID,
		DeviceID:         scope.DeviceID,
		DeviceInstanceID: scope.DeviceInstanceID,
		Attributes:       []Observation{},
	}
	seen := map[string]struct{}{}
	for _, row := range rows {
		observation, err := decodeObservationRow(row)
		if err != nil {
			return nil, backendFailure("decode device snapshot", err)
		}
		key := observation.NodeName + "/" + observation.AttributeName
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		topic := DeviceIdentity{
			DeviceScope:   DeviceScope{AccountID: observation.AccountID, DeviceID: observation.DeviceID, DeviceInstanceID: observation.DeviceInstanceID},
			NodeName:      observation.NodeName,
			AttributeName: observation.AttributeName,
		}.TelemetryTopic()
		if auth.AnyMatch(claims.Publ, topic) || auth.AnyMatch(claims.Subs, topic) {
			response.Attributes = append(response.Attributes, observation)
			continue
		}
		response.ExcludedAttributeCount++
	}
	response.IsPartial = response.ExcludedAttributeCount > 0
	return response, nil
}

func (s *Service) GetNodeHistory(ctx context.Context, claims *auth.Claims, request NodeHistoryRequest) (*HistoryResponse, error) {
	if err := ValidateDeviceScope(request.DeviceScope); err != nil {
		return nil, err
	}
	if err := validateSegment("nodeName", request.NodeName); err != nil {
		return nil, err
	}
	if !s.ensureReadAuthorized(claims, []string{request.AccountID, request.DeviceID, request.DeviceInstanceID, request.NodeName}) {
		return nil, forbidden("token does not authorize the requested node scope")
	}
	rows, err := s.store.QueryRows(ctx, buildNodeHistoryQuery(request, historyResultLimit+1))
	if err != nil {
		return nil, backendFailure("query node history", err)
	}
	if len(rows) > historyResultLimit {
		return nil, windowTooLarge("history window would return more than 10000 observations")
	}
	observations := make([]Observation, 0, len(rows))
	for _, row := range rows {
		observation, err := decodeObservationRow(row)
		if err != nil {
			return nil, backendFailure("decode node history", err)
		}
		topic := DeviceIdentity{
			DeviceScope:   DeviceScope{AccountID: observation.AccountID, DeviceID: observation.DeviceID, DeviceInstanceID: observation.DeviceInstanceID},
			NodeName:      observation.NodeName,
			AttributeName: observation.AttributeName,
		}.TelemetryTopic()
		if auth.AnyMatch(claims.Publ, topic) || auth.AnyMatch(claims.Subs, topic) {
			observations = append(observations, observation)
		}
	}
	sortObservations(observations)
	return &HistoryResponse{From: request.From, To: request.To, Observations: observations}, nil
}

func (s *Service) GetAttributeHistory(ctx context.Context, claims *auth.Claims, request AttributeHistoryRequest) (*HistoryResponse, error) {
	if err := ValidateDeviceIdentity(request.DeviceIdentity); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, unauthorized("missing auth context", nil)
	}
	if !auth.AnyMatch(claims.Publ, request.TelemetryTopic()) && !auth.AnyMatch(claims.Subs, request.TelemetryTopic()) {
		return nil, forbidden("token does not authorize the requested attribute history")
	}
	rows, err := s.store.QueryRows(ctx, buildAttributeHistoryQuery(request, historyResultLimit+1))
	if err != nil {
		return nil, backendFailure("query attribute history", err)
	}
	if len(rows) > historyResultLimit {
		return nil, windowTooLarge("history window would return more than 10000 observations")
	}
	observations := make([]Observation, 0, len(rows))
	for _, row := range rows {
		observation, err := decodeObservationRow(row)
		if err != nil {
			return nil, backendFailure("decode attribute history", err)
		}
		observations = append(observations, observation)
	}
	sortObservations(observations)
	return &HistoryResponse{From: request.From, To: request.To, Observations: observations}, nil
}
