package main

import (
	"code.google.com/p/google-api-go-client/plus/v1"
	"github.com/rcrowley/go-metrics"
	"log"
)

var (
	findAttempts  = metrics.NewCounter()
	findSuccesses = metrics.NewCounter()
	findFailures  = metrics.NewCounter()
)

func init() {
	registry.Register("feed_retriever_find_attempts", findAttempts)
	registry.Register("feed_retriever_find_successes", findSuccesses)
	registry.Register("feed_retriever_find_failures", findFailures)
}

type FeedRetriever struct {
	client *plus.Service
}

type FeedStorage interface {
	Find(string) (Feed, error)
}

type personUnion struct {
	actor *plus.Person
	err   error
}

// TODO An obvious place to cache data.
func (f *FeedRetriever) Find(userId string) (Feed, error) {
	findAttempts.Inc(1)
	ch := make(chan personUnion)
	go func() {
		actor, err := f.retrievePerson(userId)
		if err != nil {
			ch <- personUnion{nil, err}
			return
		}
		ch <- personUnion{actor, nil}
	}()

	feed, err := f.retrieveActivities(userId)
	if err != nil {
		findFailures.Inc(1)
		return nil, err
	}

	u := <-ch
	if u.err != nil {
		findFailures.Inc(1)
		return nil, u.err
	}
	findSuccesses.Inc(1)
	return &ActorFeed{u.actor, feed}, nil
}

func (f *FeedRetriever) retrievePerson(userId string) (*plus.Person, error) {
	log.Printf("Person: %s", userId)
	return f.client.People.Get(userId).Do()
}

func (f *FeedRetriever) retrieveActivities(userId string) (*plus.ActivityFeed, error) {
	log.Printf("List Public Activities of User: %s", userId)
	return f.client.Activities.List(userId, "public").Do()
}
