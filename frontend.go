package main

import (
	"bytes"
	"log"
	"net/http"
	"regexp"
	"text/template"
)

type Frontend struct {
	host              string
	feedStore         FeedStorage
	askForURLTemplate *template.Template
	feedTemplate      *template.Template
	feedMetaTemplate  *template.Template
}

type FeedView struct {
	Feed
	Req *http.Request
}

var userIdPath = regexp.MustCompile(`/u/(\d+)$`)
var userIdMetaPath = regexp.MustCompile(`/u_meta/(\d+)$`)
var justUserIdR = regexp.MustCompile(`^\d+$`)
var plusUrlR = regexp.MustCompile(`^https?://plus.google.com/(\d+)/?`)

var Body404 = []byte("No such feed.\n")
var Body500 = []byte("Something went wrong. Wait a minute, please.\n")
var Body503 = []byte("Taking too long.\n")

func NewFrontend(fs FeedStorage, host string, templateDir string) *Frontend {
	askForURLTemplate := template.Must(template.ParseFiles(templateDir + "/ask_for_url.template.html"))
	feedTemplate := template.Must(template.ParseFiles(templateDir + "/feed.template.xml"))
	feedMetaTemplate := template.Must(template.ParseFiles(templateDir + "/feed_meta.template.html"))
	return &Frontend{host, fs, askForURLTemplate, feedTemplate, feedMetaTemplate}
}

//   GET / -> AskForURL (HEAD, too)
//   GET /u/some_user_id -> UserFeed() (HEAD, too)
//   GET /u_meta/some_user_id -> UserFeedMeta() (HEAD, too)
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

	uMetaMatch := userIdMetaPath.FindAllStringSubmatch(r.URL.Path, -1)
	if len(uMetaMatch) > 0 && len(uMetaMatch[0][1]) > 0 && (r.Method == "GET" || r.Method == "HEAD") {
		r.Form.Add("user_id", uMetaMatch[0][1])
		f.UserFeedMeta(w, r)
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

// Handlers
func (f *Frontend) UserFeedMeta(w http.ResponseWriter, r *http.Request) {
	feed := f.verifyUserOrErrorResponse(w, r)
	if feed == nil {
		return
	}

	feedView := &FeedView{feed, r}
	buf := new(bytes.Buffer)
	err := f.feedMetaTemplate.Execute(buf, feedView)
	if err != nil {
		log.Printf("Error in UserFeedMeta template: %s", err)
		Sigh500(w, r)
		return
	}
	w.Header().Add("Content-Type", `text/html; charset="utf-8"`)
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func (f *Frontend) UserFeed(w http.ResponseWriter, r *http.Request) {
	feed := f.verifyUserOrErrorResponse(w, r)
	if feed == nil {
		return
	}

	feedView := &FeedView{feed, r}
	buf := new(bytes.Buffer)
	err := f.feedTemplate.Execute(buf, feedView)
	if err != nil {
		log.Printf("Error in UserFeed template: %s", err)
		Sigh500(w, r)
		return
	}
	w.Header().Add("Content-Type", `application/atom+xml; charset="utf-8"`)
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func (f *Frontend) verifyUserOrErrorResponse(w http.ResponseWriter, r *http.Request) Feed {
	userId := PlausibleUserId(r.Form.Get("user_id"))
	if userId == "" {
		// TODO: flash[:notice] thing
		// user name seems invalid
		NoSuchFeed(w, r)
		return nil
	}

	feed, err := f.feedStore.Find(userId)

	if err != nil {
		log.Printf("Finding the feed for a user blew up: %s", err)
		Sigh500(w, r)
		return nil
	}

	if feed == nil {
		NoSuchFeed(w, r)
		return nil
	}

	return feed
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

	http.Redirect(w, r, "/u_meta/"+userId, http.StatusFound)

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

func (fv *FeedView) AtomURL() string {
	return "http://" + fv.Req.Host + "/u/" + fv.ActorId()
}

func (fv *FeedView) MetaURL() string {
	return "http://" + fv.Req.Host + "/u_meta/" + fv.ActorId()
}

func (fv *FeedView) Title() string {
	name := fv.Feed.ActorName()
	if name == "" {
		name = "Unknown User"
	}
	return name + " on Google+"
}
