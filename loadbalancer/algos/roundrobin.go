package algos

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"

	"sync"
)

type RoundRobin struct {
	backends  []*Backend
	beIndices map[string]int
	beCount   int64
	idx       int64
	mu        sync.RWMutex
}

func NewRoundRobin(rawURLs []string) (*RoundRobin, error) {
	backends := make([]*Backend, len(rawURLs))
	beIndices := make(map[string]int)

	for i, rawURL := range rawURLs {
		beIndices[rawURL] = i

		url, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}

		backends[i] = &Backend{
			status: readyMask, // TODO: Implement readiness checks
			url:    url,
		}
	}

	return &RoundRobin{
		idx:       -1,
		backends:  backends,
		beIndices: beIndices,
		beCount:   int64(len(backends)),
	}, nil
}

func (rr *RoundRobin) Register(rawURL string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	url, err := url.Parse(rawURL)
	if _, present := rr.beIndices[rawURL]; present {
		log.Printf("%v already registered", rawURL)
		return nil
	}
	if err != nil {
		return err
	}

	newBackend := &Backend{
		url:    url,
		status: readyMask, // TODO: Implement readiness checks
	}
	if rr.idx >= 0 {
		rr.idx = rr.idx % rr.beCount // make sure the algorithm is fair
	}
	if rr.idx == 0 {
		rr.idx = rr.beCount // pick the new one next if I am at the end
	}
	rr.backends = append(rr.backends, newBackend)
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
	for {
		currIdx := atomic.AddInt64(&rr.idx, 1) % rr.beCount
		if connection, ready := rr.backends[currIdx].GetOpenConnection(); ready {
			return connection
		}
		// TODO: Do some exponential backoff if all backends are dead
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
