package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// backend is a reverseProxy peer from ORIGIN_URL to URL
type backend struct {
	URL          *url.URL
	IsAlive      bool
	m            sync.Mutex
	ReverseProxy *httputil.ReverseProxy // reverse proxy used to serve the http through
}

type serverPool struct {
	backends []*backend // list of available backends for this serverPool
	current  uint64     // current pointer to last attempted peer
}

func (s *serverPool) AddBackend(backend *backend) {
	s.backends = append(s.backends, backend)
}

func (s *serverPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

func (s *serverPool) FindBackend() (*backend, error) {
	if len(s.backends) == 0 {
		return nil, errors.New("no backend available")
	}
	return s.backends[0], nil
}

// loadbalance
func (s *serverPool) Loadbalance(w http.ResponseWriter, r *http.Request) {
	peer, err := s.FindBackend()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	peer.ReverseProxy.ServeHTTP(w, r)
}

func main() {
	var serverPool serverPool

	u, err := url.Parse("http://localhost:8080")
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	backend := backend{
		URL:          u,
		IsAlive:      true,
		ReverseProxy: proxy,
	}

	fmt.Println(&backend)

	serverPool.AddBackend(&backend)

	// run all backends
	go runBackend(":8080")

	// run lb server
	server := http.Server{
		Addr:    ":8000",
		Handler: http.HandlerFunc(serverPool.Loadbalance),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}

// run backend server
func runBackend(port string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from the proxy backend in port %s! %s", port, time.Now())
	})

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
