package main

import (
	"code.google.com/p/google-api-go-client/plus/v1"
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
	simpleKeyFile      = flag.String("simpleKeyFile", "", "file containing a working Google simple key")
	templateDir          = flag.String("templateDir", "./templates", "Directory containing the templates to render html and feeds")
	frontendReadTimeout  = flag.Duration("frontendReadTimeout", socketTimeout, "frontend http server's socket read timeout")
	frontendWriteTimeout = flag.Duration("frontendWriteTimeout", socketTimeout, "frontend http server's socket write timeout")
)

// TODO: handle posts that were reshares
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

func feedStorage(simpleFile string) (FeedStorage, error) {
	simpleKey, err := ioutil.ReadFile(simpleFile)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(string(simpleKey)) // FIXME make json
	t := &SimpleKeyTransport{Key: key, Transport: http.DefaultTransport}
	srv, err := plus.New(&http.Client{Transport: t})
	if err != nil {
		return nil, err
	}
	retriever := &FeedRetriever{srv}
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
