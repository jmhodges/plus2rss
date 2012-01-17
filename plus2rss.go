package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Service interface {
	ShutdownChan() chan string
}

// This global sucks.
var FeedStore FeedStorage

// TODO: fix timestamps, add attachments, handle posts that were reshares
func main() {
	var frontendHost string
	var frontendAddr string
	var simpleKeyFile string
	var templateDir string
	var frontendReadTimeout = time.Duration(4e8) // 400 millis
	var frontendWriteTimeout = time.Duration(4e8) // 400 millis

	flag.StringVar(&frontendHost, "vhost", "localhost:6543", "the virtual Host header to respond to in the frontend")
	flag.StringVar(&frontendAddr, "http", "localhost:6543", "address to run the frontend o (e.g. :6543, localhost:4321)")
	flag.StringVar(&simpleKeyFile, "simpleKeyFile", "", "File containing simple API access key for Google")
	flag.StringVar(&templateDir, "templateDir", ".", "Directory containing the templates to render html and feeds")

	flag.DurationVar(&frontendReadTimeout, "frontendReadTimeout", frontendReadTimeout, "frontend http server's socket read timeout")
	flag.DurationVar(&frontendWriteTimeout, "frontendWriteTimeout", frontendWriteTimeout, "frontend http server's socket write timeout")
	flag.Parse()

	fs, err := feedStorage(simpleKeyFile)
	if err != nil {
		log.Fatalf("Could not boot feed storage: %s", err)
	}
	FeedStore = fs

	f, server := frontend(frontendHost, frontendAddr, templateDir, frontendReadTimeout, frontendWriteTimeout)
	_ = server
	err = <-f.ShutdownChan()
	log.Printf("frontend shutdown: %s", err)
}

func feedStorage(simpleKeyFile string) (FeedStorage, error) {
	if strings.Trim(simpleKeyFile, " \r\n\t") == "" {
		return nil, errors.New("Google API client simple key file must be given")
	}
	simpleKeySlice, err := ioutil.ReadFile(simpleKeyFile)
	if err != nil {
		return nil, err
	}
	simpleKey := strings.Trim(string(simpleKeySlice), " \r\n\t")
	retriever := &FeedRetriever{http.DefaultClient, simpleKey}
	return retriever, nil
}

func frontend(host, addr, templateDir string, readTimeout, writeTimeout time.Duration) (*Frontend, *http.Server) {
	f := NewFrontend(host, templateDir)
	server := &http.Server{addr, f, readTimeout, writeTimeout, 0}

	go func() {
		err := server.ListenAndServe()
		f.ShutdownChan() <- err
	}()
	return f, server
}
