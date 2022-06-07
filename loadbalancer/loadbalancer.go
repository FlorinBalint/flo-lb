package loadbalancer

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
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

func (s *Server) openNewConnectionForIndex(idx int64, url *url.URL) *backend {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *Server) next() (*backend, error) {
	var be *backend
	var err error
	// We cannot defer RUnlock because we could call openNewConnectionForIndex()
	s.mu.RLock()
	for be == nil {
		currIdx := atomic.AddInt64(&s.idx, 1) % s.beCount
		if nextBE := s.backends[currIdx]; nextBE != nil && nextBE.isReady {
			be = nextBE
			s.mu.RUnlock()
		} else if nextBE == nil {
			targetURL, urlErr := url.Parse(s.cfg.Backends[currIdx])
			if urlErr != nil {
				err = urlErr
				s.mu.RUnlock()
				break
			}
			s.mu.RUnlock()
			be = s.openNewConnectionForIndex(currIdx, targetURL)
		}
	}

	return be, err
}

// lbHandler is Round Robin handler for loadbalancing
func (s *Server) lbHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Received request for %v\n", r.URL)
	be, err := s.next()
	if err != nil {
		log.Printf("Error while trying to reach a backend: %v", err)
	} else {
		fmt.Printf("Will forward request to %v\n", be)
		be.connection.ServeHTTP(w, r)
	}
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
