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
	url        *url.URL
	connection http.Handler
	status     int32
	mu         sync.RWMutex
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

func (b *Backend) openConnection() http.Handler {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Someone else might have opened a connection since we released the
	// read lock
	if b.connection != nil {
		return b.connection
	}
	// TODO(#1): Add possibility to do TLS handshakes here as well (configurable).
	reverseProxy := httputil.NewSingleHostReverseProxy(b.url)
	b.connection = reverseProxy
	return reverseProxy
}

func (b *Backend) GetOpenConnection() (http.Handler, bool) {
	b.mu.RLock()
	if b.status != aliveAndReady {
		b.mu.RUnlock()
		return nil, false
	}

	// We don't care about 100% status & connection sync,
	// that could change either way until we do a request
	// (or midflight) and we can't do synchronous requests.
	if b.connection != nil {
		b.mu.RUnlock()
		return b.connection, true
	} else {
		b.mu.RUnlock()
		return b.openConnection(), true
	}
}
