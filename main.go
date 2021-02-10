package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/vrnvu/gbalancer/loadbalancer"
	"github.com/vrnvu/gbalancer/server"
)

func generateServerUrls(numberOfBackends int) []string {
	initialValue := 8080
	results := make([]string, 0)
	for i := 0; i < numberOfBackends; i++ {
		serverURLPortNumber := initialValue + i
		serverURLPort := ":" + strconv.Itoa(serverURLPortNumber)
		results = append(results, serverURLPort)
	}
	return results
}

func main() {
	var numberOfBackends int
	var serverPool loadbalancer.ServerPool

	flag.IntVar(&numberOfBackends, "b", 3, "Set the number of backend servers")
	flag.Parse()

	if numberOfBackends < 1 {
		log.Fatal("Invalid number of backend servers. Run with flag -b N, where N is bigger than 0")
	}

	serverURLs := generateServerUrls(numberOfBackends)

	for _, serverURL := range serverURLs {
		u, err := url.Parse("http://localhost" + serverURL)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(u)

		backend := loadbalancer.Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: proxy,
		}

		serverPool.AddBackend(&backend)

		// run all backends
		go server.RunBackend(serverURL)

	}

	// run lb server
	server := http.Server{
		Addr:    ":8000",
		Handler: http.HandlerFunc(serverPool.Loadbalance),
	}

	// To check the health of a backend we make a tcp connection with a timeout of 2
	// In order to iterate all backends in the worst case scenario
	// we will need an interval bigger than timeout * numberOfBackends
	// Se we "naively" add one second
	timeout := 2
	interval := numberOfBackends*timeout + 1
	go serverPool.HealthCheck(interval, timeout)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}
