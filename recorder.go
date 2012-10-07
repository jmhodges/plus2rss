package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"sync/atomic"
)

// RecordingTransport records HTTP responses as JSON-marshalled
// ResponseFixtures on disk. Be careful with this code as it will store
// Set-Cookie headers and its ilk if not used carefully. Implements
// http.RoundTripper.
type RecordingTransport struct {
	Transport http.RoundTripper
	DirPath   string
	Num       int32
}

func (t *RecordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Capture the URL before any other RoundTrippers can modify the URL
	// further.
	u, err := url.Parse(r.URL.String())
	if err != nil {
		return nil, err
	}
	re, err := t.Transport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	b := new(bytes.Buffer)
	b.ReadFrom(re.Body)
	rr := &httptest.ResponseRecorder{
		Code:      re.StatusCode,
		HeaderMap: re.Header,
		Body:      bytes.NewBuffer(b.Bytes()),
	}

	rf := &ResponseFixture{URL: u, Response: rr}
	re.Body = ioutil.NopCloser(b)
	j, err := rf.MarshalJSON()
	if err != nil {
		return nil, err
	}

	num := atomic.AddInt32(&t.Num, 1)
	path := filepath.Join(t.DirPath, "response"+strconv.Itoa(int(num-1))+".json")
	err = ioutil.WriteFile(path, j, 0644)
	if err != nil {
		return nil, err
	}
	return re, nil
}

type jsonFixture struct {
	URL       string              `json:"url"`
	Code      int                 `json:"code"`
	Body      string              `json:"body"`
	HeaderMap map[string][]string `json:"headers"`
}

// ResponseFixture contains the URL, headers, and body of a given HTTP
// response that was stored as JSON.
type ResponseFixture struct {
	URL      *url.URL
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
		Code:      obj.Code,
		Body:      bytes.NewBuffer([]byte(obj.Body)),
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
		URL:       r.URL.String(),
		Code:      r.Response.Code,
		Body:      string(r.Response.Body.Bytes()),
		HeaderMap: make(map[string][]string),
	}
	for k, v := range r.Response.HeaderMap {
		obj.HeaderMap[k] = v
	}
	return json.Marshal(obj)
}
