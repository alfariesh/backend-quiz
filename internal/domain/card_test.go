package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCardState_String(t *testing.T) {
	tests := []struct {
		state    CardState
		expected string
	}{
		{CardStateNew, "new"},
		{CardStateLearning, "learning"},
		{CardStateReview, "review"},
		{CardStateRelearning, "relearning"},
		{CardState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestRatingConstants(t *testing.T) {
	assert.Equal(t, Rating(1), RatingAgain)
	assert.Equal(t, Rating(2), RatingHard)
	assert.Equal(t, Rating(3), RatingGood)
	assert.Equal(t, Rating(4), RatingEasy)
}

func TestCardStateConstants(t *testing.T) {
	assert.Equal(t, CardState(0), CardStateNew)
	assert.Equal(t, CardState(1), CardStateLearning)
	assert.Equal(t, CardState(2), CardStateReview)
	assert.Equal(t, CardState(3), CardStateRelearning)
}
