package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateShareCode(t *testing.T) {
	code1 := generateShareCode()
	code2 := generateShareCode()

	assert.Len(t, code1, 12) // 6 bytes = 12 hex chars
	assert.Len(t, code2, 12)
	assert.NotEqual(t, code1, code2, "share codes should be unique")
}

func TestGenerateShareCode_HexEncoded(t *testing.T) {
	code := generateShareCode()

	// Should only contain hex characters
	for _, c := range code {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"share code should be hex encoded, got char: %c", c)
	}
}
