package main

import (
	"github.com/rcrowley/go-metrics"
	"text/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const (
	vars = `{{range .}}{{.Name}} {{.Value}}
{{end}}`
)

var (
	index = []byte(`<!DOCTYPE html>
<html>
  <a href="/vars">/vars</a>
</html>
`)
)

var varsTmpl = template.Must(template.New("vars").Parse(vars))

func NewStatServer(addr string) *http.Server {
	d := time.Duration(400 * time.Millisecond)
	m := http.NewServeMux()
	m.Handle("/vars", &StatHandler{registry})
	m.Handle("/", &ControlIndexHandler{})
	return &http.Server{addr, m, d, d, 0, nil}
}

type ControlIndexHandler struct {}

func (c *ControlIndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(index)
}

type Stat struct {
	Name string
	Value string
}

type statSlice []Stat

func (s statSlice) Len() int {
	return len(s)
}

func (s statSlice) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s statSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type statHolder struct {
	Stats []Stat 
}

type StatHandler struct {
	reg metrics.Registry
}

func (s *StatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stats := []Stat{
		Stat{"boot_time_utc", bootTime.String()},
		Stat{"boot_time_epoch_nanos", strconv.FormatInt(bootTime.UnixNano(), 10)},
	}
	s.reg.Each(func(s string, obj interface{}) {
		switch v := obj.(type) {
		case metrics.Counter:
			stats = append(stats, Stat{s, strconv.FormatInt(v.Count(), 10)})
		}
	})
	sort.Sort(statSlice(stats))
	w.Header().Add("Content-Type", "plain/text")
	w.WriteHeader(http.StatusOK)
	err := varsTmpl.Execute(w, stats)
	if err != nil {
		log.Printf("ERROR unable to execute /vars template: %v", err)
	}
}
