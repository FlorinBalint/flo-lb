package loadbalancer

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"

	pb "github.com/FlorinBalint/flo_lb"
)

type backend struct {
	url        *url.URL
	connection *httputil.ReverseProxy
	isReady    bool
}

func (b *backend) String() string {
	return fmt.Sprintf("address: %v", b.url)
}

type Server struct {
	cfg      *pb.Config
	backends []*backend
	beCount  int64
	mu       sync.RWMutex
	idx      int64
}

func New(cfg *pb.Config) (*Server, error) {
	if cfg.GetBackend().GetDynamic() != nil {
		// TODO(issues/5): Implement dynamic service discovery
		log.Fatalf("Dynamic service discovery is not yet supported!")
	}

	backends := make([]*backend, len(cfg.GetBackend().GetStatic().GetUrls()))
	for i := 0; i < len(backends); i++ {
		rawURL := cfg.GetBackend().GetStatic().GetUrls()[i]
		url, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}
		backends[i] = &backend{
			url:     url,
			isReady: false, // TODO implemenet readiness / health checks
		}
	}

	return &Server{
		cfg:      cfg,
		backends: backends,
		beCount:  int64(len(backends)),
		idx:      -1,
	}, nil
}

func (s *Server) openNewConnection(idx int64) *backend {
	s.mu.Lock()
	defer s.mu.Unlock()
	reverseProxy := httputil.NewSingleHostReverseProxy(s.backends[idx].url)
	s.backends[idx].connection = reverseProxy
	s.backends[idx].isReady = true
	return s.backends[idx]
}

func (s *Server) next() *backend {
	s.mu.RLock()
	for {
		currIdx := atomic.AddInt64(&s.idx, 1) % s.beCount
		if nextBE := s.backends[currIdx]; nextBE.isReady {
			s.mu.RUnlock()
			return nextBE
		} else if !nextBE.isReady {
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
	fmt.Printf("Starting load balancer with backends %v\n", s.cfg.GetBackend().GetStatic().GetUrls())
	fmt.Printf("%v balancer will start listening on port %v\n", s.cfg.GetName(), s.cfg.GetPort())

	server := http.Server{
		Addr:    fmt.Sprintf(":%v", s.cfg.GetPort()),
		Handler: http.HandlerFunc(s.lbHandler),
	}

	return server.ListenAndServe()
}
