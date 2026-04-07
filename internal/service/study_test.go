package service

import (
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

func TestToFSRSCard_NewCard(t *testing.T) {
	now := time.Now()
	card := &domain.Card{
		Due:           now,
		Stability:     0,
		Difficulty:    0,
		ElapsedDays:   0,
		ScheduledDays: 0,
		Reps:          0,
		Lapses:        0,
		State:         domain.CardStateNew,
		LastReview:    nil,
	}

	fc := toFSRSCard(card)

	assert.Equal(t, fsrs.New, fc.State)
	assert.Equal(t, float64(0), fc.Stability)
	assert.Equal(t, float64(0), fc.Difficulty)
	assert.Equal(t, uint64(0), fc.Reps)
	assert.Equal(t, uint64(0), fc.Lapses)
	assert.True(t, fc.LastReview.IsZero())
}

func TestToFSRSCard_ReviewCard(t *testing.T) {
	now := time.Now()
	lastReview := now.Add(-24 * time.Hour)
	card := &domain.Card{
		Due:           now,
		Stability:     15.5,
		Difficulty:    5.3,
		ElapsedDays:   1,
		ScheduledDays: 1,
		Reps:          3,
		Lapses:        1,
		State:         domain.CardStateReview,
		LastReview:    &lastReview,
	}

	fc := toFSRSCard(card)

	assert.Equal(t, fsrs.Review, fc.State)
	assert.Equal(t, 15.5, fc.Stability)
	assert.Equal(t, 5.3, fc.Difficulty)
	assert.Equal(t, uint64(3), fc.Reps)
	assert.Equal(t, uint64(1), fc.Lapses)
	assert.False(t, fc.LastReview.IsZero())
}

func TestFromFSRSCard(t *testing.T) {
	now := time.Now()
	fc := &fsrs.Card{
		Due:           now.Add(24 * time.Hour),
		Stability:     20.0,
		Difficulty:    4.5,
		ElapsedDays:   1,
		ScheduledDays: 1,
		Reps:          1,
		Lapses:        0,
		State:         fsrs.Review,
		LastReview:    now,
	}

	card := &domain.Card{}
	fromFSRSCard(fc, card)

	assert.Equal(t, domain.CardStateReview, card.State)
	assert.Equal(t, 20.0, card.Stability)
	assert.Equal(t, 4.5, card.Difficulty)
	assert.Equal(t, 1, card.Reps)
	assert.Equal(t, 0, card.Lapses)
	assert.NotNil(t, card.LastReview)
}

func TestFSRSIntegration_NewCardReview(t *testing.T) {
	// Test the full FSRS flow: new card → rate Good → should move to Learning/Review
	params := fsrs.DefaultParam()
	params.RequestRetention = 0.9
	params.EnableFuzz = false
	f := fsrs.NewFSRS(params)

	// New card
	card := fsrs.NewCard()
	now := time.Now()

	// Rate "Good" on a new card
	result := f.Next(card, now, fsrs.Good)

	assert.Greater(t, result.Card.Stability, float64(0), "stability should increase after review")
	assert.Greater(t, result.Card.Difficulty, float64(0), "difficulty should be set after review")
	assert.Equal(t, uint64(1), result.Card.Reps, "reps should be 1 after first review")
	assert.Equal(t, uint64(0), result.Card.Lapses, "lapses should be 0 for good rating")
	assert.True(t, result.Card.Due.After(now), "next due should be in the future")
}

func TestFSRSIntegration_ReviewFlow(t *testing.T) {
	params := fsrs.DefaultParam()
	params.RequestRetention = 0.9
	params.EnableFuzz = false
	f := fsrs.NewFSRS(params)

	card := fsrs.NewCard()
	now := time.Now()

	// First review: Good
	r1 := f.Next(card, now, fsrs.Good)
	require.Greater(t, r1.Card.Stability, float64(0))

	// Second review: Good (later)
	now2 := r1.Card.Due
	r2 := f.Next(r1.Card, now2, fsrs.Good)

	assert.Greater(t, r2.Card.Stability, r1.Card.Stability, "stability should grow with successful reviews")
	assert.Equal(t, uint64(2), r2.Card.Reps)
	assert.True(t, r2.Card.Due.After(now2))
}

func TestFSRSIntegration_LapseFlow(t *testing.T) {
	params := fsrs.DefaultParam()
	params.RequestRetention = 0.9
	params.EnableFuzz = false
	f := fsrs.NewFSRS(params)

	card := fsrs.NewCard()
	now := time.Now()

	// Move card through Learning into Review state with repeated Good ratings
	r := f.Next(card, now, fsrs.Good)
	for r.Card.State != fsrs.Review {
		r = f.Next(r.Card, r.Card.Due, fsrs.Good)
	}
	require.Equal(t, fsrs.Review, r.Card.State, "card should be in Review state")

	stableBefore := r.Card.Stability

	// Now lapse: Again on a Review card
	r2 := f.Next(r.Card, r.Card.Due, fsrs.Again)

	assert.Less(t, r2.Card.Stability, stableBefore, "stability should decrease on lapse")
	assert.Equal(t, uint64(1), r2.Card.Lapses, "lapses should increment on Again")
}

func TestFSRSIntegration_AllRatings(t *testing.T) {
	params := fsrs.DefaultParam()
	params.RequestRetention = 0.9
	params.EnableFuzz = false
	f := fsrs.NewFSRS(params)

	card := fsrs.NewCard()
	now := time.Now()

	// Preview all ratings
	recordLog := f.Repeat(card, now)

	// All 4 ratings should produce results
	assert.Contains(t, recordLog, fsrs.Again)
	assert.Contains(t, recordLog, fsrs.Hard)
	assert.Contains(t, recordLog, fsrs.Good)
	assert.Contains(t, recordLog, fsrs.Easy)

	// Easy should give the longest interval
	easyDue := recordLog[fsrs.Easy].Card.Due
	goodDue := recordLog[fsrs.Good].Card.Due
	hardDue := recordLog[fsrs.Hard].Card.Due

	assert.True(t, easyDue.After(goodDue) || easyDue.Equal(goodDue), "easy due should be >= good due")
	assert.True(t, goodDue.After(hardDue) || goodDue.Equal(hardDue), "good due should be >= hard due")
}

func TestFSRSIntegration_Retrievability(t *testing.T) {
	params := fsrs.DefaultParam()
	params.RequestRetention = 0.9
	f := fsrs.NewFSRS(params)

	card := fsrs.NewCard()
	now := time.Now()

	// New card has 0 retrievability
	r := f.GetRetrievability(card, now)
	assert.Equal(t, float64(0), r, "new card should have 0 retrievability")

	// After review, retrievability should be > 0
	result := f.Next(card, now, fsrs.Good)
	rAfter := f.GetRetrievability(result.Card, now)
	assert.Greater(t, rAfter, float64(0), "reviewed card should have retrievability > 0")
}

func TestFSRSIntegration_CustomRetention(t *testing.T) {
	// Higher retention = shorter intervals
	highRetention := fsrs.DefaultParam()
	highRetention.RequestRetention = 0.95
	highRetention.EnableFuzz = false
	fHigh := fsrs.NewFSRS(highRetention)

	lowRetention := fsrs.DefaultParam()
	lowRetention.RequestRetention = 0.8
	lowRetention.EnableFuzz = false
	fLow := fsrs.NewFSRS(lowRetention)

	card := fsrs.NewCard()
	now := time.Now()

	rHigh := fHigh.Next(card, now, fsrs.Good)
	rLow := fLow.Next(card, now, fsrs.Good)

	// Lower retention target → longer intervals (fewer reviews needed)
	assert.True(t, rLow.Card.Due.After(rHigh.Card.Due) || rLow.Card.Due.Equal(rHigh.Card.Due),
		"lower retention should give longer or equal intervals")
}

func TestToPreviewInfo(t *testing.T) {
	now := time.Now()
	info := fsrs.SchedulingInfo{
		Card: fsrs.Card{
			Due:           now.Add(24 * time.Hour),
			Stability:     10.0,
			Difficulty:    5.0,
			ScheduledDays: 1,
			State:         fsrs.Review,
		},
	}

	preview := toPreviewInfo(info)

	assert.Equal(t, 10.0, preview.Stability)
	assert.Equal(t, 5.0, preview.Difficulty)
	assert.Equal(t, 1, preview.ScheduledDays)
	assert.Equal(t, "review", preview.State)
}

func TestCountCards(t *testing.T) {
	svc := &StudyService{}

	cards := []domain.Card{
		{State: domain.CardStateNew},
		{State: domain.CardStateNew},
		{State: domain.CardStateLearning},
		{State: domain.CardStateReview},
		{State: domain.CardStateReview},
		{State: domain.CardStateReview},
		{State: domain.CardStateRelearning},
	}

	counts := svc.countCards(cards)

	assert.Equal(t, 2, counts.New)
	assert.Equal(t, 2, counts.Learning) // Learning + Relearning
	assert.Equal(t, 3, counts.Review)
	assert.Equal(t, 7, counts.Total)
}
