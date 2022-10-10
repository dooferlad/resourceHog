package main

import (
	"context"
	"fmt"
	"github.com/docker/go-units"
	"github.com/gorilla/handlers"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
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
	if s == "" {
		return 0
	}
	v, err := units.FromHumanSize(s)
	if err != nil {
		panic(err)
	}
	return v
}

func ParseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return v
}

func WriteFileOfSize(name string, size int64) error {
	buf := make([]byte, 1024*1024)

	if f, err := os.Create(name); err != nil {
		return err
	} else {
		for bytesRemaining := size; bytesRemaining > 0; {
			// Update size if needed to not over-write
			if len(buf) > int(bytesRemaining) {
				buf = buf[:bytesRemaining]
			}

			if _, err = f.Write(buf); err != nil {
				return err
			}

			bytesRemaining -= int64(len(buf))
		}

		if err = f.Close(); err != nil {
			return err
		}
	}

	return nil
}

func ReadFileOfSize(name string, size int64) error {
	buf := make([]byte, 1024*1024)

	f, err := os.Open(name)
	defer f.Close()
	if err != nil {
		return err
	}

	for bytesRemaining := size; bytesRemaining > 0; {
		// Update size if needed to not over-read
		if len(buf) > int(bytesRemaining) {
			buf = buf[:bytesRemaining]
		}

		readBytes, err := f.Read(buf)

		if err != nil {
			return err
		}

		bytesRemaining -= int64(readBytes)
	}

	return nil
}

func CPUHog(ctx context.Context, rc chan uint64) {
	// Generate random numbers. https://en.wikipedia.org/wiki/Linear_congruential_generator
	s := uint64(123456)
	a := uint64(25214903917)
	c := uint64(11)
	m := uint64(1) << 48

	for {
		// Need to generate a lot of random numbers before letting a chance of a thread/goroutine switch
		// to really tie up a CPU, so only let a switch happen on a reasonably long timer
		sendTime := time.Now().Add(time.Millisecond * 10)
		for {
			if time.Now().After(sendTime) {
				break
			}
		}

		s = (a*s + c) % m

		select {
		case rc <- s:
			// We send numbers somewhere so the compiler doesn't optimise away our RNG.
		case <-ctx.Done():
			logrus.Info("... Terminating CPU hog")
			return
		}

		select {
		case <-rc:
		}
	}
}

func HogFromQuery(query url.Values) *Hog {
	hog := Hog{
		CPU:          FromHumanSize(query.Get("cpu")),
		RAM:          FromHumanSize(query.Get("ram")),
		Time:         ParseDuration(query.Get("time")),
		DiskWrite:    FromHumanSize(query.Get("disk_write")),
		DiskRead:     FromHumanSize(query.Get("disk_read")),
		ResponseSize: FromHumanSize(query.Get("response_size")),
	}

	return &hog
}

func (h *Hog) Respond(w http.ResponseWriter) {
	wg := sync.WaitGroup{}

	if h.ResponseSize > 0 {
		wg.Add(1)

		go func() {
			b := []byte{0}

			for remaining := h.ResponseSize; remaining > 0; remaining-- {
				_, _ = w.Write(b)
			}

			wg.Done()
		}()
	}

	if h.Time > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), h.Time)
		rc := make(chan uint64, 100)
		defer cancel() // to avoid resource leeks - normally we call cancel inside a specific hog termination loop

		if h.CPU > 0 {
			logrus.Infof("Using %d CPUs for %v seconds", h.CPU, h.Time)
			for i := 0; i < int(h.CPU); i++ {
				logrus.Info("Creating CPU hog")
				go CPUHog(ctx, rc)
			}
		}

		if h.RAM > 0 {
			logrus.Infof("Using %d RAM for %v seconds", h.RAM, h.Time)

			// Putting this in a function gives the compiler the bright idea to compile it away, which we don't
			// want, so don't be tempted. If this stops working, turn to cgo :-(
			x := make([]byte, h.RAM)
			for i := int64(0); i < h.RAM; i++ {
				x[i] = byte(i)
				if x[i] != byte(i) {
					panic("what the heck!")
				}
			}
		}

		<-ctx.Done()
	}

	if h.DiskRead > 0 {
		wg.Add(1)
		go func() {
			const fileName = "resourceHogReadFile"
			err := WriteFileOfSize(fileName, h.DiskRead)
			if err != nil {
				panic(err)
			}
			err = ReadFileOfSize(fileName, h.DiskRead)
			if err != nil {
				panic(err)
			}
			wg.Done()
		}()
	}

	if h.DiskWrite > 0 {
		wg.Add(1)
		go func() {
			err := WriteFileOfSize("resourceHogWriteFile", h.DiskWrite)
			if err != nil {
				panic(err)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	logrus.Info("Done responding")
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

	// When using the memory hog we can allocate a lot of memory that isn't quickly cleaned up unless there is pressure
	// to do so. This sets a soft memory limit for the GC of 100MiB, which causes it to rapidly reclaim memory. The
	// call requires Gi 1.9+
	debug.SetMemoryLimit(100 * 1024 * 1024)

	logrus.Info("Ready")
	return &s, nil
}

// cleanup is called when the process is terminated. It is useful for cleanup
// that is needed when deferred functions wouldn't be called.
func (s *Server) cleanup() {

}

func (s *Server) Serve() {
	m := mux.NewRouter()
	m.Path("/").HandlerFunc(s.HogHandler)

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
