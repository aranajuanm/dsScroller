package node

import (
	"math"
	"time"
)

type TokenRateState struct {
	refillPeriodNanos   uint64
	refillAmount        uint64
	capacity            uint64
	bucket              uint64
	lastRefillTimeNanos int64
	spill               float64
}

func NewState(refillPeriodNanos, refillAmount, capacity uint64) *TokenRateState {
	state := TokenRateState{}

	state.refillPeriodNanos = refillPeriodNanos
	state.refillAmount = refillAmount
	state.lastRefillTimeNanos = time.Now().UnixNano()

	if capacity < refillAmount {
		state.capacity = refillAmount
	} else {
		state.capacity = capacity
	}

	state.bucket = capacity
	state.spill = 0

	return &state
}

func newState(refillPeriodNanos, refillAmount, capacity, bucket uint64, lastRefillTimeNanos int64, spill float64) *TokenRateState {
	state := TokenRateState{}

	state.refillPeriodNanos = refillPeriodNanos
	state.refillAmount = refillAmount
	state.capacity = capacity
	state.bucket = bucket
	state.lastRefillTimeNanos = lastRefillTimeNanos
	state.spill = spill

	return &state
}

func (s *TokenRateState) copy() *TokenRateState {
	return newState(s.refillPeriodNanos, s.refillAmount, s.capacity, s.bucket, s.lastRefillTimeNanos, s.spill)
}

func (s *TokenRateState) refill(currentTimeNanos int64) {
	if currentTimeNanos < s.lastRefillTimeNanos {
		return
	}

	durationSinceLastRefillNanos := currentTimeNanos - s.lastRefillTimeNanos

	if uint64(durationSinceLastRefillNanos) > s.refillPeriodNanos {
		rawElapsedPeriods := float64(durationSinceLastRefillNanos) / float64(s.refillPeriodNanos)

		rawRefillAmount := float64(s.refillAmount) * rawElapsedPeriods
		refillAmount, spill := math.Modf(rawRefillAmount)
		s.bucket += uint64(refillAmount)
		s.spill += spill

		if s.spill >= 1 {
			correction := uint64(s.spill)
			s.spill -= float64(correction)
			s.bucket += correction
		}

		if s.bucket > s.capacity {
			s.bucket = s.capacity
		}

		s.lastRefillTimeNanos = currentTimeNanos
	}
}

func (s *TokenRateState) subtract(weight uint64) bool {
	if weight > s.bucket {
		return false
	}

	s.bucket -= weight
	return true
}

func (s *TokenRateState) copyStateFrom(n *TokenRateState) {
	s.refillPeriodNanos = n.refillPeriodNanos
}
