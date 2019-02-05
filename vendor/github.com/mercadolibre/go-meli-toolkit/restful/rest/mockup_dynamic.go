package rest

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const MOCK_DUPLICATE_ERROR_PREFIX string = "Dynamic DynamicMock already exists for url"
const DYN_MOCK_SERVER_NOT_INITIALIZED string = "Mock server not initialized. Call the rest.StartDynamicMockupServer() method first"

var dynamicMockContainer = synchronizedMockContainer{}
var dynamicMockServer *httptest.Server

// StaticMatcher - Default URL matching. Process all query params
var StaticMatcher = MatchingMode{
	Matcher: func(query *map[string]string) string {
		return createParamsKey(query)
	},
	AllowDuplicates: true,
}

// DynamicURLMatcher - Dynamic URL matching. Ignore query params, so all variations will match
var DynamicURLMatcher = MatchingMode{
	Matcher: func(query *map[string]string) string {
		return ""
	},
	AllowDuplicates: false,
}

func init() {
	flag.BoolVar(&mockUpEnv, "DynamicMock", false,
		"Use 'DynamicMock' flag to tell package rest that you would like to use dynamicMockups.")

	flag.Parse()
	startDynamicMockupServ()
}

type parametrizedDynamicMock struct {
	DynamicMock *DynamicMock
	dynamic     bool
	queries     map[string]bool
}

func (p *parametrizedDynamicMock) AddQuery(query *map[string]string) {
	key := createParamsKey(query)
	p.queries[key] = true
}

func (p *parametrizedDynamicMock) IsInQuery(matchingMode MatchingMode, query *map[string]string) bool {
	key := matchingMode.Matcher(query)
	_, ok := p.queries[key]
	return ok
}

func newParametrizedDynamicMock(m *DynamicMock) *parametrizedDynamicMock {
	value := &parametrizedDynamicMock{}
	value.DynamicMock = m
	value.queries = make(map[string]bool)
	return value
}

type synchronizedMockContainer struct {
	container map[string]*parametrizedDynamicMock
	lock      sync.RWMutex
}

func (m *synchronizedMockContainer) Init() {
	m.lock = *new(sync.RWMutex)
	m.Flush()
}

func (m *synchronizedMockContainer) Flush() {
	m.lock.Lock()
	m.container = make(map[string]*parametrizedDynamicMock)
	m.lock.Unlock()
}

func (m *synchronizedMockContainer) AddDynamicMock(DynamicMock *DynamicMock) error {
	var dynamicMockErr error

	if DynamicMock.URLMatcher.Matcher == nil {
		//Set static matcher as the default
		DynamicMock.URLMatcher = StaticMatcher
	}
	keyComponent, queryParams, dynamicMockErr := parseAndExtractParameters(DynamicMock.URL)

	m.lock.Lock()
	if m.container == nil {
		panic(DYN_MOCK_SERVER_NOT_INITIALIZED)
	}
	key := DynamicMock.HTTPMethod + " " + keyComponent
	if mc, ok := m.container[key]; !ok {
		m.container[key] = newParametrizedDynamicMock(DynamicMock)
		m.container[key].AddQuery(&queryParams)
	} else {
		if !mc.DynamicMock.URLMatcher.AllowDuplicates {
			dynamicMockErr = fmt.Errorf("%s %s", MOCK_DUPLICATE_ERROR_PREFIX, DynamicMock.URL)
		} else {
			m.container[key] = newParametrizedDynamicMock(DynamicMock)
			m.container[key].AddQuery(&queryParams)
		}
	}
	m.lock.Unlock()
	return dynamicMockErr
}

func (m *synchronizedMockContainer) GetDynamicMock(req *http.Request) (*DynamicMock, error) {
	var err error
	var DynamicMock *DynamicMock
	requestURL := req.Header.Get("X-Original-Url")

	m.lock.Lock()
	if m.container == nil {
		panic(DYN_MOCK_SERVER_NOT_INITIALIZED)
	}
	normalizedURL, queryParams, err := parseAndExtractParameters(requestURL)
	if err == nil {
		// Search match by composite key, then by indexed regexes
		key := req.Method + " " + normalizedURL
		if parametrizedDynamicMock, ok := m.container[key]; ok {
			if parametrizedDynamicMock.IsInQuery(parametrizedDynamicMock.DynamicMock.URLMatcher, &queryParams) {
				DynamicMock = parametrizedDynamicMock.DynamicMock
			} else {
				err = errors.New(MOCK_NOT_FOUND_ERROR)
			}
		} else {
			for _, v := range m.container {

				expr := v.DynamicMock.URL

				isMatch, _ := regexp.MatchString(expr, requestURL)
				if isMatch && v.DynamicMock.ParseURLRegex && req.Method == v.DynamicMock.HTTPMethod {
					DynamicMock = v.DynamicMock
					break
				}
			}
			if DynamicMock == nil {
				err = errors.New(MOCK_NOT_FOUND_ERROR)
			}
		}
	}
	m.lock.Unlock()

	if err != nil {
		return nil, err
	}
	return DynamicMock, nil
}

// Create key by param values
func createParamsKey(queryParams *map[string]string) string {
	key := ""
	queryKeys := make([]string, 0)
	for _, v := range *queryParams {
		queryKeys = append(queryKeys, v)
	}
	sort.Strings(queryKeys)
	for _, pv := range queryKeys {
		key = fmt.Sprintf("%s-%s", key, pv)
	}
	return key
}

// DynamicMock serves the purpose of creating dynamicMockups.
// All requests will be sent to the dynamicMockup server if dynamicMockup is activated.
// To activate the dynamicMockup *environment* you have two ways: using the flag -DynamicMock
//	go test -DynamicMock
//
// Or by programmatically starting the dynamicMockup server
// 	StartDynamicMockupServer()
type DynamicMock struct {

	// Request URL
	URL string

	// Request HTTP Method (GET, POST, PUT, PATCH, HEAD, DELETE, OPTIONS)
	// As a good practice use the constants in http package (http.MethodGet, etc.)
	HTTPMethod string

	// Request array Headers
	ReqHeaders http.Header

	// Request Body, used with POST, PUT & PATCH
	ReqBody string

	// Response HTTP Code
	RespHTTPCode int

	// Response Array Headers
	RespHeaders http.Header

	// Response Body
	RespBody string

	// Query params handler for URL matching. See StaticURLMatch and DynamicURLMatch
	URLMatcher MatchingMode

	// If true, will parse and treat the URL as a regex
	ParseURLRegex bool
}

// Matching mode for URLs
// Matcher: function that performs the URL query params (*map[string]string) filtering
// AllowDuplicates: specifies if the DynamicMock supports adding duplicates (that is, overwriting the previous DynamicMock reponse/request data)
type MatchingMode struct {
	Matcher         func(*map[string]string) string
	AllowDuplicates bool
}

// StartDynamicMockupServer sets the enviroment to send all client requests
// to the dynamicMockup server.
func StartDynamicMockupServer() {
	mockUpEnv = true

	if dynamicMockServer == nil {
		startDynamicMockupServ()
	}
}

// StopDynamicMockupServer stop sending requests to the dynamicMockup server
func StopDynamicMockupServer() {
	if dynamicMockServer == nil {
		panic(DYN_MOCK_SERVER_NOT_INITIALIZED)
	}

	mockUpEnv = false
	dynamicMockServer.Close()

	dynamicMockServer = nil
	mockServerURL = nil
	mux = nil
}

func startDynamicMockupServ() {
	if mockUpEnv {
		mux = http.NewServeMux()
		dynamicMockServer = httptest.NewServer(mux)
		mux.HandleFunc("/", dynamicMockupHandler)
		dynamicMockContainer.Init()

		var err error
		if mockServerURL, err = url.Parse(dynamicMockServer.URL); err != nil {
			panic(err)
		}
	}
}

// AddDynamicMockups ...
func AddDynamicMockups(mocks ...*DynamicMock) error {
	var err error
	for _, m := range mocks {
		err = dynamicMockContainer.AddDynamicMock(m)
	}
	return err
}

// Parse and extract the key and queryparams for a given URL, accepts regexes
func parseAndExtractParameters(urlStr string) (string, map[string]string, error) {
	var urlErr error
	keyComponent, queryParams, urlErr := parseAndExtractURL(urlStr)

	if urlErr != nil {
		_, err := regexp.Compile(urlStr)
		if err != nil {
			urlErr = fmt.Errorf("Error parsing DynamicMock with url=%s. Cause: %s", urlStr, err.Error())
		} else {
			keyComponent = urlStr
		}
	}

	return keyComponent, queryParams, urlErr
}

// Parse and extract the key and queryparams for a given URL
func parseAndExtractURL(urlStr string) (string, map[string]string, error) {
	params := make(map[string]string)

	urlObj, err := url.Parse(urlStr)
	if err != nil {
		return "", params, err
	}
	result := urlStr

	// Sort query param strings
	if len(urlObj.RawQuery) > 0 {
		result = strings.Replace(urlStr, urlObj.RawQuery, "", 1)
		mk := make([]string, len(urlObj.Query()))
		i := 0
		for k := range urlObj.Query() {
			mk[i] = k
			i++
		}
		sort.Strings(mk)
		for i := 0; i < len(mk); i++ {
			params[mk[i]] = urlObj.Query().Get(mk[i])
		}
	}

	//Strip trailing slashes question marks
	if (result[len(result)-1] == '?') || (result[len(result)-1] == '/') {
		result = result[0 : len(result)-1]
	}

	return result, params, nil
}

// FlushDynamicMockups ...
func FlushDynamicMockups() {
	dynamicMockContainer.Flush()
}

func dynamicMockupHandler(writer http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	body, _ := ioutil.ReadAll(req.Body)

	m, err := dynamicMockContainer.GetDynamicMock(req)
	if err == nil && m != nil {
		matchesBody := true
		if m.ReqBody != "" {
			matchesBody = m.ReqBody == string(body)
		}
		matchesHeaders := true
		if m.ReqHeaders != nil {
			for h := range m.ReqHeaders {
				expectedHeader := m.ReqHeaders.Get(h)
				foundHeader := req.Header.Get(h)
				if expectedHeader != foundHeader {
					matchesHeaders = false
					break
				}
			}
		}

		if !matchesBody {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte(MOCK_NOT_MATCH_BODY))
			return
		}

		if !matchesHeaders {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte(MOCK_NOT_MATCH_HEADERS))
			return
		}

		// Add headers
		for k, v := range m.RespHeaders {
			for _, vv := range v {
				writer.Header().Add(k, vv)
			}
		}

		writer.WriteHeader(m.RespHTTPCode)
		writer.Write([]byte(m.RespBody))
		return
	}

	// Fallback path, no DynamicMock match found
	writer.WriteHeader(http.StatusBadRequest)
	writer.Write([]byte(MOCK_NOT_FOUND_ERROR))
}
