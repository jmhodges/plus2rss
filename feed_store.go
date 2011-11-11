package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type FeedRetriever struct {
	Client *http.Client
	Key    string
}

type FeedStorage interface {
	Find(string) (Feed, error)
}

type Feed interface {
	Title() string
	Id() string
	Updated() string
	Items() []Activity
}

type Activity interface {
	Verb() string
	Updated() string
	Published() string
	Content() string
	Title() string
	Id() string
	URL() string
	ActorName() string
}

var FeedStore FeedStorage

type InvalidStatusCode int

func (isc InvalidStatusCode) Error() string {
	return fmt.Sprintf("Error status code from Google+: %d", int(isc))
}

// TODO An obvious place to cache data.
func (f *FeedRetriever) Find(userId string) (Feed, error) {
	jsdata, err := f.retrieve(userId)
	if err != nil {
		return nil, err
	}
	if jsdata == nil {
		return nil, nil
	}
	feed := new(JSONFeed)
	err = json.Unmarshal(jsdata, feed)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func (f *FeedRetriever) retrieve(userId string) ([]byte, error) {
	// There's a query escape, but not a url escape. Wha?
	urlNoKey := "https://www.googleapis.com/plus/v1/people/" + userId + "/activities/public?key="
	url := urlNoKey + f.Key
	log.Printf("Plus Get: %s", urlNoKey)

	r, err := f.Client.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == 404 {
		return nil, nil
	}

	if r.StatusCode != 200 {
		return nil, InvalidStatusCode(r.StatusCode)
	}

	defer r.Body.Close()

	var body []byte
	var readErr error
	var n int
	for readErr == nil {
		suffix := make([]byte, 2e5) // 2K
		n, readErr = r.Body.Read(suffix)
		if n < len(suffix) {
			suffix = suffix[:n]
		}
		body = append(body, suffix...)
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	return body, err
}

type JSONImage struct {
	URL    string `json:"url"`
	Type   string // optional (e.g. profile images)
	Height int64  // optional (e.g. profile images)
	Width  int64  // optional (e.g. profile images)
}

type JSONActor struct {
	Id          string
	DisplayName string
	URL         string
	Image       JSONImage
}

type JSONPlusObject struct {
	ObjectType string
	Id         string
	Actor      JSONActor
	Content    string
	URL        string
	// TODO: replies, plusoners, resharers
}

type JSONActivity struct {
	Kind            string
	JTitle          string `json:"title"`
	JPublished      string `json:"published"`
	JUpdated        string `json:"updated"`
	JId             string `json:"id"`
	JURL            string `json:"url"`
	Actor           JSONActor
	JVerb           string `json:"verb"`
	Object          JSONPlusObject
	Annotation      string
	CrosspostSource string `json:"crosspostSource"`
	// TODO: provider, access, attachments
}

type JSONFeed struct {
	JTitle   string          `json:"title"`
	JUpdated string          `json:"updated"`
	JId      string          `json:"id"`
	JItems   []*JSONActivity `json:"items"`
	Actor    JSONActor
}

func (j *JSONFeed) Title() string {
	return j.JTitle
}

func (j *JSONFeed) Id() string {
	return j.JId
}

func (j *JSONFeed) Updated() string {
	return j.JUpdated
}

func (j *JSONFeed) Items() []Activity {
	acts := make([]Activity, 0, len(j.JItems))
	for _, a := range j.JItems {
		acts = append(acts, a)
	}
	return acts
}

func (a *JSONActivity) Verb() string {
	return a.JVerb
}

func (a *JSONActivity) Published() string {
	return a.JPublished
}

func (a *JSONActivity) Updated() string {
	return a.JUpdated
}

func (a *JSONActivity) Content() string {
	return a.Object.Content
}

func (a *JSONActivity) Title() string {
	return a.JTitle
}

func (a *JSONActivity) Id() string {
	return a.JId
}

func (a *JSONActivity) URL() string {
	return a.JURL
}

func (a *JSONActivity) ActorName() string {
	return a.Actor.DisplayName
}
