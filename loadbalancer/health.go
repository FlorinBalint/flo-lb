package loadbalancer

import (
	"context"
	"log"
	"net/http"
	"time"
)

// pingBackend checks if the backend is alive.
func (s *Server) liveness(ctx context.Context, aliveCh chan bool, be *backend) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	healthPath := be.url.String() + s.cfg.GetHealthCheck().GetProbe().GetHttpGet().GetPath()
	// TODO Healthcheck: Consider adding extra args to the request.
	req, err := http.NewRequest("GET", healthPath, nil)
	if err != nil {
		aliveCh <- false
		log.Printf("Error creating request to %v: %v", req, err)
	}
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("%v is unreachable, error: %v", healthPath, err.Error())
		aliveCh <- false
		return
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-OK status: %v", resp.StatusCode)
		aliveCh <- false
		return
	}
	aliveCh <- true
}

func (s *Server) checkHealth(ctx context.Context, be *backend) {
	go func() {
		msg := "alive"
		aliveCh := make(chan bool)
		go s.liveness(ctx, aliveCh, be)
		select {
		case <-ctx.Done():
			log.Printf("context canceled")
			break
		case be.isAlive = <-aliveCh:
			if !be.isAlive {
				msg = "dead"
			}
		}

		// We need to make sure that all threads / routines see the updated value
		log.Printf("%v checked %v by healthcheck", be.url, msg)
	}()
}

func (s *Server) StartHealthChecks(ctx context.Context) {
	if s.cfg.GetHealthCheck().GetProbe().GetCommand() != nil {
		log.Fatalf("Custom command health probes are not yet supported!")
	}

	initDelay := s.cfg.GetHealthCheck().GetInitialDelay().AsDuration()
	log.Printf("Waiting an initial delay of %v for backends to wake up.", initDelay)
	time.Sleep(initDelay)

	log.Printf("Starting to check the health of backends")
	// Now do requests asynchronously
	go func() {
		t := time.NewTicker(s.cfg.GetHealthCheck().GetPeriod().AsDuration())
		defer t.Stop()
		for true {
			select {
			case <-ctx.Done():
				break
			case <-t.C:
				for _, backend := range s.backends {
					s.checkHealth(ctx, backend)
				}
			}
		}
	}()
}
