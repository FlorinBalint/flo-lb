package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	pb "github.com/FlorinBalint/flo_lb/proto"
	"google.golang.org/protobuf/proto"
)

var (
	hostOverride  = flag.String("host_override", "", "Host to announce himself instead of localhst")
	port          = flag.Int("port", 8081, "Port to listen to")
	name          = flag.String("name", "Server", "Name of the service")
	registerURL   = flag.String("register_url", "", "URL for registering to the load balancer")
	deregisterURL = flag.String("deregister_url", "", "URL for registering to the load balancer")
)

func getRoot(w http.ResponseWriter, r *http.Request) {
	log.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}

func getHello(w http.ResponseWriter, r *http.Request) {
	log.Printf("got /hello request\n")
	resp := fmt.Sprintf("Hello from %v\n", *name)
	io.WriteString(w, resp)
}

func iAmAlive(w http.ResponseWriter, r *http.Request) {
	log.Printf("got /healthz request\n")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func registerRequest() (*http.Request, error) {
	bodyReq := &pb.RegisterRequest{
		Port: proto.Int32(int32(*port)),
	}

	if len(*hostOverride) != 0 {
		bodyReq.Host = hostOverride
	} else {
		bodyReq.Host = proto.String("localhost")
	}

	if body, err := proto.Marshal(bodyReq); err != nil {
		return nil, fmt.Errorf("Unable to create register request: %v", err)
	} else {
		return http.NewRequest(
			"POST", *registerURL, bytes.NewBuffer(body),
		)
	}
}

func deregisterRequest() (*http.Request, error) {
	bodyReq := &pb.DeregisterRequest{
		Port: proto.Int32(int32(*port)),
	}

	if len(*hostOverride) != 0 {
		bodyReq.Host = hostOverride
	} else {
		bodyReq.Host = proto.String("localhost")
	}

	if body, err := proto.Marshal(bodyReq); err != nil {
		return nil, fmt.Errorf("Unable to create register request: %v", err)
	} else {
		return http.NewRequest(
			"POST", *deregisterURL, bytes.NewBuffer(body),
		)
	}
}

func registerToLBIfNeeded() error {
	if len(*registerURL) != 0 {
		register, err := registerRequest()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Registering to %v", register.URL)
		resp, err := http.DefaultClient.Do(register)
		if err != nil {
			return fmt.Errorf("Error registering %v", err)
		} else if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Register got non OK status %v", resp.StatusCode)
		}
	}
	return nil
}

func deregisterIfNeeded() {
	if len(*deregisterURL) != 0 {
		deregister, err := deregisterRequest()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("De-registering to %v", deregister.URL)
		resp, err := http.DefaultClient.Do(deregister)
		if err != nil {
			log.Fatalf("Error deregistering %v", err)
		} else if resp.StatusCode != http.StatusOK {
			log.Fatalf("Deregister got non OK status %v", resp.StatusCode)
		}
	}
}

func main() {
	flag.Parse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/hello", getHello)
	mux.HandleFunc("/healthz", iAmAlive)
	log.Printf("%v will start listening on %v\n", *name, *port)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", *port),
		Handler: mux,
	}

	srv.RegisterOnShutdown(deregisterIfNeeded)
	err := registerToLBIfNeeded()
	if err != nil {
		log.Fatalf("Error registering to the LB")
	}

	err = srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Printf("server closed\n")
	} else if err != nil {
		log.Printf("error listening for server: %s\n", err)
	}
}
