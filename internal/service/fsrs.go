package service

import (
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/rekanesiads/backend-quiz/config"
	"github.com/rekanesiads/backend-quiz/internal/domain"
)

func buildFSRS(user *domain.User, cfg config.FSRSConfig) *fsrs.FSRS {
	params := fsrs.DefaultParam()
	params.RequestRetention = user.DesiredRetention
	params.MaximumInterval = cfg.MaxInterval
	params.EnableFuzz = cfg.EnableFuzz

	if len(user.FSRSWeights) == len(params.W) {
		copy(params.W[:], user.FSRSWeights)
	}

	return fsrs.NewFSRS(params)
}

func toFSRSCard(c *domain.Card) fsrs.Card {
	fc := fsrs.Card{
		Due:           c.Due,
		Stability:     c.Stability,
		Difficulty:    c.Difficulty,
		ElapsedDays:   uint64(c.ElapsedDays),
		ScheduledDays: uint64(c.ScheduledDays),
		Reps:          uint64(c.Reps),
		Lapses:        uint64(c.Lapses),
		State:         fsrs.State(c.State),
	}
	if c.LastReview != nil {
		fc.LastReview = *c.LastReview
	}
	return fc
}

func fromFSRSCard(fc *fsrs.Card, c *domain.Card) {
	c.Due = fc.Due
	c.Stability = fc.Stability
	c.Difficulty = fc.Difficulty
	c.ElapsedDays = int(fc.ElapsedDays)
	c.ScheduledDays = int(fc.ScheduledDays)
	c.Reps = int(fc.Reps)
	c.Lapses = int(fc.Lapses)
	c.State = domain.CardState(fc.State)
	if !fc.LastReview.IsZero() {
		c.LastReview = &fc.LastReview
	}
}

func toPreviewInfo(info fsrs.SchedulingInfo) PreviewInfo {
	return PreviewInfo{
		Due:           info.Card.Due,
		Stability:     info.Card.Stability,
		Difficulty:    info.Card.Difficulty,
		ScheduledDays: int(info.Card.ScheduledDays),
		State:         domain.CardState(info.Card.State).String(),
	}
}
