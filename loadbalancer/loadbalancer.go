package loadbalancer

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/FlorinBalint/flo_lb/loadbalancer/algos"
	pb "github.com/FlorinBalint/flo_lb/proto"
)

type lbAlgorithm interface {
	Register(rawURL string) error
	Unregister(url string) error
	Next() *httputil.ReverseProxy
	RegisterCheck(ctx context.Context, chk *algos.Checker)
}

var _ lbAlgorithm = (*algos.RoundRobin)(nil)

type Server struct {
	cfg    *pb.Config
	server *http.Server
	lbAlgo lbAlgorithm
	mu     sync.RWMutex
}

func New(cfg *pb.Config) (*Server, error) {
	if cfg.GetBackend().GetDynamic() != nil {
		// TODO(issues/5): Implement dynamic service discovery
		log.Fatalf("Dynamic service discovery is not yet supported!")
	}

	roundRobin, err := algos.NewRoundRobin(
		cfg.GetBackend().GetStatic().GetUrls(),
	)

	if err != nil {
		return nil, err
	}

	lb := &Server{
		cfg:    cfg,
		lbAlgo: roundRobin,
	}
	lb.server = &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.GetPort()),
		Handler: http.HandlerFunc(lb.ServeHTTP),
	}

	return lb, nil
}

// ServeHTTP is Round Robin handler for loadbalancing
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request for %v\n", r.URL)
	s.lbAlgo.Next().ServeHTTP(w, r)
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
