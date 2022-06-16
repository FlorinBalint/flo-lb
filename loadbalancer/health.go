package loadbalancer

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/FlorinBalint/flo_lb/loadbalancer/algos"
)

// pingBackend checks if the backend is alive.
func (s *Server) alive(ctx context.Context, be *algos.Backend) bool {
	healthPath := be.URL() + s.cfg.GetHealthCheck().GetProbe().GetHttpGet().GetPath()
	// TODO Healthcheck: Consider adding extra args to the request.
	req, err := http.NewRequest("GET", healthPath, nil)
	if err != nil {
		log.Printf("Error creating request to %v: %v\n, will consider the backend down", req, err)
		return false
	}
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("%v is unreachable, error: %v", healthPath, err.Error())
		return false
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-OK status: %v", resp.StatusCode)
		return false
	}
	return true
}

func (s *Server) checkHealth(ctx context.Context, be *algos.Backend) {
	msg := "alive"
	alive := s.alive(ctx, be)
	be.SetAlive(alive)
	if !alive {
		msg = "dead"
	}

	// We need to make sure that all threads / routines see the updated value
	log.Printf("%v checked %v by healthcheck", be.URL(), msg)
}

func (s *Server) StartHealthChecks(ctx context.Context) {
	if s.cfg.GetHealthCheck().GetProbe().GetCommand() != nil {
		log.Fatalf("Custom command health probes are not yet supported!")
	}

	initDelay := s.cfg.GetHealthCheck().GetInitialDelay().AsDuration()
	log.Printf("Waiting an initial delay of %v for backends to wake up.", initDelay)
	time.Sleep(initDelay)

	log.Printf("Starting to check the health of backends")
	period := s.cfg.GetHealthCheck().GetPeriod().AsDuration()
	s.lbAlgo.RegisterCheck(
		ctx,
		algos.NewChecker(
			s.checkHealth, period,
		),
	)
}
