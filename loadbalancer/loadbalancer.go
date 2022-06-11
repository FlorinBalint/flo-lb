package loadbalancer

import (
	"context"
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
	isAlive    bool
	isReady    bool
}

func (b *backend) String() string {
	return fmt.Sprintf("address: %v", b.url)
}

type Server struct {
	cfg      *pb.Config
	server   *http.Server
	backends []*backend
	beCount  int64
	// TODO: Consider improving the granularity of the lock
	mu  sync.RWMutex
	idx int64
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
			isReady: true, // TODO implemenet readiness / health checks
			isAlive: false,
		}
	}

	lb := &Server{
		cfg:      cfg,
		backends: backends,
		beCount:  int64(len(backends)),
		idx:      -1,
	}
	lb.server = &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.GetPort()),
		Handler: http.HandlerFunc(lb.ServeHTTP),
	}

	return lb, nil
}

func (s *Server) openNewConnection(idx int64) *backend {
	s.mu.Lock()
	defer s.mu.Unlock()
	reverseProxy := httputil.NewSingleHostReverseProxy(s.backends[idx].url)
	s.backends[idx].connection = reverseProxy
	s.backends[idx].isReady = true // TODO implement readiness checks
	return s.backends[idx]
}

func (s *Server) next() *backend {
	s.mu.RLock()
	for {
		currIdx := atomic.AddInt64(&s.idx, 1) % s.beCount
		if nextBE := s.backends[currIdx]; nextBE.connection != nil && nextBE.isAlive {
			s.mu.RUnlock()
			return nextBE
		} else if nextBE.connection == nil && nextBE.isAlive && nextBE.isReady { // try to open a new connection
			s.mu.RUnlock()
			return s.openNewConnection(currIdx)
		} // else check next backend, this one is not alive
		// TODO: Do some exponential backoff if all backends are dead
	}
}

// ServeHTTP is Round Robin handler for loadbalancing
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request for %v\n", r.URL)
	be := s.next()
	be.connection.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	lbContext, cancel := context.WithCancel(ctx)
	defer cancel()

	if s.cfg.GetHealthCheck() != nil {
		s.StartHealthChecks(lbContext)
	}

	log.Printf("Starting load balancer with backends %v\n", s.cfg.GetBackend().GetStatic().GetUrls())
	log.Printf("%v balancer will start listening on port %v\n", s.cfg.GetName(), s.cfg.GetPort())
	return s.server.ListenAndServe()
}

func (s *Server) Close() error {
	return s.server.Close()
}
