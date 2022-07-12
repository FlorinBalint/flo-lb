package algos

import (
	"net/http"
	"testing"
)

func TestLeastConnsHandler(t *testing.T) {
	readyBE := upAndReadyBackend(t, 0)
	readyBEWith3Cons := readyBackendWithConnections(t, []http.Handler{nil, nil, nil})
	unreadyBe1 := unreadyBackend(t, 1)
	unreadyBe2 := unreadyBackend(t, 2)
	unreadyBeWith3cons := unreadyBackendWithConnections(t, []http.Handler{nil, nil, nil})

	tests := []struct {
		name     string
		backends []*Backend
		wantBE   *Backend
	}{
		{
			name:     "Ready BE is chosen",
			backends: []*Backend{readyBE, unreadyBe1},
			wantBE:   readyBE,
		},
		{
			name:     "Least connections BE is chosen",
			backends: []*Backend{readyBEWith3Cons, readyBE}, // the heap will reoder them
			wantBE:   readyBE,
		},
		{
			name:     "Unready BEs are skipped, returns nil",
			backends: []*Backend{unreadyBe1, unreadyBe2, unreadyBeWith3cons},
			wantBE:   nil,
		},
		{
			name:     "Unready BEs are skipped, choses the ready one",
			backends: []*Backend{unreadyBe1, unreadyBe2, readyBEWith3Cons},
			wantBE:   readyBEWith3Cons,
		},
		{
			name:     "No BEs, returns nil",
			backends: []*Backend{},
			wantBE:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			leastConns, _ := newLeastConnsWithbackends(test.backends)
			// TODO figure out a way to actually test the Handler() method
			be := leastConns.nextBackend(nil)
			if be != test.wantBE {
				t.Errorf("want %v, got %v", test.wantBE, be)
			}
		})
	}
}
