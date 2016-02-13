package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rcrowley/go-metrics"
	"google.golang.org/api/plus/v1"
)

type Service interface {
	ShutdownChan() chan string
}

const (
	timeout = time.Duration(5 * time.Second)
)

var (
	frontendHost         = flag.String("vhost", "localhost:6543", "the virtual Host header to respond to in the frontend")
	frontendAddr         = flag.String("http", "localhost:6543", "address to run the frontend on (e.g. :6543, localhost:4321)")
	simpleKeyFile        = flag.String("simpleKeyFile", "", "file containing a working Google simple key")
	templateDir          = flag.String("templateDir", "./templates", "Directory containing the templates to render html and feeds")
	frontendReadTimeout  = flag.Duration("frontendReadTimeout", timeout, "frontend http server's total request read timeout")
	frontendWriteTimeout = flag.Duration("frontendWriteTimeout", timeout, "frontend http server's total request write timeout")
	controlAddr          = flag.String("controlAddr", "localhost:5432", "the address to run the control HTTP server on")
	registry             = metrics.NewRegistry()
	bootTime             = time.Now().UTC()
)

// TODO: handle posts that were reshares
func main() {
	flag.Parse()
	lg := log.New(os.Stderr, "", 0)
	if *simpleKeyFile == "" {
		lg.Fatalf("plus2rss: -simpleKeyFile=FILE is a required command-line argument")
	}

	ch := make(chan error)
	cs := NewStatServer(*controlAddr)
	go func() {
		ch <- cs.ListenAndServe()
	}()

	fs, err := feedStorage(*simpleKeyFile, lg)
	if err != nil {
		lg.Fatalf("Could not boot feed storage: %s", err)
	}

	fr := frontend(fs, *frontendHost, *frontendAddr, *templateDir, *frontendReadTimeout, *frontendWriteTimeout)
	go func() {
		ch <- fr.ListenAndServe()
	}()

	err = <-ch
	lg.Printf("frontend shutdown: %s", err)
}

func feedStorage(simpleFile string, lg *log.Logger) (FeedStorage, error) {
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
	retriever := &FeedRetriever{srv, lg}
	return retriever, nil
}

func frontend(fs FeedStorage, host, addr, templateDir string, readTimeout, writeTimeout time.Duration) *http.Server {
	m := NewFrontendMux(fs, host, templateDir)
	return &http.Server{Addr: addr, Handler: m, ReadTimeout: readTimeout, WriteTimeout: writeTimeout}
}
