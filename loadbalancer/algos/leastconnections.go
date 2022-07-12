package algos

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	pb "github.com/FlorinBalint/flo_lb/proto"
)

type beComparator struct{}

// We need a min heap, so we use the reverse order
func (beComparator) Less(a, b *Backend) bool {
	return b.ConnectionsCount() < a.ConnectionsCount()
}

type LeastConnections struct {
	backends *AdressablePQ[string, *Backend]
	backoff  *Backoff
	mu       sync.RWMutex
}

func newLeastConnsWithbackends(backends []*Backend) (*LeastConnections, error) {
	bePQ := NewPQWithComparator[string, *Backend](beComparator{})
	for _, be := range backends {
		// Note: This normally takes O(n) in our case because
		// each element keeps its position (no need for a smart BuildHeap).
		bePQ.Push(be.URL(), be)
	}

	return &LeastConnections{
		backends: bePQ,
		// TODO consider making all these (and max backoffs) configurable
		backoff: NewBackoff(
			300*time.Millisecond, // initial sleep
			3*time.Second,        // max sleep
			10*time.Second,       // sleep time reset
			2.0,                  // growth factor
		),
	}, nil
}

func NewLeastConnections(beCfg *pb.BackendConfig) (*LeastConnections, error) {
	var rawURLs []string
	if beCfg.GetStatic() != nil {
		rawURLs = beCfg.GetStatic().GetUrls()
	}

	var backends []*Backend
	for _, rawURL := range rawURLs {
		if backend, err := NewBackend(rawURL); err != nil {
			return nil, err
		} else {
			backends = append(backends, backend)
		}
	}
	return newLeastConnsWithbackends(backends)
}

func (lConn *LeastConnections) Register(rawURL string) error {
	newBe, err := NewBackend(rawURL)
	if err != nil {
		return err
	}
	lConn.mu.Lock()
	defer lConn.mu.Unlock()
	if !lConn.backends.Push(rawURL, newBe) {
		log.Printf("%v already registered", rawURL)
	}
	return nil
}

func (lConn *LeastConnections) Deregister(rawURL string) error {
	lConn.mu.Lock()
	defer lConn.mu.Unlock()
	if !lConn.backends.Remove(rawURL) {
		return fmt.Errorf("Tried to remove unknown backend %v", rawURL)
	}
	return nil
}

func (lConn *LeastConnections) bestAliveBE(r *http.Request) *Backend {
	lConn.mu.Lock()
	defer lConn.mu.Unlock()
	var popped []*Backend
	defer func() {
		// Push popped Backends back
		for _, tried := range popped {
			lConn.backends.Push(tried.URL(), tried)
		}
	}()

	for !lConn.backends.Empty() {
		minConnsBE := lConn.backends.Pop()
		popped = append(popped, minConnsBE)
		if minConnsBE.IsAliveAndReady() {
			return minConnsBE
		}
	}
	return nil
}

func (lConn *LeastConnections) nextBackend(r *http.Request) *Backend {
	if lConn.backends.Empty() {
		return nil
	}

	lConn.mu.RLock()
	// Try top optimistically
	if !lConn.backends.Empty() && lConn.backends.Top().IsAliveAndReady() {
		minConnsBE := lConn.backends.Top()
		lConn.mu.RUnlock()
		return minConnsBE
	}
	lConn.mu.RUnlock() // cannot defer, because bestAlive needs the write lock
	return lConn.bestAliveBE(r)
}

func (lConn *LeastConnections) Handler(r *http.Request) http.Handler {
	minConnsBE := lConn.nextBackend(r)
	if minConnsBE == nil {
		return UnavailableHandler{}
	}
	lConn.mu.Lock()
	defer lConn.mu.Unlock()
	res, _ := minConnsBE.GetOpenConnection(r)
	lConn.backends.Emplace(minConnsBE.URL(), minConnsBE)
	return res
}

func (lConn *LeastConnections) RegisterCheck(ctx context.Context, chk *Checker) {
	chk.beSupplier = func() []*Backend {
		lConn.mu.RLock()
		defer lConn.mu.RUnlock()
		return lConn.backends.Values()
	}

	chk.runInBackground(ctx)
}
