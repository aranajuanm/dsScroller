package retry

import (
	"net/http"
	"time"
)

// Simple Retry Strategy
type simpleRetryStrategy struct {
	maxRetries     int
	delay          time.Duration
	allowedMethods []string
}

func NewSimpleRetryStrategy(maxRetries int, delay time.Duration, verbs ...string) RetryStrategy {
	if maxRetries < 0 || delay < 0 {
		return nil
	}
	allowedMethods := defaultRetriableMethods
	if len(verbs) > 0 {
		allowedMethods = verbs
	}
	return simpleRetryStrategy{maxRetries, delay, allowedMethods}
}

func (r simpleRetryStrategy) ShouldRetry(req *http.Request, resp *http.Response, err error, retries int) RetryResponse {
	retry := retries < r.maxRetries && (err != nil || resp.StatusCode >= http.StatusInternalServerError) && isMethodAllowed(req.Method, r.allowedMethods)
	return &retryResponse{retry, r.delay}
}
