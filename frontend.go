package main

import (
	"gorilla.googlecode.com/hg/gorilla/mux"
	"http"
	"log"
	"os"
	"regexp"
	"template"
)

type Frontend struct {
	router *mux.Router
	shutdownChan chan string
	server *http.Server
}

func (f *Frontend) initRoutes(router *mux.Router) {
	router.HandleFunc(`/u/{user_id:\d+}`, UserFeed).Methods("GET", "HEAD")
	router.HandleFunc(`/`, AskForURL).Methods("GET", "HEAD")
	router.HandleFunc(`/plus/enqueue`, EnqueueURLOrUserId).Methods("POST")
}

func (f *Frontend) Start() os.Error {
	go func() {
		err := f.server.ListenAndServe()
		f.shutdownChan <- err.String()
	}()
	return nil
}

func (f *Frontend) ShutdownChan() chan string {
	return f.shutdownChan
}

// Handlers
var feedTemplate = template.Must(template.ParseFile("feed.template.xml"))
type FeedWithRequest struct {
	Feed
	Req *http.Request
}

func UserFeed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) // FIXME this is one massive lock and i hate it.
	userId := PlausibleUserId(vars["user_id"])
	if userId == "" {
		// TODO: flash[:notice] thing
		// user name seems invalid
		NoSuchFeed(w, r)
		return
	}

	feed, err := FeedStore.Find(userId)

	if err != nil {
		log.Printf("Finding the feed for a user blew up: %s", err)
		Sigh500(w, r)
		return
	}

	if feed == nil {
		NoSuchFeed(w, r)
		return
	}

	w.Header().Add("Content-Type", `application/atom+xml; charset="utf-8"`)
	w.WriteHeader(http.StatusOK)
	feedWithR := &FeedWithRequest{feed, r}
	err = feedTemplate.Execute(w, feedWithR)
	if err != nil {
		log.Printf("UserFeed template execute: %s", err)
	}

}

var askForURLTemplate = template.Must(template.ParseFile("ask_for_url.template.html"))
func AskForURL(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := askForURLTemplate.Execute(w, nil)
	if err != nil {
		log.Printf("AskForUrl template execute: %s", err)
	}
}

func EnqueueURLOrUserId(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("client error on %s: %s\n", r.URL.Raw, err)
		// TODO: flash[:notice] thing
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	urlOrUserId := r.FormValue("url_or_user_id")
	if urlOrUserId == "" {
		// TODO: flash[:notice] thing
		// user name seems invalid
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	userId := PlausibleUserId(urlOrUserId)

	http.Redirect(w, r, "/u/"+userId, http.StatusFound)

	return
}

func NoSuchFeed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("No such feed.\n"))
}

var justUserIdR = regexp.MustCompile(`^\d+$`)
var plusUrlR = regexp.MustCompile(`^https?://plus.google.com/(\d+)/?`)
func PlausibleUserId(urlOrUserId string) string {
	if justUserIdR.MatchString(urlOrUserId) {
		return urlOrUserId
	}
	m := plusUrlR.FindStringSubmatch(urlOrUserId)
	if m != nil && len(m) == 2 {
		return m[1]
	}
	return ""
}

var Body500 = []byte("Something went wrong. Wait a minute, please.\n")
func Sigh500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(Body500)
}

var Body503 = []byte("Taking too long.\n")
func Sigh503(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(Body503)
}
