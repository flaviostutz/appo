package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/google/uuid"
)

func (s *Service) RegisterDeviceInstance(ctx context.Context, claims *auth.Claims, request RegistrationRequest) (*RegistrationResponse, error) {
	if err := ValidateRegistrationRequest(request); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, unauthorized("missing auth context", nil)
	}
	if !s.ensureRegistrationAuthorized(claims, request.AccountID, request.DeviceID) {
		return nil, forbidden("token does not authorize registration for the requested namespace")
	}
	if s.signer == nil {
		return nil, backendFailure("registration signer is not configured", nil)
	}
	deviceInstanceID, err := s.generateDeviceInstanceID(ctx, request)
	if err != nil {
		return nil, err
	}
	sub := fmt.Sprintf("%s/%s/%s", request.AccountID, request.DeviceID, deviceInstanceID)
	publ := []string{fmt.Sprintf("%s/%s/%s/+/+", request.AccountID, request.DeviceID, deviceInstanceID)}
	subs := []string{fmt.Sprintf("%s/%s/%s/+/+/set", request.AccountID, request.DeviceID, deviceInstanceID)}
	token, signedClaims, err := s.signer.Sign(sub, publ, subs, time.Now().UTC())
	if err != nil {
		return nil, backendFailure("sign registration token", err)
	}
	return &RegistrationResponse{
		AccountID:        request.AccountID,
		DeviceID:         request.DeviceID,
		DeviceInstanceID: deviceInstanceID,
		Sub:              sub,
		Publ:             publ,
		Subs:             subs,
		Iss:              signedClaims.Issuer,
		IssuedAt:         signedClaims.IssuedAt.UTC(),
		ExpiresAt:        signedClaims.ExpiresAt.UTC(),
		Token:            token,
	}, nil
}

func (s *Service) generateDeviceInstanceID(ctx context.Context, request RegistrationRequest) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate, err := uuid.NewV7()
		if err != nil {
			return "", backendFailure("generate device instance id", err)
		}
		rows, err := s.store.QueryRows(ctx, buildRegistrationCollisionQuery(request.AccountID, request.DeviceID, candidate.String()))
		if err != nil {
			return "", backendFailure("check device instance collision", err)
		}
		if len(rows) == 0 {
			return candidate.String(), nil
		}
	}
	return "", backendFailure("generate unique device instance id", nil)
}
