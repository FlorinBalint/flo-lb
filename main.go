package main

import (
	"errors"
	"flag"
	"fmt"

	"net/http"

	"github.com/FlorinBalint/flo-lb/addresses"
	"github.com/FlorinBalint/flo-lb/loadbalancer"
)

var (
	backendsFlag = addresses.Flag("backends", nil, "Backends to balance receive workload")
	port         = flag.Int("port", 8080, "Port to listen to")
	name         = flag.String("name", "Flo LB Service", "Name of the service")
)

func main() {
	flag.Parse()

	lbCfg := loadbalancer.Config{
		Port:     *port,
		Backends: *backendsFlag,
	}

	lb := loadbalancer.New(lbCfg, *name)
	err := lb.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Proxy server was closed")
	} else if err != nil {
		fmt.Printf("error starting proxy: %v", err)
	}
}
