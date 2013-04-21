package main

import (
	"github.com/rcrowley/go-metrics"
	"log"
	"net/http"
	"sort"
	"strconv"
	"text/template"
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
	m.Handle("/", http.HandlerFunc(ControlIndexHandler))
	return &http.Server{Addr: addr, Handler: m, ReadTimeout: d, WriteTimeout: d,}
}

func ControlIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(index)
}

type Stat struct {
	Name  string
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
		case metrics.Timer:
			fifteen := strconv.FormatFloat(v.Rate15(), 'g', -1, 64)
			stats = append(stats, Stat{s + "_fifteen_minute_rate", fifteen})
			five := strconv.FormatFloat(v.Rate5(), 'g', -1, 64)
			stats = append(stats, Stat{s + "_five_minute_rate", five})
			stats = append(stats, Stat{s + "_one_minute_rate", strconv.FormatFloat(v.Rate1(), 'g', -1, 64)})

			// Since Percentile returns integer nanos, its values are easier
			// to read when treated as such.
			p50 := strconv.FormatInt(int64(v.Percentile(0.50)), 10)
			stats = append(stats, Stat{s + "_p50", p50})
			p99 := strconv.FormatInt(int64(v.Percentile(0.99)), 10)
			stats = append(stats, Stat{s + "_p99", p99})
			p999 := strconv.FormatInt(int64(v.Percentile(0.999)), 10)
			stats = append(stats, Stat{s + "_p999", p999})
			p9999 := strconv.FormatInt(int64(v.Percentile(0.9999)), 10)
			stats = append(stats, Stat{s + "_p9999", p9999})
		default:
			// TODO(jmhodges): Gauges, Meters, Histograms, Samples
		}
	})
	sort.Sort(statSlice(stats))
	w.Header().Add("Content-Type", "plain/text; charset=utf8")
	w.WriteHeader(http.StatusOK)
	err := varsTmpl.Execute(w, stats)
	if err != nil {
		log.Printf("ERROR unable to execute /vars template: %v", err)
	}
}
