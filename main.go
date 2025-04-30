package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

const version = "v2.0.0"

func main() {
	renderHeader()
	setDNS()
	isAndroid := checkAndroid()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		login(isAndroid)
	}()

	go func() {
		defer wg.Done()
		configureBPB(isAndroid)
	}()

	server := &http.Server{Addr: ":8976"}
	http.HandleFunc("/oauth/callback", callback)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			failMessage("Error serving localhost.")
			log.Fatalln(err)
		}
	}()

	wg.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
}
