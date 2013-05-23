package main

import (
	"code.google.com/p/google-api-go-client/plus/v1"
	"log"
)

type FeedRetriever struct {
	client *plus.Service
	lg     *log.Logger
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

// TODO An obvious place to cache data.
func (f *FeedRetriever) Find(userId string) (feed Feed, err error) {
	findAttempts.Inc(1)
	findTimer.Time(func() { feed, err = f.find(userId) })
	if err == nil {
		findSuccesses.Inc(1)
	} else {
		findFailures.Inc(1)
	}
	return feed, err
}

func (f *FeedRetriever) find(userId string) (Feed, error) {
	ch := make(chan error)
	var actor *plus.Person
	go func() {
		var pErr error
		actor, pErr = f.retrievePerson(userId)
		ch <- pErr
	}()

	feed, err := f.retrieveActivities(userId)
	if err != nil {
		return nil, err
	}

	err = <-ch
	if err != nil {
		return nil, err
	}
	return &ActorFeed{actor, feed}, nil
}

func (f *FeedRetriever) retrievePerson(userId string) (*plus.Person, error) {
	f.lg.Printf("Person: %s", userId)
	return f.client.People.Get(userId).Do()
}

func (f *FeedRetriever) retrieveActivities(userId string) (*plus.ActivityFeed, error) {
	f.lg.Printf("List Public Activities of User: %s", userId)
	return f.client.Activities.List(userId, "public").Do()
}
