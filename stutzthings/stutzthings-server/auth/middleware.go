package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type claimsContextKey struct{}

type BearerAuthenticator struct {
	secret []byte
}

func NewBearerAuthenticator(secret []byte) BearerAuthenticator {
	return BearerAuthenticator{secret: secret}
}

func (a BearerAuthenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractBearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}

		claims, err := ParseAndValidateToken(token, a.secret)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}

		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
	})
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(*Claims)
	return claims, ok
}

func ExtractClaimsFromHeader(headerValue string, secret []byte) (*Claims, error) {
	token, ok := extractBearerToken(headerValue)
	if !ok {
		return nil, ErrInvalidToken
	}
	return ParseAndValidateToken(token, secret)
}

func extractBearerToken(headerValue string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(headerValue))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

func writeAuthError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
