package retry

import (
	"net/http"
	"time"
)

var defaultRetriableMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}

type RetryStrategy interface {
	ShouldRetry(req *http.Request, resp *http.Response, err error, retries int) RetryResponse
}

type RetryResponse interface {
	Retry() bool
	Delay() time.Duration
}

type retryResponse struct {
	retry bool
	delay time.Duration
}

func (r *retryResponse) Retry() bool {
	return r.retry
}

func (r *retryResponse) Delay() time.Duration {
	return r.delay
}

func isMethodAllowed(method string, allowedMethods []string) bool {
	for i := 0; i < len(allowedMethods); i++ {
		if allowedMethods[i] == method {
			return true
		}
	}
	return false
}
