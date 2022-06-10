package loadbalancer

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"
)

// pingBackend checks if the backend is alive.
func (s *Server) isAlive(be *backend) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	healthPath := be.url.String() + s.cfg.GetHealthCheck().GetProbe().GetHttpGet().GetPath()
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Get(healthPath)
	if err != nil {
		log.Printf("Unreachable to %v, error: %v", healthPath, err.Error())
		return false
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-OK status: %v", resp.StatusCode)
		return false
	}
	return true
}

func (s *Server) checkHealth(be *backend) {
	go func() {
		msg := "ok"
		val := true
		if !s.isAlive(be) {
			msg = "dead"
			val = false
		}
		alivePtr := (*unsafe.Pointer)(unsafe.Pointer(&be.isAlive))
		// We need to make sure that all threads / routines see the updated value
		atomic.StorePointer(alivePtr, unsafe.Pointer(&val))
		log.Printf("%v checked %v by healthcheck", be.url, msg)
	}()
}

func (s *Server) healthCheck() {
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
		for ; true; <-t.C {
			for _, backend := range s.backends {
				s.checkHealth(backend)
			}
		}
	}()
}
