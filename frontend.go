package main

import (
	"bytes"
	"code.google.com/p/google-api-go-client/googleapi"
	"github.com/bmizerany/pat"
	html "html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	text "text/template"
)

var (
	Body404 = []byte("No such feed.\n")
	Body500 = []byte("Something went wrong. Wait a minute, please.\n")
	Body503 = []byte("Taking too long.\n")

	userIdPath     = regexp.MustCompile(`/u/(\d+)$`)
	userIdMetaPath = regexp.MustCompile(`/u_meta/(\d+)$`)
	justUserIdR    = regexp.MustCompile(`^\d+$`)
	plusUrlR       = regexp.MustCompile(`^https?://plus.google.com/(\d+)/?`)
)

type Frontend struct {
	host              string
	feedStore         FeedStorage
	askForURLTemplate *html.Template
	feedMetaTemplate  *html.Template
	feedTemplate      *text.Template
}

//   GET / -> AskForURL (HEAD, too)
//   GET /u/some_user_id -> UserFeed() (HEAD, too)
//   GET /u_meta/some_user_id -> UserFeedMeta() (HEAD, too)
//   POST /plus/enqueue -> CheckURLOrUserId
func NewFrontendMux(fs FeedStorage, host string, templateDir string) http.Handler {
	askForURLTemplate := html.Must(html.ParseFiles(templateDir + "/ask_for_url.template.html"))
	feedMetaTemplate := html.Must(html.ParseFiles(templateDir + "/feed_meta.template.html"))
	feedTemplate := text.Must(text.ParseFiles(templateDir + "/feed.template.xml"))
	host = strings.TrimRight(host, "/")
	f := &Frontend{host, fs, askForURLTemplate, feedMetaTemplate, feedTemplate}
	m := pat.New()

	askForURL := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.AskForURL(w, r)
	})
	m.Get("/", askForURL)
	m.Head("/", askForURL)

	userFeed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.UserFeed(w, r)
	})
	m.Get("/u/:user_id", userFeed)
	m.Head("/u/:user_id", userFeed)

	userFeedMeta := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.UserFeedMeta(w, r)
	})
	m.Get("/u_meta/:user_id", userFeedMeta)
	m.Head("/u_meta/:user_id", userFeedMeta)

	m.Post("/plus/enqueue", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.CheckURLOrUserId(w, r)
	}))

	hf := func(w http.ResponseWriter, r *http.Request) {
		if r.Host != f.host {
			log.Printf("Request asked for %s, expected %s", r.Host, f.host)
			http.Redirect(w, r, "http://"+f.host, 302)
			return
		}
		m.ServeHTTP(w, r)
	}

	return http.HandlerFunc(hf)
}

func (f *Frontend) UserFeedMeta(w http.ResponseWriter, r *http.Request) {
	feed := f.verifyUserOrErrorResponse(w, r)
	if feed == nil {
		return
	}

	feedView := &FeedView{feed, r}
	buf := new(bytes.Buffer)
	err := f.feedMetaTemplate.Execute(buf, feedView)
	if err != nil {
		log.Printf("ERROR UserFeedMeta template execute: %s", err)
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

	var err error
	feedExecuteTiming.Time(func() {
		err = f.feedTemplate.Execute(buf, feedView)
	})
	if err != nil {
		log.Printf("ERROR UserFeed template execute: %s", err)
		Sigh500(w, r)
		return
	}
	w.Header().Add("Content-Type", `application/atom+xml; charset="utf-8"`)
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func (f *Frontend) verifyUserOrErrorResponse(w http.ResponseWriter, r *http.Request) Feed {
	userId := PlausibleUserId(r.FormValue(":user_id"))
	if userId == "" {
		// TODO: flash[:notice] thing
		// user name seems invalid
		NoSuchFeed(w, r)
		return nil
	}

	feed, err := f.feedStore.Find(userId)

	if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
		NoSuchFeed(w, r)
		return nil
	} else if err != nil {
		log.Printf("ERROR Finding the feed for a user blew up: %#v", err)
		Sigh500(w, r)
		return nil
	}

	return feed
}

func (f *Frontend) AskForURL(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := f.askForURLTemplate.Execute(w, nil)
	if err != nil {
		log.Printf("ERROR AskForUrl template execute: %s", err)
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

// FeedView is a struct that wraps a Feed and an HTTP request to simplify the
// Atom XML template.
type FeedView struct {
	Feed
	Req *http.Request
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
