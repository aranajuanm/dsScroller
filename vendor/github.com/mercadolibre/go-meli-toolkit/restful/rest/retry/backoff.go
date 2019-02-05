package retry

import (
	"math"
	"math/rand"
	"net/http"
	"time"
)

const DEFAULT_FACTOR = 0.2
const DEFAULT_MULTIPLIER = 2.0

type backoffRetryStrategy struct {
	min            time.Duration
	max            time.Duration
	factor         float64
	multiplier     float64
	allowedMethods []string
}

func NewExponentialBackoffRetryStrategyWithCustomFactorAndMultiplier(min time.Duration, max time.Duration, factor float64, multiplier float64, verbs ...string) RetryStrategy {
	if min <= 0 || max <= 0 || max <= min {
		return nil
	}
	allowedMethods := defaultRetriableMethods
	if len(verbs) > 0 {
		allowedMethods = verbs
	}
	return backoffRetryStrategy{min, max, factor, multiplier, allowedMethods}
}

func NewExponentialBackoffRetryStrategy(min time.Duration, max time.Duration, verbs ...string) RetryStrategy {
	return NewExponentialBackoffRetryStrategyWithCustomFactorAndMultiplier(min, max, DEFAULT_FACTOR, DEFAULT_MULTIPLIER, verbs...)
}

func (r backoffRetryStrategy) ShouldRetry(req *http.Request, resp *http.Response, err error, retries int) RetryResponse {
	var delay time.Duration
	retry := (err != nil || resp.StatusCode >= http.StatusInternalServerError) && isMethodAllowed(req.Method, r.allowedMethods)
	if retry {
		delay = getDelay(r.min, r.factor, r.multiplier, retries)
		retry = delay <= r.max
	}
	return &retryResponse{retry, delay}
}

func getDelay(min time.Duration, factor float64, multiplier float64, retries int) time.Duration {
	interval := min.Nanoseconds() * int64(math.Pow(multiplier, float64(retries)))
	return getFromInterval((1-factor)*float64(interval), (1+factor)*float64(interval))
}

func getFromInterval(a float64, b float64) time.Duration {
	return time.Duration(rand.Float64()*(b-a) + a)
}
