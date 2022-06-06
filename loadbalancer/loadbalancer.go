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
	beCount  uint64
	mu       sync.RWMutex
	idx      uint64
}

func New(cfg Config, name string) *Server {
	return &Server{
		service:  name,
		cfg:      cfg,
		backends: make([]*backend, len(cfg.Backends)),
		beCount:  uint64(len(cfg.Backends)),
		idx:      0,
	}
}

func (s *Server) next() (*backend, error) {
	s.mu.RLock()
	var BE *backend
	var err error

	for BE == nil {
		currIdx := atomic.LoadUint64(&s.idx) % s.beCount
		atomic.AddUint64(&s.idx, 1)
		if nextBE := s.backends[currIdx]; nextBE != nil && nextBE.isReady {
			BE = nextBE
		} else if nextBE == nil {
			targetURL, urlErr := url.Parse(s.cfg.Backends[currIdx])
			if urlErr != nil {
				err = urlErr
				s.mu.RLock()
				break
			}
			reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

			s.mu.RUnlock()
			BE = &backend{
				addr:       targetURL.String(),
				connection: reverseProxy,
				isReady:    true, // TODO implemenet readiness / health checks
			}

			s.mu.Lock()
			s.backends[currIdx] = BE
			s.mu.Unlock()
		}
	}

	return BE, err
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
