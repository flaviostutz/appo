package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Publ []string `json:"publ"`
	Subs []string `json:"subs"`
	jwt.RegisteredClaims
}

func (c *Claims) Validate() error {
	if strings.TrimSpace(c.Subject) == "" {
		return fmt.Errorf("%w: missing sub claim", ErrInvalidToken)
	}
	if c.ExpiresAt == nil {
		return fmt.Errorf("%w: missing exp claim", ErrInvalidToken)
	}
	if len(c.Publ) == 0 && len(c.Subs) == 0 {
		return fmt.Errorf("%w: missing publ and subs claims", ErrInvalidToken)
	}
	return nil
}

type Signer struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func DecodeBase64Secret(encoded string) ([]byte, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, fmt.Errorf("jwt signing secret is required")
	}
	secret, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("decode jwt signing secret: %w", err)
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("jwt signing secret must not be empty")
	}
	return secret, nil
}

func NewSigner(secret []byte, issuer string, ttl time.Duration) (*Signer, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("jwt signing secret is required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("jwt token ttl must be > 0")
	}
	return &Signer{secret: secret, issuer: strings.TrimSpace(issuer), ttl: ttl}, nil
}

func ParseAndValidateToken(tokenString string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrInvalidToken, token.Method.Alg())
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("%w: token validation failed", ErrInvalidToken)
	}
	if err := claims.Validate(); err != nil {
		return nil, err
	}
	return claims, nil
}

func (s *Signer) Sign(subject string, publ []string, subs []string, now time.Time) (string, *Claims, error) {
	if s == nil {
		return "", nil, fmt.Errorf("jwt signer is not configured")
	}
	now = now.UTC()
	claims := &Claims{
		Publ: append([]string(nil), publ...),
		Subs: append([]string(nil), subs...),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", nil, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, claims, nil
}
