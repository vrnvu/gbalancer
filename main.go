package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/vrnvu/gbalancer/loadbalancer"
	"github.com/vrnvu/gbalancer/server"
)

func main() {
	var serverPool loadbalancer.ServerPool

	serverURLs := [3]string{
		":8081",
		":8082",
		":8083",
	}

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

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}
