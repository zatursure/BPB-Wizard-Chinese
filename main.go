package main

import (
	"context"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

const version = "v2.1.2"

var (
	srcPath    string
	workerPath string
	cachePath  string
	isAndroid  = false
	workerURL  = "https://github.com/bia-pain-bache/BPB-Worker-Panel/releases/latest/download/worker.js"
)

func main() {
	initPaths()
	setDNS()
	checkAndroid()
	if runtime.GOOS == "windows" {
		enableVirtualTerminalProcessing()
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		runWizard()
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
