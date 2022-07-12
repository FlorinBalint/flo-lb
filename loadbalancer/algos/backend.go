package algos

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

const aliveMask int32 = 0x0001
const readyMask int32 = 0x0002
const aliveAndReady int32 = aliveMask | readyMask

type Backend struct {
	rawURL      string
	url         *url.URL
	connections []http.Handler
	status      int32
	mu          sync.RWMutex
}

type UnavailableHandler struct{}

func (UnavailableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte("No available service\n"))
}

func (b *Backend) String() string {
	return fmt.Sprintf("address: %v", b.url)
}

func (b *Backend) URL() string {
	return b.url.String()
}

func NewBackend(rawURL string) (*Backend, error) {
	actualUrl, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	return &Backend{
		rawURL: rawURL,
		status: readyMask, // TODO: Implement readiness checks
		url:    actualUrl,
	}, nil
}

func (b *Backend) andMaskStatus(mask int32) {
	for {
		newStatus := atomic.LoadInt32(&b.status) & mask
		if atomic.CompareAndSwapInt32(&b.status, b.status, newStatus) {
			break
		}
	}
}

func (b *Backend) orMaskStatus(mask int32) {
	for {
		newStatus := atomic.LoadInt32(&b.status) | mask
		if atomic.CompareAndSwapInt32(&b.status, b.status, newStatus) {
			break
		}
	}
}

func (b *Backend) SetAlive(alive bool) {
	if alive {
		b.orMaskStatus(aliveMask)
	} else {
		b.andMaskStatus(^aliveMask)
	}
}

func (b *Backend) SetReady(ready bool) {
	if ready {
		b.orMaskStatus(readyMask)
	} else {
		b.andMaskStatus(^readyMask)
	}
}

func (b *Backend) IsAlive() bool {
	return atomic.LoadInt32(&b.status)&aliveMask > 0
}

func (b *Backend) IsReady() bool {
	return atomic.LoadInt32(&b.status)&readyMask > 0
}

func (b *Backend) IsAliveAndReady() bool {
	return atomic.LoadInt32(&b.status) == aliveAndReady
}

func (b *Backend) ConnectionsCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.connections)
}

func (b *Backend) openConnection() http.Handler {
	b.mu.Lock()
	defer b.mu.Unlock()
	reverseProxy := httputil.NewSingleHostReverseProxy(b.url)
	// TODO(#7): Check when we close a connection
	b.connections = append(b.connections, reverseProxy)
	return reverseProxy
}

func (b *Backend) GetOpenConnection(_ *http.Request) (http.Handler, bool) {
	// TODO(#7): Open connection based on stickiness config.
	b.mu.RLock()
	if b.status != aliveAndReady {
		b.mu.RUnlock()
		return nil, false
	}

	b.mu.RUnlock()
	return b.openConnection(), true
}
