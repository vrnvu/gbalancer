package loadbalancer

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Backend is a reverseProxy peer from ORIGIN_URL to URL
type Backend struct {
	URL          *url.URL
	Alive        bool
	m            sync.Mutex
	ReverseProxy *httputil.ReverseProxy // reverse proxy used to serve the http through
}

// ServerPool holds a list of backend peers
type ServerPool struct {
	Backends []*Backend // list of available Backends for this ServerPool
	current  uint64     // current pointer to last attempted peer, used for balancing algorithms
}

// AddBackend adds a Backend reference into the pool of peers
func (s *ServerPool) AddBackend(Backend *Backend) {
	s.Backends = append(s.Backends, Backend)
}

func (s *ServerPool) nextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.Backends)))
}

func (b *Backend) enable() {
	b.m.Lock()
	defer b.m.Unlock()
	b.Alive = true
}

func (b *Backend) disable() {
	b.m.Lock()
	defer b.m.Unlock()
	b.Alive = false
}

func (b *Backend) isAlive() bool {
	b.m.Lock()
	defer b.m.Unlock()
	return b.Alive
}

// checkHealth returns the state of a backend
// for now it returns true or false according if its up or down
// todo chechHealth return a enum with DOWN, UP, BAD
// so we can implement smarter algorithms for balancing
func (b *Backend) checkHealth(timeoutSeconds int) bool {
	timeout := time.Duration(timeoutSeconds) * time.Second
	conn, err := net.DialTimeout("tcp", b.URL.Host, timeout)
	if err != nil {
		log.Println("can not establish a connection with backend " + b.URL.Host)
		return false
	}
	if conn.Close(); err != nil {
		log.Println("error closing tcp connection with backend " + b.URL.Host)
	}
	return true
}

// HealthCheck routine pings the backends pool every interval and updates backends
func (s *ServerPool) HealthCheck(seconds, timeout int) {
	t := time.NewTicker(time.Duration(seconds) * time.Second)
	for {
		select {
		case <-t.C:
			for _, b := range s.Backends {
				// todo chechHealth return a enum with DOWN, UP, BAD
				status := b.checkHealth(timeout)
				switch status {
				case true:
					b.enable()
					log.Printf("HealthCheck %s [%t]\n", b.URL, status)
				case false:
					b.disable()
					log.Printf("HealthCheck %s [%t]\n", b.URL, status)
				}
			}

		}
	}
}

// FindBackend attempts to find a valid backend from the list of peers
// An error is return if an exception happens
func (s *ServerPool) FindBackend() (*Backend, error) {
	if len(s.Backends) == 0 {
		return nil, errors.New("no Backend available")
	}
	return s.findRoundRobin()
}

func (s *ServerPool) findRoundRobin() (*Backend, error) {
	next := s.nextIndex()
	l := len(s.Backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.Backends)
		if s.Backends[idx].isAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.Backends[idx], nil
		}
	}
	return nil, errors.New("round roubin failed to get a valid Backend")
}

// Loadbalance runs the balancing algorithm for each requeest
func (s *ServerPool) Loadbalance(w http.ResponseWriter, r *http.Request) {
	// FindBackend is for now a wrapper over our algorithm
	peer, err := s.FindBackend()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	peer.ReverseProxy.ServeHTTP(w, r)
}
