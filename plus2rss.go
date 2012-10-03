package main

import (
	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/plus/v1"
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

const socketTimeout = time.Duration(400 * time.Millisecond)

var (
	frontendHost         = flag.String("vhost", "localhost:6543", "the virtual Host header to respond to in the frontend")
	frontendAddr         = flag.String("http", "localhost:6543", "address to run the frontend on (e.g. :6543, localhost:4321)")
	oauthSecretFile      = flag.String("oauthSecretFile", "", "file containing a working Google OAuth2 client id and secret")
	oauthCacheFile       = flag.String("oauthCacheFile", "", "file containing a cached Google OAuth2 access and refresh token")
	templateDir          = flag.String("templateDir", "./templates", "Directory containing the templates to render html and feeds")
	frontendReadTimeout  = flag.Duration("frontendReadTimeout", socketTimeout, "frontend http server's socket read timeout")
	frontendWriteTimeout = flag.Duration("frontendWriteTimeout", socketTimeout, "frontend http server's socket write timeout")
)

// TODO: fix timestamps, add attachments, handle posts that were reshares
func main() {
	flag.Parse()

	if *oauthSecretFile == "" {
		log.Fatalf("plus2rss: -oauthSecretFile=FILE is a required command-line argument")
	}

	if *oauthCacheFile == "" {
		log.Fatalf("plus2rss: -oauthCacheFile=FILE is a required command-line argument")
	}

	fs, err := feedStorage(*oauthSecretFile, *oauthCacheFile)
	if err != nil {
		log.Fatalf("Could not boot feed storage: %s", err)
	}

	f := frontend(fs, *frontendHost, *frontendAddr, *templateDir, *frontendReadTimeout, *frontendWriteTimeout)
	err = <-f.ShutdownChan()
	log.Printf("frontend shutdown: %s", err)
}

func feedStorage(secretFile, cacheFile string) (FeedStorage, error) {
	oauthSecret, err := ioutil.ReadFile(secretFile)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(oauthSecret), "\n")
	if len(lines) < 3 {
		return nil, errors.New("not enough lines in oauth config file")
	}

	clientId := strings.TrimSpace(lines[0])
	secret := strings.TrimSpace(lines[1])

	c := &oauth.Config{
		ClientId:     clientId,
		ClientSecret: secret,
		Scope:        plus.PlusMeScope,
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
	}

	transport := &oauth.Transport{
		Config:    c,
		Transport: http.DefaultTransport,
	}
	tokenCache := oauth.CacheFile(cacheFile)
	tok, err := tokenCache.Token()
	if err != nil {
		return nil, err
	}

	// Handle refreshing tokens outside of this process
	transport.Token = &oauth.Token{AccessToken: tok.AccessToken}
	srv, err := plus.New(transport.Client())
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
