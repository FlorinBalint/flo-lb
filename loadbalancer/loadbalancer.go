package loadbalancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"

	"github.com/FlorinBalint/flo-lb/addresses"
)

type Config struct {
	Port     int
	Backends addresses.Addresses
}

type backend struct {
	addr       string
	connection *httputil.ReverseProxy
	isReady    bool
}

func (b *backend) String() string {
	return fmt.Sprintf("address: %v", b.addr)
}

type Server struct {
	service  string
	cfg      Config
	backends []*backend
	beCount  int64
	mu       sync.RWMutex
	idx      int64
}

func New(cfg Config, name string) *Server {
	return &Server{
		service:  name,
		cfg:      cfg,
		backends: make([]*backend, len(cfg.Backends)),
		beCount:  int64(len(cfg.Backends)),
		idx:      -1,
	}
}

func (s *Server) openNewConnection(idx int64) *backend {
	s.mu.Lock()
	defer s.mu.Unlock()
	url := s.cfg.Backends[idx]
	if s.backends[idx] == nil {
		reverseProxy := httputil.NewSingleHostReverseProxy(url)
		be := &backend{
			addr:       url.String(),
			connection: reverseProxy,
			isReady:    true, // TODO implemenet readiness / health checks
		}
		s.backends[idx] = be
	}
	return s.backends[idx]
}

func (s *Server) next() *backend {
	s.mu.RLock()
	for {
		currIdx := atomic.AddInt64(&s.idx, 1) % s.beCount
		if nextBE := s.backends[currIdx]; nextBE != nil && nextBE.isReady {
			s.mu.RUnlock()
			return nextBE
		} else if nextBE == nil {
			s.mu.RUnlock()
			return s.openNewConnection(currIdx)
		}
	}
}

// lbHandler is Round Robin handler for loadbalancing
func (s *Server) lbHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Received request for %v\n", r.URL)
	be := s.next()
	fmt.Printf("Will forward request to %v\n", be)
	be.connection.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe() error {
	fmt.Printf("Starting load balancer with backends %v\n", s.cfg.Backends)
	fmt.Printf("%v balancer will start listening on port %v\n", s.service, s.cfg.Port)

	server := http.Server{
		Addr:    fmt.Sprintf(":%v", s.cfg.Port),
		Handler: http.HandlerFunc(s.lbHandler),
	}

	return server.ListenAndServe()
}
