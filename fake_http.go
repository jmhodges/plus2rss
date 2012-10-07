package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
)

var (
	UnknownHTTPMethod                   = errors.New("no such HTTP method known to FakeClientTransport")
	UnknownURL                          = errors.New("no such URL known to FakeClientTransport")
	_                 http.RoundTripper = &FakeClientTransport{}
)

// FakeClientTransport implements http.RoundTripper to provide a means of
// writing tests for code using a remote HTTP service without a network
// connection. It assumes all information needed to specify a correct response
// is in the URL and the HTTP method used. See also
// net/http/httptest.ResponseRecorder.
type FakeClientTransport struct {
	items map[string]map[string]*httptest.ResponseRecorder
}

func (m *FakeClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	methMap, ok := m.items[req.Method]
	if !ok {
		return nil, UnknownHTTPMethod
	}
	rr, ok := methMap[req.URL.String()]
	if !ok {
		return nil, UnknownURL
	}
	re := &http.Response{
		Request:    req,
		Body:       ioutil.NopCloser(bytes.NewBuffer(rr.Body.Bytes())),
		StatusCode: rr.Code,
		Status:     http.StatusText(rr.Code),
		Header:     make(map[string][]string),
	}
	for k, v := range rr.HeaderMap {
		re.Header[k] = v
	}
	return re, nil
}

// Add adds a ResponseRecorder that is will be returned when the given url is
// accessed with the given HTTP method.
func (t *FakeClientTransport) Add(u *url.URL, method string, re *httptest.ResponseRecorder) {
	if t.items == nil {
		t.items = make(map[string]map[string]*httptest.ResponseRecorder)
	}
	methMap, ok := t.items[method]
	if !ok {
		methMap = make(map[string]*httptest.ResponseRecorder)
		t.items[method] = methMap
	}
	methMap[u.String()] = re
}
