package algos

import (
	"net/http/httputil"
	"net/url"
	"testing"
)

func TestSetAlive(t *testing.T) {
	url, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Unexpected url error %v", err)
	}
	be := &Backend{
		url:    url,
		status: 0,
	}

	be.SetAlive(true)
	if !be.IsAlive() {
		t.Errorf("be.IsAlive() want true, got false")
	}

	be.SetAlive(false)
	if be.IsAlive() {
		t.Errorf("be.IsAlive() want false, got true")
	}

	be.SetReady(true)
	if be.IsAlive() {
		t.Errorf("be.IsAlive() want false, got true")
	}

	be.SetAlive(true)
	if !be.IsAlive() {
		t.Errorf("be.IsAlive() want true, got false")
	}

	if !be.IsAliveAndReady() {
		t.Errorf("be.IsAliveAndReady() want true, got false")
	}
}

func TestSetReady(t *testing.T) {
	url, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.Fatalf("Unexpected url error %v", err)
	}
	be := &Backend{
		url:    url,
		status: 0,
	}

	if be.IsAliveAndReady() {
		t.Errorf("be.IsAliveAndReady() want false, got true")
	}

	be.SetReady(true)
	if !be.IsReady() {
		t.Errorf("be.IsReady() want true, got false")
	}

	be.SetReady(false)
	if be.IsReady() {
		t.Errorf("be.IsReady() want false, got true")
	}

	be.SetAlive(true)
	if be.IsReady() {
		t.Errorf("be.IsReady() want false, got true")
	}

	be.SetReady(true)
	if !be.IsAlive() {
		t.Errorf("be.IsReady() want true, got false")
	}
	if !be.IsAliveAndReady() {
		t.Errorf("be.IsAliveAndReady() want true, got false")
	}
}

func TestGetOpenConnection(t *testing.T) {
	localUrl, err := url.Parse("http://localhost:0")
	localCon := httputil.NewSingleHostReverseProxy(localUrl)
	if err != nil {
		t.Fatalf("Illegal url error: %v", err)
	}
	tests := []struct {
		name         string
		url          *url.URL
		existing     *httputil.ReverseProxy
		mask         int32
		expectNonNil bool
	}{
		{
			name:         "Unready backend does not open a connection",
			url:          localUrl,
			mask:         aliveMask,
			expectNonNil: false,
		},
		{
			name:         "Unhealthy backend does not open a connection",
			url:          localUrl,
			mask:         readyMask,
			expectNonNil: false,
		},
		{
			name:         "Alive and ready backend opens a connection",
			url:          localUrl,
			mask:         aliveAndReady,
			expectNonNil: true,
		},
		{
			name:         "Alive and ready backend opens a connection",
			url:          localUrl,
			mask:         aliveAndReady,
			expectNonNil: true,
		},

		{
			name:         "Alive and ready backend reuses existing connection",
			url:          localUrl,
			mask:         readyMask | aliveMask,
			existing:     localCon,
			expectNonNil: true,
		},
	}

	for _, test := range tests {
		be := &Backend{
			url:        test.url,
			status:     test.mask,
			connection: test.existing,
		}

		con, ok := be.GetOpenConnection()
		if !ok && test.expectNonNil {
			t.Errorf("backend.GetOpenConnection() want non nil, got nil")
		} else if ok && !test.expectNonNil {
			t.Errorf("backend.GetOpenConnection() want nil, got non-nil")
		}

		if test.expectNonNil && test.existing != nil && con != test.existing {
			t.Errorf("backend.GetOpenConnection() opened new connection, expected to reuse existing")
		}
	}
}
