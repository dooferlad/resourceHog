package main

import (
	"fmt"
	"github.com/docker/go-units"
	"github.com/gorilla/handlers"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
}

type Hog struct {
	CPU          int64
	RAM          int64
	Time         time.Duration
	Network      int64
	DiskWrite    int64
	DiskRead     int64
	ResponseSize int64
}

func FromHumanSize(s string) int64 {
	v, err := units.FromHumanSize(s)
	if err != nil {
		panic(err)
	}
	return v
}

func ParseDuration(s string) time.Duration {
	v, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return v
}

func HogFromQuery(query url.Values) *Hog {
	hog := Hog{
		CPU:          FromHumanSize(query.Get("cpu")),
		RAM:          FromHumanSize(query.Get("ram")),
		Time:         ParseDuration(query.Get("time")),
		Network:      FromHumanSize(query.Get("network")),
		DiskWrite:    FromHumanSize(query.Get("disk_write")),
		DiskRead:     FromHumanSize(query.Get("disk_read")),
		ResponseSize: FromHumanSize(query.Get("response_size")),
	}

	return &hog
}

func (h *Hog) Respond(w http.ResponseWriter) {
	wg := sync.WaitGroup{}

	if h.ResponseSize > 0 {

	}
}

func (s *Server) HogHandler(w http.ResponseWriter, r *http.Request) {
	h := HogFromQuery(r.URL.Query())

	fmt.Printf("%#v\n", h)

	h.Respond(w)
}

func New() (*Server, error) {
	logrus.Info("Init...")

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	s := Server{}

	go func() {
		<-c
		s.cleanup()
		os.Exit(1)
	}()

	logrus.Info("Ready")
	return &s, nil
}

// cleanup is called when the process is terminated. It is useful for cleanup
// that is needed when deferred functions wouldn't be called.
func (s *Server) cleanup() {

}

func (s *Server) Serve() {

	m := mux.NewRouter()
	m.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./serve/static"))))
	m.Path("/hog").HandlerFunc(s.HogHandler)

	if err := http.ListenAndServe(":6776", handlers.RecoveryHandler()(m)); err != nil {
		logrus.Fatal(err)
	}
}

func main() {
	if s, err := New(); err != nil {
		logrus.Fatal(err)
	} else {
		s.Serve()
	}
}
