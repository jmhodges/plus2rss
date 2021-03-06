package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"google.golang.org/api/googleapi"
	plus "google.golang.org/api/plus/v1"
)

var (
	personResp    = mustResponse(ioutil.ReadFile("./testdata/person.json"))
	feedResp      = mustResponse(ioutil.ReadFile("./testdata/feed.json"))
	person404Resp = mustResponse(ioutil.ReadFile("./testdata/person_404.json"))
	feed404Resp   = mustResponse(ioutil.ReadFile("./testdata/feed_404.json"))
)

func TestSuccessfulFind(t *testing.T) {
	tr := &FakeClientTransport{}
	tr.Add(personResp.URL, "GET", personResp.Response)
	tr.Add(feedResp.URL, "GET", feedResp.Response)
	srv, err := plus.New(&http.Client{Transport: tr})
	if err != nil {
		t.Fatalf("unable to make Google+ client: %s", err)
	}
	userId := "116810148281701144465"
	fr := &FeedRetriever{srv, nullLog()}
	feed, err := fr.Find(userId)
	if err != nil {
		t.Fatalf("unable to Find id: %s", err)
	}
	if userId != feed.ActorId() {
		t.Errorf("expected id: %#v, actual id: %#v", userId, feed.Id())
	}
	if "Russ Cox" != feed.ActorName() {
		t.Errorf("expected name: \"Russ Cox\", actual name: %#v", feed.ActorName())
	}
	feedID := "plus2rss-" + userId
	if feed.Id() != feedID {
		t.Errorf("feed.ID: want %q, got %q", feedID, feed.Id())
	}
}

var resp404Table = []struct {
	pers *ResponseFixture
	feed *ResponseFixture
}{
	{personResp, feed404Resp},
	{person404Resp, feedResp},
	{person404Resp, feed404Resp},
}

func Test404Find(t *testing.T) {
	for _, rs := range resp404Table {
		tr := &FakeClientTransport{}
		tr.Add(person404Resp.URL, "GET", rs.pers.Response)
		tr.Add(feed404Resp.URL, "GET", rs.feed.Response)

		srv, err := plus.New(&http.Client{Transport: tr})
		if err != nil {
			t.Errorf("unable to make Google+ client: %s", err)
			continue
		}
		fr := &FeedRetriever{srv, nullLog()}
		_, err = fr.Find("444")
		if err == nil {
			t.Errorf("no error returned on 404")
			continue
		}
		gerr, ok := err.(*googleapi.Error)
		if !ok {
			t.Errorf("expected response to be a google api error, was %#v", err)
			continue
		}
		if gerr.Code != 404 {
			t.Errorf("expected error to be have code 404, was %d", gerr.Code)
			continue
		}
	}
}

func mustResponse(j []byte, err error) *ResponseFixture {
	if err != nil {
		panic(err)
	}
	r := &ResponseFixture{}
	if err := r.UnmarshalJSON(j); err != nil {
		panic(err)
	}
	return r
}

func nullLog() *log.Logger {
	return log.New(ioutil.Discard, "", 0)
}
