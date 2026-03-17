package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchTopicFilter(t *testing.T) {
	assert.True(t, MatchTopicFilter("a/b/+/d/#", "a/b/c/d/e"))
	assert.True(t, MatchTopicFilter("a/b/c", "a/b/c"))
	assert.False(t, MatchTopicFilter("a/b/c", "a/b/c/d"))
	assert.False(t, MatchTopicFilter("a/+/c", "a/b/d"))
}

func TestFilterCanMatchPrefix(t *testing.T) {
	assert.True(t, FilterCanMatchPrefix("acct/device/+/+/+", []string{"acct", "device"}, 5))
	assert.True(t, FilterCanMatchPrefix("acct/device/instance/node/+", []string{"acct", "device", "instance"}, 5))
	assert.False(t, FilterCanMatchPrefix("other/device/+/+/+", []string{"acct", "device"}, 5))
}
