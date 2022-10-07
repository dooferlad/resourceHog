package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
}

type Hog struct {
	CPU          string
	RAM          string
	Time         string
	Network      string
	DiskWrite    string
	DiskRead     string
	ResponseSize string
}

func HogFromQuery(query url.Values) *Hog {
	hog := Hog{
		CPU:          query.Get("cpu"),
		RAM:          query.Get("ram"),
		Time:         query.Get("time"),
		Network:      query.Get("network"),
		DiskWrite:    query.Get("disk_write"),
		DiskRead:     query.Get("disk_read"),
		ResponseSize: query.Get("response_size"),
	}

	return &hog
}

func (s *Server) HogHandler(w http.ResponseWriter, r *http.Request) {
	h := HogFromQuery(r.URL.Query())

	fmt.Printf("%#v\n", h)
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

	if err := http.ListenAndServe(":6776", m); err != nil {
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
