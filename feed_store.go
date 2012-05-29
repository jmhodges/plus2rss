package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)


type InvalidStatusCode struct {
	code int
	urlNoKey string
}

func (isc *InvalidStatusCode) Error() string {
	return fmt.Sprintf("Error status code %d from Google+ url %s", isc.code, isc.urlNoKey)
}

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
	ActorName() string
	ActorId() string
}

type ActorFeed struct {
	actor *JSONActor
	feed  *JSONFeed
}

// TODO An obvious place to cache data.
func (f *FeedRetriever) Find(userId string) (Feed, error) {
	personData, err := f.retrievePerson(userId)
	if err != nil || personData == nil {
		return nil, err
	}
	actor := new(JSONActor)
	err = json.Unmarshal(personData, actor)
	if err != nil {
		return nil, err
	}
	jsdata, err := f.retrieveActivities(userId)
	if err != nil || jsdata == nil {
		return nil, err
	}
	feed := new(JSONFeed)
	err = json.Unmarshal(jsdata, feed)
	if err != nil {
		return nil, err
	}
	return &ActorFeed{actor, feed}, nil
}

func (f *FeedRetriever) retrievePerson(userId string) ([]byte, error) {
	urlNoKey :=  "https://www.googleapis.com/plus/v1/people/" + userId + "?key="
	log.Printf("Person: %s", urlNoKey)
	return f.get(urlNoKey)
}

func (f *FeedRetriever) retrieveActivities(userId string) ([]byte, error) {
	// There's a query escape, but not a url escape. Wha?
	urlNoKey := "https://www.googleapis.com/plus/v1/people/" + userId + "/activities/public?key="
	log.Printf("List Public Activities: %s", urlNoKey)
	return f.get(urlNoKey)
}

func (f *FeedRetriever) get(urlNoKey string) ([]byte, error) {
	url := urlNoKey + f.Key
	r, err := f.Client.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == 404 {
		return nil, nil
	}

	if r.StatusCode != 200 {
		return nil, &InvalidStatusCode{r.StatusCode, urlNoKey}
	}

	body := new(bytes.Buffer)
	_, err = io.Copy(body, r.Body)

	if err != nil && err != io.EOF {
		return nil, err
	}

	r.Body.Close()
	return body.Bytes(), err
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
	Attachments() []Attachment
}

type Attachment interface {
	ObjectType() string
	DisplayName() string
	Id() string
	Content() string
	URL() string
	Image() Image
	FullImage() Image
	IsVideo() bool
	IsPhoto() bool
	IsArticle() bool
}

type Image interface {
	URL() string
	Type() string
	Height() int64
	Width() int64
}

type Embed interface {
	URL() string
	Type() string
}

type JSONImage struct {
	JURL    string `json:"url"`
	JType   string `json:"type"`   // optional (e.g. profile images)
	JHeight int64  `json:"height"` // optional (e.g. profile images)
	JWidth  int64  `json:"width"`  // optional (e.g. profile images)
}

// Embeddable video link
type JSONAttachment struct {
	JObjectType  string     `json:"objectType"` // "video", "photo", or "article"
	JDisplayName string     `json:"displayName"`
	JId          string     `json:"id"`
	JContent     string     `json:"content"` // snippet of text if ObjectType == "article"
	JURL         string     `json:"url"`     // link to the attachment, is of type text/html
	JImage       *JSONImage `json:"image"`
	JFullImage   *JSONImage `json:"fullImage"`
}

type JSONActor struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayName"`
	URL         string `json:"url"`
	Image       *JSONImage `json:"image"`
}

type JSONPlusObject struct {
	ObjectType   string
	Id           string
	Actor        JSONActor
	Content      string
	URL          string
	JAttachments []*JSONAttachment `json:"attachments"`
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
	Object          *JSONPlusObject
	Annotation      string
	CrosspostSource string `json:"crosspostSource"`
	// TODO: provider, access
}

type JSONFeed struct {
	JTitle   string          `json:"title"`
	JUpdated string          `json:"updated"`
	JId      string          `json:"id"`
	JItems   []*JSONActivity `json:"items"`
}

func (j *ActorFeed) Title() string {
	return j.feed.JTitle
}

func (j *ActorFeed) Id() string {
	return j.feed.JId
}

func (j *ActorFeed) Updated() string {
	return j.feed.JUpdated
}

func (j *ActorFeed) Items() []Activity {
	acts := make([]Activity, 0, len(j.feed.JItems))
	for _, a := range j.feed.JItems {
		acts = append(acts, a)
	}
	return acts
}

func (j *ActorFeed) ActorName() string {
	return j.actor.DisplayName
}

func (j *ActorFeed) ActorId() string {
	return j.actor.Id
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

func (a *JSONActivity) Attachments() []Attachment {
	as := make([]Attachment, len(a.Object.JAttachments))
	for i, ao := range a.Object.JAttachments {
		as[i] = ao
	}
	return as
}

func (a *JSONAttachment) ObjectType() string {
	return a.JObjectType
}
func (a *JSONAttachment) DisplayName() string {
	return a.JDisplayName
}
func (a *JSONAttachment) Id() string {
	return a.JId
}
func (a *JSONAttachment) Content() string {
	return a.JContent
}
func (a *JSONAttachment) URL() string {
	return a.JURL
}
func (a *JSONAttachment) Image() Image {
	return a.JImage
}
func (a *JSONAttachment) FullImage() Image {
	return a.JFullImage
}

func (a *JSONAttachment) IsVideo() bool {
	return a.JObjectType == "video"
}
func (a *JSONAttachment) IsPhoto() bool {
	return a.JObjectType == "photo"
}
func (a *JSONAttachment) IsArticle() bool {
	return a.JObjectType == "article"
}

func (i *JSONImage) URL() string {
	return i.JURL
}

func (i *JSONImage) Type() string {
	return i.JType
}

func (i *JSONImage) Height() int64 {
	return i.JHeight
}

func (i *JSONImage) Width() int64 {
	return i.JWidth
}
