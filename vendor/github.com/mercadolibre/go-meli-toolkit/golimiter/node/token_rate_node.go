package node

import (
	"math"
	"sync/atomic"
	"time"
	"unsafe"
)

type TokenRateNode struct {
	maxRate           uint64
	normalizeRate     uint64
	refillPeriodNanos uint64
	bucketWidthMillis uint64
	normalizedRate    uint64
	state             *TokenRateState
}

const (
	minuteNanos        uint64 = 60 * 1000000000
	minuteMillis              = 60 * 1000
	minRefillRateNanos        = 1
)

func New(maxRate, bucketWidthMillis uint64) *TokenRateNode {
	node := TokenRateNode{}

	node.bucketWidthMillis = bucketWidthMillis
	node.setMaxRate(maxRate)

	return &node
}

func (n *TokenRateNode) setMaxRate(maxRate uint64) {
	n.maxRate = maxRate

	n.refillPeriodNanos = minuteNanos / maxRate

	if n.refillPeriodNanos < minRefillRateNanos {
		n.refillPeriodNanos = minRefillRateNanos
	}

	rawRate := float64(maxRate*n.refillPeriodNanos) / float64(minuteNanos)
	adjust := offset(rawRate)

	n.refillPeriodNanos *= uint64(adjust)
	n.normalizedRate = uint64(rawRate * float64(adjust))

	n.reset()
}

func offset(d float64) int {
	min := math.MaxFloat64
	output := 1

	for i := 1; i < 20; i++ {
		prod := d * float64(i)
		_, f := math.Modf(prod)
		diff := math.Min(f, 1-f)

		if diff < min {
			min = diff
			output = i
		}
	}

	return output
}

func (n *TokenRateNode) stateAddress() *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(&n.state))
}

func (n *TokenRateNode) reset() {
	capacity := uint64(math.Ceil(float64(n.bucketWidthMillis*n.maxRate) / float64(minuteMillis)))
	state := NewState(n.refillPeriodNanos, n.normalizedRate, capacity)
	atomic.StorePointer(n.stateAddress(), unsafe.Pointer(state))
}

func (n *TokenRateNode) Reject(weight uint64) bool {
	state := (*TokenRateState)(atomic.LoadPointer(n.stateAddress()))
	next := state.copy()
	currentTimeNanos := time.Now().UnixNano()

	for {
		next.refill(currentTimeNanos)

		if !next.subtract(weight) {
			return true
		}

		if atomic.CompareAndSwapPointer(n.stateAddress(), unsafe.Pointer(state), unsafe.Pointer(next)) {
			return false
		} else {
			state = (*TokenRateState)(atomic.LoadPointer(n.stateAddress()))
			next.copyStateFrom(state)
		}
	}
}
