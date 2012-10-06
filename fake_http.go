package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
)

var (
	UnknownHTTPMethod = errors.New("no such HTTP method known to FakeClientTransport")
	UnknownURL = errors.New("no such URL known to FakeClientTransport")
	_ http.RoundTripper = &FakeClientTransport{}
)

// FakeClientTransport implements http.RoundTripper to provide a means of
// writing tests of code using a remote HTTP service without a network
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
		Request: req,
		Body: ioutil.NopCloser(bytes.NewBuffer(rr.Body.Bytes())),
		StatusCode: rr.Code,
		Status: http.StatusText(rr.Code),
		Header: make(map[string][]string),
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

type jsonFixture struct {
	URL string `json:"url"`
	Code int `json:"code"`
	Body string `json:"body"`
	HeaderMap map[string][]string `json:"headers"`
}

// ResponseFixture contains the URL, headers, and body of a given HTTP
// response that was stored as JSON.
type ResponseFixture struct {
	URL *url.URL
	Response *httptest.ResponseRecorder
}

func (r *ResponseFixture) UnmarshalJSON(j []byte) error {
	obj := &jsonFixture{}
	err := json.Unmarshal(j, obj)
	if err != nil {
		return err
	}
	u, err := url.Parse(obj.URL)
	if err != nil {
		return err
	}
	re := &httptest.ResponseRecorder{
		Code: obj.Code,
		Body: bytes.NewBuffer([]byte(obj.Body)),
		HeaderMap: make(map[string][]string),
	}
	for k, v := range obj.HeaderMap {
		arr := make([]string, 0, len(v))
		for _, vv := range v {
			arr = append(arr, vv)
		}
		re.HeaderMap[k] = arr
	}
	r.URL = u
	r.Response = re
	return nil
}

func (r *ResponseFixture) MarshalJSON() ([]byte, error) {
	obj := &jsonFixture{
		URL: r.URL.String(),
		Code: r.Response.Code,
		Body: string(r.Response.Body.Bytes()),
		HeaderMap: make(map[string][]string),
	}
	return json.Marshal(obj)
}
