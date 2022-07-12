package algos

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
)

func backendWithStatus(t *testing.T, status int32, idx int) *Backend {
	t.Helper()

	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, fmt.Sprintf("%d", idx))
	}
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not open port: %v", err)
	}
	srv := http.Server{
		Addr:    listener.Addr().String(),
		Handler: http.HandlerFunc(handler),
	}
	url, _ := url.Parse(fmt.Sprintf("http://%v", srv.Addr))
	return &Backend{
		url:         url,
		connections: []http.Handler{srv.Handler},
		status:      status,
	}
}

func unreadyBackend(t *testing.T, idx int) *Backend {
	return backendWithStatus(t, aliveMask, idx)
}

func upAndReadyBackend(t *testing.T, idx int) *Backend {
	return backendWithStatus(t, aliveAndReady, idx)
}

func backendWithConnsAndStatus(t *testing.T, conns []http.Handler, status int32) *Backend {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not open port: %v", err)
	}
	srv := http.Server{
		Addr: listener.Addr().String(),
	}
	if len(conns) > 0 {
		srv.Handler = conns[0]
	}
	url, _ := url.Parse(fmt.Sprintf("http://%v", srv.Addr))
	handlers := append(conns[1:], srv.Handler)

	return &Backend{
		url:         url,
		connections: handlers,
		status:      status,
	}
}

func unreadyBackendWithConnections(t *testing.T, conns []http.Handler) *Backend {
	return backendWithConnsAndStatus(t, conns, aliveMask)
}

func readyBackendWithConnections(t *testing.T, conns []http.Handler) *Backend {
	return backendWithConnsAndStatus(t, conns, aliveAndReady)
}
