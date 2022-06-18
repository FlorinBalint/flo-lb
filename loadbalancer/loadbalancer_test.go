package loadbalancer

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FlorinBalint/flo_lb/loadbalancer/algos"
	pb "github.com/FlorinBalint/flo_lb/proto"
	"google.golang.org/protobuf/proto"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
)

const healthcheckPeriod = 50 * time.Millisecond

func alwaysAliveBackend() *testBackend {
	testBe := &testBackend{
		requestsReceived: 0,
	}
	tbeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" { // don't count health checks
			atomic.AddInt32(&testBe.requestsReceived, 1)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	testBe.handler = tbeHandler
	return testBe
}

func neverAliveBackend() *testBackend {
	testBe := &testBackend{
		requestsReceived: 0,
	}
	tbeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" { // don't count health checks
			http.Error(w, "I am not alive!", http.StatusInternalServerError)
			return
		}
		atomic.AddInt32(&testBe.requestsReceived, 1)
	})
	testBe.handler = tbeHandler
	return testBe
}

func aliveThenNotBackend() *testBackend {
	testBe := &testBackend{
		requestsReceived: 0,
	}
	tbeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			if atomic.LoadInt32(&testBe.requestsReceived) == 0 {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			} else {
				http.Error(w, "I died!", http.StatusInternalServerError)
			}
			return
		}

		atomic.AddInt32(&testBe.requestsReceived, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	testBe.handler = tbeHandler
	return testBe
}

type testBackend struct {
	handler          http.Handler
	requestsReceived int32
	server           *httptest.Server
}

func (tbe *testBackend) startListen(t *testing.T) {
	t.Helper()
	tbe.server = httptest.NewServer(tbe.handler)
}

func (tbe *testBackend) stop(t *testing.T) {
	t.Helper()
	tbe.server.Close()
}

type testCase struct {
	name         string
	backends     []*testBackend
	wantRequests []int32
}

func (tc *testCase) backendCfg() *pb.BackendConfig {
	beAddresses := make([]string, len(tc.backends))
	for i, be := range tc.backends {
		beAddresses[i] = be.server.URL
	}

	return &pb.BackendConfig{
		Type: &pb.BackendConfig_Static{
			Static: &pb.StaticBackends{
				Urls: beAddresses,
			},
		},
	}
}

func (tc *testCase) newRoundRobinBalancer(t *testing.T) (*Server, error) {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("Failed to find a free LB port")
	}
	cfg := &pb.Config{
		Name:    proto.String("Test LB"),
		Port:    proto.Int32(int32(addr.Port)),
		Backend: tc.backendCfg(),
		HealthCheck: &pb.HealthCheck{
			Probe: &pb.HealthProbe{
				Type: &pb.HealthProbe_HttpGet{
					HttpGet: &pb.HttpGet{
						Path: proto.String("/healthz"),
					},
				},
			},
			Period: &durationpb.Duration{
				Nanos: int32(healthcheckPeriod),
			},
		},
	}
	return New(cfg)
}

func TestLBHandler(t *testing.T) {
	const noRequests int = 3
	tests := []*testCase{
		{
			name:         "one backend receives all requests",
			backends:     []*testBackend{alwaysAliveBackend()},
			wantRequests: []int32{3},
		},
		{
			name:         "two alive backends receive requests",
			backends:     []*testBackend{alwaysAliveBackend(), alwaysAliveBackend()},
			wantRequests: []int32{2, 1},
		},
		{
			name:         "two backends one alive and one dead",
			backends:     []*testBackend{alwaysAliveBackend(), neverAliveBackend()},
			wantRequests: []int32{3, 0},
		},
		{
			name:         "two backends one alive and dies, one always alive",
			backends:     []*testBackend{aliveThenNotBackend(), alwaysAliveBackend()},
			wantRequests: []int32{1, 2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, be := range test.backends {
				go be.startListen(t)
				defer be.stop(t)
			}

			lb, err := test.newRoundRobinBalancer(t)
			defer lb.Close()
			if err != nil {
				t.Errorf("Error creating LB: %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			lb.StartHealthChecks(ctx)

			frontend := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(2 * healthcheckPeriod) // wait for healthchecks
					lb.ServeHTTP(w, r)
				},
				))
			defer frontend.Close()
			requests := make([]*http.Request, noRequests)

			for i := 0; i < noRequests; i++ {
				requests[i], err = http.NewRequest("GET", frontend.URL, nil)
				if err != nil {
					t.Errorf("failed creating request %v:\n %v", i, err)
				}
			}

			for i := 0; i < noRequests; i++ {
				_, err = frontend.Client().Do(requests[i])
				if err != nil {
					t.Errorf("error doing request %v:\n %v", i, err)
				}
			}

			for i, be := range test.backends {
				got := atomic.LoadInt32(&be.requestsReceived)
				if test.wantRequests[i] != got {
					t.Errorf("backend %v got %v requests, want %v", i,
						got, test.wantRequests[i])
				}
			}
		})
	}
}

type fakeLbAlgo struct {
	registerUrl   string
	deregisterUrl string
}

func (fakeAlgo *fakeLbAlgo) Register(rawURL string) error {
	fakeAlgo.registerUrl = rawURL
	return nil
}

func (fakeAlgo *fakeLbAlgo) Deregister(rawURL string) error {
	fakeAlgo.deregisterUrl = rawURL
	return nil
}

func (fakeAlgo *fakeLbAlgo) Handler(r *http.Request) http.Handler {
	return nil
}

func (fakeAlgo *fakeLbAlgo) RegisterCheck(ctx context.Context, chk *algos.Checker) {
}

func fakeBalancer(t *testing.T) (*Server, error) {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("Failed to find a free LB port")
	}

	cfg := &pb.Config{
		Name: proto.String("Fake LB"),
		Port: proto.Int32(int32(addr.Port)),
		Backend: &pb.BackendConfig{
			Type: &pb.BackendConfig_Dynamic{
				Dynamic: &pb.DynamicBackends{
					RegisterPath:   proto.String("/register"),
					DeregisterPath: proto.String("/deregister"),
				},
			},
		},
	}

	if srv, err := New(cfg); err != nil {
		return nil, err
	} else {
		srv.lbAlgo = &fakeLbAlgo{}
		return srv, nil
	}
}

func registerRequest(t *testing.T,
	lbAddr string,
	host string, port *int32) (*http.Request, error) {
	t.Helper()
	bodyReq := &pb.RegisterRequest{
		Host: proto.String(host),
		Port: port,
	}

	if body, err := proto.Marshal(bodyReq); err != nil {
		return nil, err
	} else {
		return http.NewRequest(
			"POST",
			fmt.Sprintf("%v/register", lbAddr), bytes.NewBuffer(body),
		)
	}
}

func deregisterRequest(t *testing.T,
	lbAddr string,
	host string, port *int32) (*http.Request, error) {
	t.Helper()
	bodyReq := &pb.DeregisterRequest{
		Host: proto.String(host),
		Port: port,
	}

	if body, err := proto.Marshal(bodyReq); err != nil {
		return nil, err
	} else {
		return http.NewRequest(
			"POST", fmt.Sprintf("%v/deregister", lbAddr), bytes.NewBuffer(body),
		)
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name             string
		host             string
		port             *int32
		expectedRegister string
	}{
		{
			name:             "Register correct host with port",
			host:             "hostA",
			port:             proto.Int32(8081),
			expectedRegister: "http://hostA:8081",
		},
		{
			name:             "Register correct host without port",
			host:             "hostA",
			expectedRegister: "http://hostA",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lb, err := fakeBalancer(t)
			if err != nil {
				t.Errorf("error creating load balancer %v", err)
			}

			frontend := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lb.server.Handler.ServeHTTP(w, r)
				},
				))
			defer frontend.Close()
			req, err := registerRequest(
				t, frontend.URL,
				test.host, test.port,
			)
			if err != nil {
				t.Errorf("error creating register request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("received unexpected error: %v", err)
			} else if resp.StatusCode != http.StatusOK {
				t.Errorf("unexpected error code, want %v, got %v", resp.StatusCode, http.StatusOK)
			}
			if test.expectedRegister != (lb.lbAlgo).(*fakeLbAlgo).registerUrl {
				t.Errorf("unexpected register, want %v, got %v", test.expectedRegister, (lb.lbAlgo).(*fakeLbAlgo).registerUrl)
			}
		})
	}
}

func TestDeregister(t *testing.T) {
	tests := []struct {
		name               string
		host               string
		port               *int32
		expectedDeregister string
	}{
		{
			name:               "Register correct host with port",
			host:               "hostA",
			port:               proto.Int32(8081),
			expectedDeregister: "http://hostA:8081",
		},
		{
			name:               "Register correct host without port",
			host:               "hostA",
			expectedDeregister: "http://hostA",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lb, err := fakeBalancer(t)
			if err != nil {
				t.Errorf("error creating load balancer %v", err)
			}
			frontend := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lb.server.Handler.ServeHTTP(w, r)
				},
				))
			defer frontend.Close()
			req, err := deregisterRequest(
				t, frontend.URL,
				test.host, test.port,
			)
			if err != nil {
				t.Errorf("error creating deregister request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("received unexpected error: %v", err)
			} else if resp.StatusCode != http.StatusOK {
				t.Errorf("unexpected error code, want %v, got %v", resp.StatusCode, http.StatusOK)
			}
			if test.expectedDeregister != (lb.lbAlgo).(*fakeLbAlgo).deregisterUrl {
				t.Errorf("unexpected register, want %v, got %v", test.expectedDeregister, (lb.lbAlgo).(*fakeLbAlgo).deregisterUrl)
			}
		})
	}
}
