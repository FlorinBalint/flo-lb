package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/FlorinBalint/flo_lb/loadbalancer"
	"github.com/FlorinBalint/flo_lb/loadbalancer/config"
	pb "github.com/FlorinBalint/flo_lb/proto"
)

var (
	configFlag     = flag.String("config", "", "Config string to use for the load balancer")
	configFormat   = flag.String("config_format", "TEXT_PROTO", "Config format to use for the load balancer")
	configFileFlag = flag.String("config_file", "", "Config file to use for the load balancer")
	port           = flag.Int("port", 8080, "Override the port listening on")
)

func overridePortIfNeeded(cfg *pb.Config) {
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			*cfg.Port = int32(*port)
		}
	})
}

func readConfig() (*pb.Config, error) {
	if (len(*configFlag) == 0) == (len(*configFileFlag) == 0) {
		return nil, fmt.Errorf("Exactly one of --config or --config_file must pe specified !")
	}
	var lbCfg *pb.Config
	var err error

	if len(*configFlag) != 0 && len(*configFormat) != 0 {
		format, ok := (pb.ConfigFormat_value[*configFormat])
		if !ok {
			return nil, fmt.Errorf("Unknown config format %v", *configFormat)
		}
		lbCfg, err = config.Parse([]byte(*configFlag), (pb.ConfigFormat)(format))
	} else if len(*configFlag) != 0 && len(*configFormat) == 0 {
		return nil, fmt.Errorf("Must specify format when passing a string config")
	} else if len(*configFileFlag) != 0 {
		lbCfg, err = config.ParseFile(*configFileFlag)
	}
	if err != nil {
		return nil, fmt.Errorf("Error while parsing the configs: %v\n", err)
	}
	overridePortIfNeeded(lbCfg)
	return lbCfg, nil
}

func main() {
	flag.Parse()
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("Flag parsing error: %v", err)
	}
	lb, err := loadbalancer.New(cfg)
	if err != nil {
		log.Fatalf("Error creating a new load balancer: %v\n", err)
	}
	err = lb.ListenAndServe(context.Background())
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Proxy server was closed")
	} else if err != nil {
		fmt.Printf("error starting proxy: %v\n", err)
	}
}
