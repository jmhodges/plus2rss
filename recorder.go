package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"sync/atomic"
)

// RecordingTransport records HTTP responses as JSON-marshalled
// ResponseFixtures on disk. Implements http.RoundTripper.
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
