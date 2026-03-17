package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignAndParseToken(t *testing.T) {
	secret, err := DecodeBase64Secret("ZGV2X3NlY3JldA==")
	require.NoError(t, err)
	signer, err := NewSigner(secret, "issuer", time.Hour)
	require.NoError(t, err)

	token, claims, err := signer.Sign("acct/device/instance", []string{"acct/device/instance/+/+"}, []string{"acct/device/instance/+/+/set"}, time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, "acct/device/instance", claims.Subject)

	parsed, err := ParseAndValidateToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, "acct/device/instance", parsed.Subject)
	assert.Equal(t, []string{"acct/device/instance/+/+"}, parsed.Publ)
	assert.Equal(t, []string{"acct/device/instance/+/+/set"}, parsed.Subs)
}
