package algos

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"sync"

	pb "github.com/FlorinBalint/flo_lb/proto"
)

const maxBackofs = 5

type RoundRobin struct {
	tlsBackends bool
	backends    []*Backend
	beIndices   map[string]int
	beCount     int64
	idx         int64
	backoff     *Backoff
	mu          sync.RWMutex
}

func NewRoundRobin(beCfg *pb.BackendConfig) (*RoundRobin, error) {
	var rawURLs []string
	if beCfg.GetStatic() != nil {
		rawURLs = beCfg.GetStatic().GetUrls()
	}

	backends := make([]*Backend, len(rawURLs))
	beIndices := make(map[string]int)

	var err error
	for i, rawURL := range rawURLs {
		beIndices[rawURL] = i

		if backends[i], err = NewBackend(rawURL); err != nil {
			return nil, err
		}
	}

	return &RoundRobin{
		idx:       -1,
		backends:  backends,
		beIndices: beIndices,
		beCount:   int64(len(backends)),
		// TODO consider making all these (and max backoffs) configurable
		backoff: NewBackoff(
			300*time.Millisecond, // initial sleep
			3*time.Second,        // max sleep
			10*time.Second,       // sleep time reset
			2.0,                  // growth factor
		),
	}, nil
}

func (rr *RoundRobin) Register(rawURL string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if _, present := rr.beIndices[rawURL]; present {
		log.Printf("%v already registered", rawURL)
		return nil
	}

	be, err := NewBackend(rawURL)
	if err != nil {
		return err
	}
	if rr.idx >= 0 {
		rr.idx = rr.idx % rr.beCount // make sure the algorithm is fair
	}
	if rr.idx == 0 {
		rr.idx = rr.beCount // pick the new one next if I am at the end
	}

	rr.backends = append(rr.backends, be)
	rr.beCount++
	rr.beIndices[rawURL] = len(rr.backends)
	return nil
}

func (rr *RoundRobin) Deregister(url string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	beIndex, present := rr.beIndices[url]
	if !present {
		return fmt.Errorf("Tried to remove unknown backend %v", url)
	}

	currIdx := rr.idx % rr.beCount
	if currIdx <= int64(beIndex) && int64(beIndex) != (rr.beCount-1) {
		rr.idx = currIdx % rr.beCount
	} else if int64(beIndex) == (rr.beCount - 1) {
		rr.idx = -1 // start over if we remove last
	} else {
		rr.idx = currIdx - 1 // the current index is after our backend's index
	}
	rr.beCount--
	rr.backends = append(rr.backends[:beIndex], rr.backends[beIndex+1:]...)

	delete(rr.beIndices, url)
	return nil
}

func (rr *RoundRobin) Handler(r *http.Request) http.Handler {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	if rr.beCount == 0 {
		return UnavailableHandler{}
	}

	tries := int64(0)
	backOffs := 0
	for {
		currIdx := atomic.AddInt64(&rr.idx, 1) % rr.beCount
		if connection, ready := rr.backends[currIdx].GetOpenConnection(r); ready {
			return connection
		}
		tries++
		if tries%rr.beCount == 0 {
			if backOffs == maxBackofs {
				return UnavailableHandler{}
			}
			rr.backoff.WaitABit()
			backOffs++
		}
	}
}

func (rr *RoundRobin) RegisterCheck(ctx context.Context, chk *Checker) {
	chk.beSupplier = func() []*Backend {
		rr.mu.RLock()
		defer rr.mu.RUnlock()
		return rr.backends
	}

	chk.runInBackground(ctx)
}
