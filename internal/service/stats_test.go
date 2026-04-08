package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMasteryLevel(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		level   string
	}{
		// Beginner: < 31
		{"zero", 0, "beginner"},
		{"low beginner", 10, "beginner"},
		{"upper beginner", 30, "beginner"},
		{"beginner boundary", 30.99, "beginner"},

		// Intermediate: 31-60
		{"intermediate start", 31, "intermediate"},
		{"mid intermediate", 45, "intermediate"},
		{"upper intermediate", 60, "intermediate"},
		{"intermediate boundary", 60.99, "intermediate"},

		// Advanced: 61-85
		{"advanced start", 61, "advanced"},
		{"mid advanced", 75, "advanced"},
		{"upper advanced", 85, "advanced"},
		{"advanced boundary", 85.99, "advanced"},

		// Mastered: >= 86
		{"mastered start", 86, "mastered"},
		{"high mastered", 95, "mastered"},
		{"perfect", 100, "mastered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.level, masteryLevel(tt.percent))
		})
	}
}
