package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
)

var (
	port = flag.Int("port", 8081, "Port to listen to")
	name = flag.String("name", "Server", "Name of the service")
)

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}

func getHello(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /hello request\n")
	resp := fmt.Sprintf("Hello from %v\n", *name)
	io.WriteString(w, resp)
}

func iAmAlive(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /healthz request\n")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	flag.Parse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/hello", getHello)
	mux.HandleFunc("/healthz", iAmAlive)

	address := fmt.Sprintf("localhost:%v", *port)
	fmt.Printf("%v will start listening on %v\n", *name, *port)

	err := http.ListenAndServe(address, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error listening for server: %s\n", err)
	}
}
