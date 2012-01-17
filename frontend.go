package main

import (
	"log"
	"net/http"
	"regexp"
	"text/template"
)

type Frontend struct {
	host         string
	shutdownChan chan error
	askForURLTemplate *template.Template
	feedTemplate *template.Template
}

type FeedWithRequest struct {
	Feed
	Req *http.Request
}

var userIdPath = regexp.MustCompile(`/u/(\d+)`)

var justUserIdR = regexp.MustCompile(`^\d+$`)
var plusUrlR = regexp.MustCompile(`^https?://plus.google.com/(\d+)/?`)

var Body404 = []byte("No such feed.\n")
var Body500 = []byte("Something went wrong. Wait a minute, please.\n")
var Body503 = []byte("Taking too long.\n")

func NewFrontend(host string, templateDir string) *Frontend {
	ch := make(chan error)
	askForURLTemplate := template.Must(template.ParseFiles(templateDir + "/ask_for_url.template.html"))
	feedTemplate := template.Must(template.ParseFiles(templateDir + "/feed.template.xml"))
	return &Frontend{host, ch, askForURLTemplate, feedTemplate}
}

//   GET / -> AskForURL (HEAD, too)
//   GET /u/some_user_id -> UserFeed() (HEAD, too)
//   POST /plus/enqueue -> CheckURLOrUserId
func (f *Frontend) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Host != f.host {
		log.Printf("Request asked for %s, expected %s", r.Host, f.host)
		http.Redirect(w, r, "http://"+f.host, 302)
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Unable to parse request"))
		return
	}

	uMatch := userIdPath.FindAllStringSubmatch(r.URL.Path, -1)
	if len(uMatch) > 0 && len(uMatch[0][1]) > 0 && (r.Method == "GET" || r.Method == "HEAD") {
		r.Form.Add("user_id", uMatch[0][1])
		f.UserFeed(w, r)
		return
	}

	if r.URL.Path == "/plus/enqueue" && r.Method == "POST" {
		f.CheckURLOrUserId(w, r)
		return
	}

	if r.URL.Path == "/" && (r.Method == "GET" || r.Method == "HEAD") {
		f.AskForURL(w, r)
		return
	}

	log.Printf("NotFound Vars: %#v\n", r.Form)
	log.Printf("NotFound Raw URL: %s\n", r.URL)
	w.WriteHeader(http.StatusNotFound)
	return
}

func (f *Frontend) ShutdownChan() chan error {
	return f.shutdownChan
}

// Handlers
func (f *Frontend) UserFeed(w http.ResponseWriter, r *http.Request) {
	userId := PlausibleUserId(r.Form.Get("user_id"))
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
	err = f.feedTemplate.Execute(w, feedWithR)
	if err != nil {
		log.Printf("UserFeed template execute: %s", err)
	}
}

func (f *Frontend) AskForURL(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := f.askForURLTemplate.Execute(w, nil)
	if err != nil {
		log.Printf("AskForUrl template execute: %s", err)
	}
}

func (f *Frontend) CheckURLOrUserId(w http.ResponseWriter, r *http.Request) {
	urlOrUserId := r.FormValue("url_or_user_id")
	if urlOrUserId == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	userId := PlausibleUserId(urlOrUserId)

	if userId == "" {
		// TODO: flash[:notice] thing
		// user name seems invalid
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/u/"+userId, http.StatusFound)

	return
}

func NoSuchFeed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write(Body404)
}

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

func Sigh500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(Body500)
}

func Sigh503(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(Body503)
}
