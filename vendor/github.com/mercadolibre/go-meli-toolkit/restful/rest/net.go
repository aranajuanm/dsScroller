package rest

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/mercadolibre/go-meli-toolkit/godog"
)

var readVerbs = [3]string{http.MethodGet, http.MethodHead, http.MethodOptions}
var contentVerbs = [3]string{http.MethodPost, http.MethodPut, http.MethodPatch}
var defaultCheckRedirectFunc func(req *http.Request, via []*http.Request) error

var maxAge = regexp.MustCompile(`(?:max-age|s-maxage)=(\d+)`)

const httpDateFormat = "Mon, 01 Jan 2006 15:04:05 GMT"
const RETRY_HEADER = "X-Retry"

func (rb *RequestBuilder) DoRequest(verb string, reqURL string, reqBody interface{}) (result *Response) {
	var cacheURL string
	var cacheResp *Response

	result = new(Response)
	reqURL = rb.BaseURL + reqURL

	//If Cache enable && operation is read: Cache GET
	if rb.EnableCache && matchVerbs(verb, readVerbs) {
		if cacheResp = resourceCache.get(reqURL); cacheResp != nil {
			cacheResp.cacheHit.Store(true)
			if !cacheResp.revalidate {
				return cacheResp
			}
		}
	}

	if !rb.EnableCache {
		rb.headersMtx.Lock()
		delete(rb.Headers, "If-None-Match")
		delete(rb.Headers, "If-Modified-Since")
		rb.headersMtx.Unlock()
	}

	//Marshal request to JSON or XML
	body, err := rb.marshalReqBody(reqBody)
	if err != nil {
		result.Err = err
		return
	}

	// Change URL to point to Mockup server
	reqURL, cacheURL, err = checkMockup(reqURL)
	if err != nil {
		result.Err = err
		return
	}

	// Make the request and
	var httpResp *http.Response
	var responseErr error

	end := false
	retries := 0
	for !end {
		request, err := http.NewRequest(verb, reqURL, bytes.NewBuffer(body))
		if err != nil {
			result.Err = err
			return
		}

		if !rb.MetricsConfig.DisableHttpConnectionsMetrics {
			//Create request
			trace := &httptrace.ClientTrace{
				GetConn: func(hostPort string) {
					godog.RecordSimpleMetric("conn_request", 1, new(godog.Tags).Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)

				},
				GotConn: func(connInfo httptrace.GotConnInfo) {
					if connInfo.Reused {
						godog.RecordSimpleMetric("conn_got", 1, new(godog.Tags).Add("status", "reused").Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)
					} else {
						godog.RecordSimpleMetric("conn_got", 1, new(godog.Tags).Add("status", "not_reused").Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)
					}

				},
				ConnectDone: func(network, addr string, err error) {
					if err != nil {
						godog.RecordSimpleMetric("conn_new", 1, new(godog.Tags).Add("status", "fail").Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)
					} else {
						godog.RecordSimpleMetric("conn_new", 1, new(godog.Tags).Add("status", "ok").Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)
					}
				},
				PutIdleConn: func(err error) {
					if err != nil {
						godog.RecordSimpleMetric("conn_put_idle", 1, new(godog.Tags).Add("status", "fail").Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)
					} else {
						godog.RecordSimpleMetric("conn_put_idle", 1, new(godog.Tags).Add("status", "ok").Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)
					}
				},
			}
			request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))
		}

		// Set extra parameters
		rb.setParams(request, cacheResp, cacheURL)

		initTime := time.Now()
		httpResp, responseErr = rb.getClient().Do(request)
		if !rb.MetricsConfig.DisableApiCallMetrics {
			if responseErr != nil {
				godog.RecordApiCallMetric(rb.MetricsConfig.TargetId, initTime, "error", retries > 0)
			} else {
				godog.RecordApiCallMetric(rb.MetricsConfig.TargetId, initTime, strconv.Itoa(httpResp.StatusCode), retries > 0)
			}
		}
		if rb.RetryStrategy != nil {
			retryResp := rb.RetryStrategy.ShouldRetry(request, httpResp, responseErr, retries)
			if retryResp.Retry() {
				retryFunc := func() (interface{}, error) {
					time.Sleep(retryResp.Delay())
					retries++
					request.Header.Set(RETRY_HEADER, strconv.Itoa(retries))
					return nil, nil
				}

				if _, err := retryLimiter.Action(1, retryFunc); err == nil {
					continue
				} else if !rb.MetricsConfig.DisableApiCallMetrics {
					godog.RecordSimpleMetric("go.api_call.retry_break", 1, new(godog.Tags).Add("target_id", rb.MetricsConfig.TargetId).ToArray()...)
				}
			}
		}
		end = true
	}
	if responseErr != nil {
		result.Err = responseErr
		return
	}

	// Read response
	defer httpResp.Body.Close()
	respBody, err := ioutil.ReadAll(httpResp.Body)

	if err != nil {
		result.Err = err
		return
	}

	// If we get a 304, return response from cache
	if httpResp.StatusCode == http.StatusNotModified {
		result = cacheResp
		return
	}

	result.Response = httpResp
	if !rb.UncompressResponse {
		result.byteBody = respBody
	} else {
		respEncoding := httpResp.Header.Get("Content-Encoding")
		if respEncoding == "" {
			respEncoding = httpResp.Header.Get("Content-Type")
		}
		switch respEncoding {
		case "gzip":
			fallthrough
		case "application/x-gzip":
			{
				if len(respBody) == 0 {
					break
				}
				gr, err := gzip.NewReader(bytes.NewBuffer(respBody))
				defer gr.Close()
				if err != nil {
					result.Err = err
				} else {
					uncompressedData, err := ioutil.ReadAll(gr)
					if err != nil {
						result.Err = err
					} else {
						result.byteBody = uncompressedData
					}
				}
			}
		default:
			{
				result.byteBody = respBody
			}
		}
	}

	ttl := setTTL(result)
	lastModified := setLastModified(result)
	etag := setETag(result)

	if !ttl && (lastModified || etag) {
		result.revalidate = true
	}

	//If Cache enable: Cache SETNX
	if rb.EnableCache && matchVerbs(verb, readVerbs) && (ttl || lastModified || etag) {
		resourceCache.setNX(cacheURL, result)
	}

	return
}

func checkMockup(reqURL string) (string, string, error) {

	cacheURL := reqURL

	if mockUpEnv {

		rURL, err := url.Parse(reqURL)
		if err != nil {
			return reqURL, cacheURL, err
		}

		rURL.Scheme = mockServerURL.Scheme
		rURL.Host = mockServerURL.Host

		return rURL.String(), cacheURL, nil
	}

	return reqURL, cacheURL, nil
}

func (rb *RequestBuilder) marshalReqBody(body interface{}) (b []byte, err error) {

	if body != nil {
		switch rb.ContentType {
		case JSON:
			b, err = json.Marshal(body)
		case XML:
			b, err = xml.Marshal(body)
		case BYTES:
			var ok bool
			b, ok = body.([]byte)
			if !ok {
				err = fmt.Errorf("bytes: body is %T(%v) not a byte slice", body, body)
			}
		}
	}

	return
}

func (rb *RequestBuilder) getClient() *http.Client {

	// This will be executed only once
	// per request builder
	rb.clientMtxOnce.Do(func() {

		tr := defaultTransport

		if cp := rb.CustomPool; cp != nil {
			if cp.Transport == nil {
				tr = &http.Transport{
					MaxIdleConnsPerHost:   rb.CustomPool.MaxIdleConnsPerHost,
					DialContext:           (&net.Dialer{Timeout: rb.getConnectionTimeout()}).DialContext,
					ResponseHeaderTimeout: rb.getRequestTimeout(),
				}

				//Set Proxy
				if cp.Proxy != "" {
					if proxy, err := url.Parse(cp.Proxy); err == nil {
						tr.(*http.Transport).Proxy = http.ProxyURL(proxy)
					}
				}
				cp.Transport = tr
			} else {
				if ctr, ok := cp.Transport.(*http.Transport); ok {
					ctr.DialContext = (&net.Dialer{Timeout: rb.getConnectionTimeout()}).DialContext
					ctr.ResponseHeaderTimeout = rb.getRequestTimeout()
					tr = ctr
				} else {
					// If custom transport is not http.Transport, timeouts will not be overwritten.
					tr = cp.Transport
				}
			}
		}

		rb.Client = &http.Client{Transport: tr, Timeout: rb.getConnectionTimeout() + rb.getRequestTimeout()}

		if !rb.FollowRedirect {
			rb.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return errors.New("Avoided redirect attempt")
			}
		} else {
			rb.Client.CheckRedirect = defaultCheckRedirectFunc
		}
	})

	return rb.Client
}

func (rb *RequestBuilder) getRequestTimeout() time.Duration {

	switch {
	case rb.DisableTimeout:
		return 0
	case rb.Timeout > 0:
		return rb.Timeout
	default:
		return DefaultTimeout
	}
}

func (rb *RequestBuilder) getConnectionTimeout() time.Duration {

	switch {
	case rb.DisableTimeout:
		return 0
	case rb.ConnectTimeout > 0:
		return rb.ConnectTimeout
	default:
		return DefaultConnectTimeout
	}
}

func (rb *RequestBuilder) setParams(req *http.Request, cacheResp *Response, cacheURL string) {

	//Custom Headers
	if rb.Headers != nil && len(rb.Headers) > 0 {
		rb.headersMtx.RLock()
		for key, values := range map[string][]string(rb.Headers) {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		rb.headersMtx.RUnlock()
	}

	//Default headers
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")

	//If mockup
	if mockUpEnv {
		req.Header.Set("X-Original-URL", cacheURL)
	}

	// Basic Auth
	if rb.BasicAuth != nil {
		req.SetBasicAuth(rb.BasicAuth.UserName, rb.BasicAuth.Password)
	}

	// User Agent
	req.Header.Set("User-Agent", func() string {
		if rb.UserAgent != "" {
			return rb.UserAgent
		}
		return "github.com/go-loco/restful"
	}())

	//Encoding
	var cType string

	switch rb.ContentType {
	case JSON:
		cType = "json"
	case XML:
		cType = "xml"
	}

	if cType != "" {
		req.Header.Set("Accept", "application/"+cType)

		if matchVerbs(req.Method, contentVerbs) {
			req.Header.Set("Content-Type", "application/"+cType)
		}
	}

	if cacheResp != nil && cacheResp.revalidate {
		switch {
		case cacheResp.etag != "":
			req.Header.Set("If-None-Match", cacheResp.etag)
		case cacheResp.lastModified != nil:
			req.Header.Set("If-Modified-Since", cacheResp.lastModified.Format(httpDateFormat))
		}
	}

}

func matchVerbs(s string, sarray [3]string) bool {
	for i := 0; i < len(sarray); i++ {
		if sarray[i] == s {
			return true
		}
	}

	return false
}

func setTTL(resp *Response) (set bool) {

	now := time.Now()

	//Cache-Control Header
	cacheControl := maxAge.FindStringSubmatch(resp.Header.Get("Cache-Control"))

	if len(cacheControl) > 1 {

		ttl, err := strconv.Atoi(cacheControl[1])
		if err != nil {
			return
		}

		if ttl > 0 {
			t := now.Add(time.Duration(ttl) * time.Second)
			resp.ttl = &t
			set = true
		}

		return
	}

	//Expires Header
	//Date format from RFC-2616, Section 14.21
	expires, err := time.Parse(httpDateFormat, resp.Header.Get("Expires"))
	if err != nil {
		return
	}

	if expires.Sub(now) > 0 {
		resp.ttl = &expires
		set = true
	}

	return
}

func setLastModified(resp *Response) bool {
	lastModified, err := time.Parse(httpDateFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return false
	}

	resp.lastModified = &lastModified
	return true
}

func setETag(resp *Response) bool {

	resp.etag = resp.Header.Get("ETag")

	return resp.etag != ""
}
