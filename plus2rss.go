package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"
)

type Service interface {
	ShutdownChan() chan string
}

// TODO: fix timestamps, add attachments, handle posts that were reshares
func main() {
	var frontendHost string
	var frontendAddr string
	var simpleKey string
	var frontendReadTimeout = time.Duration(4e8) // 400 millis
	var frontendWriteTimeout = time.Duration(4e8) // 400 millis

	flag.StringVar(&frontendHost, "vhost", "localhost:6543", "the virtual Host header to respond to in the frontend")
	flag.StringVar(&frontendAddr, "http", "localhost:6543", "address to run the frontend o (e.g. :6543, localhost:4321)")
	flag.StringVar(&simpleKey, "simpleKey", "", "Simple API Access key for Google")
	flag.DurationVar(&frontendReadTimeout, "frontendReadTimeout", frontendReadTimeout, "frontend http server's socket read timeout")
	flag.DurationVar(&frontendWriteTimeout, "frontendWriteTimeout", frontendWriteTimeout, "frontend http server's socket write timeout")
	flag.Parse()

	fs, err := feedStorage(simpleKey)
	if err != nil {
		log.Fatalf("Could not boot feed storage: %s", err)
	}
	FeedStore = fs

	f, server := frontend(frontendHost, frontendAddr, frontendReadTimeout, frontendWriteTimeout)
	_ = server
	err = <-f.ShutdownChan()
	log.Printf("frontend shutdown: %s", err)
}

func feedStorage(simpleKey string) (FeedStorage, error) {
	if strings.Trim(simpleKey, " \r\n\t") == "" {
		return nil, errors.New("Google API client secret cannot be blank")
	}
	retriever := &FeedRetriever{http.DefaultClient, simpleKey}
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
