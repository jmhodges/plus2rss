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
