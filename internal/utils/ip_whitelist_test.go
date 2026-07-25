package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndMatchIPWhitelist(t *testing.T) {
	entries, err := NormalizeIPWhitelist([]string{" 203.0.113.10 ", "203.0.113.0/24", "203.0.113.1/24", "::ffff:203.0.113.10"})
	require.NoError(t, err)
	assert.Equal(t, []string{"203.0.113.10", "203.0.113.0/24"}, entries)
	assert.True(t, IPAllowed("203.0.113.10", entries))
	assert.True(t, IPAllowed("203.0.113.99", entries))
	assert.False(t, IPAllowed("198.51.100.1", entries))
	_, err = NormalizeIPWhitelist([]string{"not-an-ip"})
	assert.Error(t, err)
}
