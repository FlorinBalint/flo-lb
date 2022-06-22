package loadbalancer

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/FlorinBalint/flo_lb/loadbalancer/algos"
	pb "github.com/FlorinBalint/flo_lb/proto"
	"google.golang.org/protobuf/proto"
)

type lbAlgorithm interface {
	Register(rawURL string) error
	Deregister(url string) error
	Handler(r *http.Request) http.Handler
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
	var roundRobin lbAlgorithm
	var err error
	mux := http.NewServeMux()
	lb := &Server{
		cfg: cfg,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%v", cfg.GetPort()),
			Handler: mux,
		},
	}

	if cfg.GetBackend().GetDynamic() != nil {
		roundRobin, err = algos.NewRoundRobin(nil)
		if err != nil {
			return nil, err
		}
		mux.Handle("/", http.HandlerFunc(lb.ServeHTTP))
		mux.Handle("/healthz", http.HandlerFunc(lb.Health))
		mux.Handle(cfg.Backend.GetDynamic().GetRegisterPath(), http.HandlerFunc(lb.RegisterNew))
		mux.Handle(cfg.Backend.GetDynamic().GetDeregisterPath(), http.HandlerFunc(lb.Deregister))
	} else if cfg.GetBackend().GetStatic() != nil {
		roundRobin, err = algos.NewRoundRobin(
			cfg.GetBackend().GetStatic().GetUrls(),
		)
		if err != nil {
			return nil, err
		}
		mux.Handle("/", http.HandlerFunc(lb.ServeHTTP))
		mux.Handle("/healthz", http.HandlerFunc(lb.Health))
	}

	lb.lbAlgo = roundRobin

	return lb, nil
}

func (s *Server) RegisterNew(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received register request")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error reading request"))
		return
	}
	regReq := &pb.RegisterRequest{}
	if err := proto.Unmarshal(body, regReq); err != nil {
		log.Printf("Failed to parse register request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error reading request"))
		return
	} else if len(regReq.GetHost()) == 0 {
		log.Printf("Received register request without host: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request must have host set"))
		return
	}

	var rawUrl string
	if regReq.Port != nil {
		rawUrl = fmt.Sprintf("http://%v:%v", regReq.GetHost(), regReq.GetPort())
	} else {
		rawUrl = fmt.Sprintf("http://%v", regReq.GetHost())
	}

	if err := s.lbAlgo.Register(rawUrl); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error handling register"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Registered"))
	}
}

func (s *Server) Deregister(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received deregister request")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error reading request"))
		return
	}
	regReq := &pb.DeregisterRequest{}
	if err := proto.Unmarshal(body, regReq); err != nil {
		log.Printf("Failed to parse unregister request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error reading request"))
		return
	} else if len(regReq.GetHost()) == 0 {
		log.Printf("Received unregister request without host: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request must have host set"))
		return
	}

	var rawUrl string
	if regReq.Port != nil {
		rawUrl = fmt.Sprintf("http://%v:%v", regReq.GetHost(), regReq.GetPort())
	} else {
		rawUrl = fmt.Sprintf("http://%v", regReq.GetHost())
	}

	if err := s.lbAlgo.Deregister(rawUrl); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error handling deregister"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Registered"))
	}
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	log.Printf("got /healthz request\n")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("I am alive"))
}

// ServeHTTP is Round Robin handler for loadbalancing
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request for %v\n", r.URL)
	s.lbAlgo.Handler(r).ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	lbContext, cancel := context.WithCancel(ctx)
	defer cancel()

	// TODO(#1): Validate that we are using https protocol
	if s.cfg.GetCert() != nil {
		err := s.SetupTLS(ctx)
		if err != nil {
			return fmt.Errorf("error loading certs %v", err)
		}
	}

	if s.cfg.GetHealthCheck() != nil {
		s.StartHealthChecks(lbContext)
	}

	log.Printf("Starting load balancer with backends %v\n", s.cfg.GetBackend().GetStatic().GetUrls())
	log.Printf("%v balancer will start listening on port %v\n", s.cfg.GetName(), s.cfg.GetPort())
	if s.cfg.GetProtocol() == pb.Protocol_HTTPS {
		return s.server.ListenAndServeTLS("", "")
	}
	return s.server.ListenAndServe()
}

func (s *Server) Close() error {
	return s.server.Close()
}
