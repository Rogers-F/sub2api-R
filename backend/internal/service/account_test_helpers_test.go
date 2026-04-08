//go:build unit

package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

// queuedHTTPUpstream is a test helper that serves pre-loaded responses in order.
type queuedHTTPUpstream struct {
	responses []*http.Response
	requests  []*http.Request
	tlsFlags  []bool
}

func (u *queuedHTTPUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (u *queuedHTTPUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, enableTLSFingerprint bool) (*http.Response, error) {
	u.requests = append(u.requests, req)
	u.tlsFlags = append(u.tlsFlags, enableTLSFingerprint)
	if len(u.responses) == 0 {
		return nil, fmt.Errorf("no mocked response")
	}
	resp := u.responses[0]
	u.responses = u.responses[1:]
	return resp, nil
}

// newJSONResponse creates a simple HTTP response for testing.
func newJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// newJSONResponseWithHeader creates a JSON response with a custom header.
func newJSONResponseWithHeader(status int, body, key, value string) *http.Response {
	resp := newJSONResponse(status, body)
	resp.Header.Set(key, value)
	return resp
}

func newAccountTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)
	return c, rec
}
