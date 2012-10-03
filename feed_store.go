package main

import (
	"code.google.com/p/google-api-go-client/plus/v1"
	"fmt"
	"log"
)

type InvalidStatusCode struct {
	code     int
	urlNoKey string
}

func (isc *InvalidStatusCode) Error() string {
	return fmt.Sprintf("Error status code %d from Google+ url %s", isc.code, isc.urlNoKey)
}

type FeedRetriever struct {
	Client *plus.Service
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
	actor *plus.Person
	feed  *plus.ActivityFeed
}

// TODO An obvious place to cache data.
func (f *FeedRetriever) Find(userId string) (Feed, error) {
	actor, err := f.retrievePerson(userId)
	if err != nil {
		return nil, err
	}
	feed, err := f.retrieveActivities(userId)
	if err != nil {
		return nil, err
	}
	return &ActorFeed{actor, feed}, nil
}

func (f *FeedRetriever) retrievePerson(userId string) (*plus.Person, error) {
	log.Printf("Person: %s", userId)
	return f.Client.People.Get(userId).Do()
}

func (f *FeedRetriever) retrieveActivities(userId string) (*plus.ActivityFeed, error) {
	log.Printf("List Public Activities of User: %s", userId)
	return f.Client.Activities.List(userId, "public").Do()
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
	pI plus.ActivityObjectAttachmentsImage
}

type JSONAttachment struct {
	pAO plus.ActivityObjectAttachments
}

type JSONActor struct {
	Id          string     `json:"id"`
	DisplayName string     `json:"displayName"`
	URL         string     `json:"url"`
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
	pA plus.Activity
}

type JSONFeed struct {
	JTitle   string          `json:"title"`
	JUpdated string          `json:"updated"`
	JId      string          `json:"id"`
	JItems   []*JSONActivity `json:"items"`
}

func (j *ActorFeed) Title() string {
	return j.feed.Title
}

func (j *ActorFeed) Id() string {
	return j.feed.Id
}

func (j *ActorFeed) Updated() string {
	return j.feed.Updated
}

func (j *ActorFeed) Items() []Activity {
	acts := make([]Activity, len(j.feed.Items))
	for i, a := range j.feed.Items {
		acts[i] = &JSONActivity{*a}
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
	return a.pA.Verb
}

func (a *JSONActivity) Published() string {
	return a.pA.Published
}

func (a *JSONActivity) Updated() string {
	return a.pA.Updated
}

func (a *JSONActivity) Content() string {
	return a.pA.Object.Content
}

func (a *JSONActivity) Title() string {
	return a.pA.Title
}

func (a *JSONActivity) Id() string {
	return a.pA.Id
}

func (a *JSONActivity) URL() string {
	return a.pA.Url
}

func (a *JSONActivity) ActorName() string {
	return a.pA.Actor.DisplayName
}

func (a *JSONActivity) Attachments() []Attachment {
	as := make([]Attachment, len(a.pA.Object.Attachments))
	for i, ao := range a.pA.Object.Attachments {
		as[i] = &JSONAttachment{*ao}
	}
	return as
}

func (a *JSONAttachment) ObjectType() string {
	return a.pAO.ObjectType
}
func (a *JSONAttachment) DisplayName() string {
	return a.pAO.DisplayName
}
func (a *JSONAttachment) Id() string {
	return a.pAO.Id
}
func (a *JSONAttachment) Content() string {
	return a.pAO.Content
}
func (a *JSONAttachment) URL() string {
	return a.pAO.Url
}
func (a *JSONAttachment) Image() Image {
	return &JSONImage{*a.pAO.Image}
}
func (a *JSONAttachment) FullImage() Image {
	fi := a.pAO.FullImage
	ai := plus.ActivityObjectAttachmentsImage{fi.Height, fi.Type, fi.Url, fi.Width}
	return &JSONImage{ai}
}

func (a *JSONAttachment) IsVideo() bool {
	return a.pAO.ObjectType == "video"
}
func (a *JSONAttachment) IsPhoto() bool {
	return a.pAO.ObjectType == "photo"
}
func (a *JSONAttachment) IsArticle() bool {
	return a.pAO.ObjectType == "article"
}

func (i *JSONImage) URL() string {
	return i.pI.Url
}

func (i *JSONImage) Type() string {
	return i.pI.Type
}

func (i *JSONImage) Height() int64 {
	return i.pI.Height
}

func (i *JSONImage) Width() int64 {
	return i.pI.Width
}
