package main

import (
	"net/http"
)

type SimpleKeyTransport struct {
	Key       string
	Transport http.RoundTripper
}

func (t *SimpleKeyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	q := r.URL.Query()
	q.Set("key", t.Key)
	r.URL.RawQuery = q.Encode()
	return t.Transport.RoundTrip(r)
}
