package main

import (
	"flag"
	"gorilla.googlecode.com/hg/gorilla/mux"
	"http"
	"log"
	"os"
)

const (
	frontendReadTimeoutNanos  = 4e8 // 400 millis
	frontendWriteTimeoutNanos = 4e8 // 400 millis
)

type Service interface {
	ShutdownChan() chan string
}

// TODO: fix timestamps, add attachments, handle posts that were reshares
func main() {
	var frontendHost string
	var frontendAddr string
	var clientSecret string

	flag.StringVar(&frontendHost, "host", "localhost", "the Host header to respond to in the frontend")
	flag.StringVar(&frontendAddr, "http", "127.0.0.1:6543", "address to run the frontend o (e.g. :6543, localhost:4321)")
	flag.StringVar(&clientSecret, "csecret", "", "OAuth2 consumer secret for Google")
	flag.Parse()

	fs, err := feedStorage(clientSecret)
	if err != nil {
		log.Fatalf("Could not boot feed storage: %s", err)
	}
	FeedStore = fs

	f, err := frontend(frontendHost, frontendAddr, frontendReadTimeoutNanos, frontendWriteTimeoutNanos)
	if err != nil {
		log.Fatalf("Could not boot frontend: %s", err)
	}

	msg := <-f.ShutdownChan()
	log.Printf("frontend shutdown: %s", msg)
}

func feedStorage(clientSecret string) (FeedStorage, os.Error) {
	retriever := new(FeedRetriever)
	retriever.Client = http.DefaultClient
	retriever.Key = clientSecret
	return retriever, nil
}

func frontend(host, addr string, readTimeout, writeTimeout int64) (Service, os.Error) {
	router := mux.Host(host).NewRouter()
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		log.Printf("NotFound Vars: %#v\n", vars)
		log.Printf("NotFound Raw URL: %s\n", r.URL.Raw)
		w.WriteHeader(http.StatusNotFound)
	})
	server := &http.Server{addr, router, readTimeout, writeTimeout, 0}
	f := &Frontend{router, make(chan string), server}
	f.initRoutes(router)
	err := f.Start()
	if err != nil {
		return nil, err
	}
	return f, nil
}
