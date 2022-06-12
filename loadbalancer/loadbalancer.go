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

	pb "github.com/FlorinBalint/flo_lb/proto"
)

type backend struct {
	url        *url.URL
	connection *httputil.ReverseProxy
	isAlive    bool
	isReady    bool
	mu         sync.RWMutex
}

func (b *backend) String() string {
	return fmt.Sprintf("address: %v", b.url)
}

func (b *backend) openNewConnection() (*httputil.ReverseProxy, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	reverseProxy := httputil.NewSingleHostReverseProxy(b.url)
	b.connection = reverseProxy
	return reverseProxy, true
}

func (b *backend) getOpenConnection() (*httputil.ReverseProxy, bool) {
	b.mu.RLock()
	readyToServe := b.isAlive && b.isReady
	if b.connection != nil && readyToServe {
		return b.connection, true
	} else if b.connection == nil && readyToServe {
		b.mu.RUnlock()
		return b.openNewConnection()
	}

	return nil, false
}

type Server struct {
	cfg      *pb.Config
	server   *http.Server
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

func (s *Server) pickNext() *httputil.ReverseProxy {
	for {
		currIdx := atomic.AddInt64(&s.idx, 1) % s.beCount
		if connection, ready := s.backends[currIdx].getOpenConnection(); ready {
			return connection
		}
		// TODO: Do some exponential backoff if all backends are dead
	}
}

// ServeHTTP is Round Robin handler for loadbalancing
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request for %v\n", r.URL)
	s.pickNext().ServeHTTP(w, r)
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
