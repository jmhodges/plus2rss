package main

import (
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

const socketTimeout = time.Duration(400 * time.Millisecond)

var (
	frontendHost         = flag.String("vhost", "localhost:6543", "the virtual Host header to respond to in the frontend")
	frontendAddr         = flag.String("http", "localhost:6543", "address to run the frontend on (e.g. :6543, localhost:4321)")
	simpleKeyFile        = flag.String("simpleKeyFile", "", "File containing simple API access key for Google")
	templateDir          = flag.String("templateDir", "./templates", "Directory containing the templates to render html and feeds")
	frontendReadTimeout  = flag.Duration("frontendReadTimeout", socketTimeout, "frontend http server's socket read timeout")
	frontendWriteTimeout = flag.Duration("frontendWriteTimeout", socketTimeout, "frontend http server's socket write timeout")
)

// TODO: fix timestamps, add attachments, handle posts that were reshares
func main() {
	flag.Parse()

	if *simpleKeyFile == "" {
		log.Fatalf("plus2rss: -simpleKeyFile=FILE is a required command-line argument")
	}

	fs, err := feedStorage(*simpleKeyFile)
	if err != nil {
		log.Fatalf("Could not boot feed storage: %s", err)
	}

	f := frontend(fs, *frontendHost, *frontendAddr, *templateDir, *frontendReadTimeout, *frontendWriteTimeout)
	err = <-f.ShutdownChan()
	log.Printf("frontend shutdown: %s", err)
}

func feedStorage(simpleKeyFile string) (FeedStorage, error) {
	simpleKeySlice, err := ioutil.ReadFile(simpleKeyFile)
	if err != nil {
		return nil, err
	}
	simpleKey := strings.TrimSpace(string(simpleKeySlice))
	retriever := &FeedRetriever{http.DefaultClient, simpleKey}
	return retriever, nil
}

func frontend(fs FeedStorage, host, addr, templateDir string, readTimeout, writeTimeout time.Duration) *Frontend {
	f := NewFrontend(fs, host, templateDir)
	server := &http.Server{addr, f, readTimeout, writeTimeout, 0, nil}

	go func() {
		f.ShutdownChan() <- server.ListenAndServe()
	}()
	return f
}
