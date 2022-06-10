package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/FlorinBalint/flo-lb/loadbalancer"
	pb "github.com/FlorinBalint/flo_lb"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	config     = flag.String("config", "", "Config proto to use for the load balancer")
	configFile = flag.String("config_file", "", "Config file to use for the load balancer")
	port       = flag.Int("port", 8080, "Override the port listening on")
)

func parseConfig(cfg []byte) (*pb.Config, error) {
	res := &pb.Config{}
	err := prototext.Unmarshal(cfg, res)
	return res, err
}

func parseConfigFromFile(cfgPath string) (*pb.Config, error) {
	content, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	return parseConfig(content)
}

func overridePortIfNeeded(cfg *pb.Config) {
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			*cfg.Port = int32(*port)
		}
	})
}

func main() {
	flag.Parse()
	if (len(*config) == 0) == (len(*configFile) == 0) {
		log.Fatalf("Exactly one of --config or --config_file must pe specified !")
	}
	var lbCfg *pb.Config
	var err error
	if len(*config) != 0 {
		lbCfg, err = parseConfig([]byte(*config))
	} else if len(*configFile) != 0 {
		lbCfg, err = parseConfigFromFile(*configFile)
	}

	if err != nil {
		log.Fatalf("Error while parsing the configs: %v\n", err)
	}

	overridePortIfNeeded(lbCfg)
	lb, err := loadbalancer.New(lbCfg)
	if err != nil {
		log.Fatalf("Error creating a new load balancer: %v\n", err)
	}
	err = lb.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Proxy server was closed")
	} else if err != nil {
		fmt.Printf("error starting proxy: %v", err)
	}
}
