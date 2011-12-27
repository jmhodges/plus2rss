package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	frontendReadTimeoutNanos  = time.Duration(4e8) // 400 millis
	frontendWriteTimeoutNanos = time.Duration(4e8) // 400 millis
)

type Service interface {
	ShutdownChan() chan string
}

// TODO: fix timestamps, add attachments, handle posts that were reshares
func main() {
	var frontendHost string
	var frontendAddr string
	var clientSecret string

	flag.StringVar(&frontendHost, "vhost", "localhost:6543", "the virtual Host header to respond to in the frontend")
	flag.StringVar(&frontendAddr, "http", "localhost:6543", "address to run the frontend o (e.g. :6543, localhost:4321)")
	flag.StringVar(&clientSecret, "csecret", "", "OAuth2 consumer secret for Google")
	flag.Parse()

	fs, err := feedStorage(clientSecret)
	if err != nil {
		log.Fatalf("Could not boot feed storage: %s", err)
	}
	FeedStore = fs

	f, server := frontend(frontendHost, frontendAddr, frontendReadTimeoutNanos, frontendWriteTimeoutNanos)
	_ = server
	err = <-f.ShutdownChan()
	log.Printf("frontend shutdown: %s", err)
}

func feedStorage(clientSecret string) (FeedStorage, error) {
	if strings.Trim(clientSecret, " \r\n\t") == "" {
		return nil, errors.New("Google API client secret cannot be blank")
	}
	retriever := &FeedRetriever{http.DefaultClient, clientSecret}
	return retriever, nil
}

func frontend(host, addr string, readTimeout, writeTimeout time.Duration) (*Frontend, *http.Server) {
	f := &Frontend{host, make(chan error)}
	server := &http.Server{addr, f, readTimeout, writeTimeout, 0}

	go func() {
		err := server.ListenAndServe()
		f.ShutdownChan() <- err
	}()
	return f, server
}
