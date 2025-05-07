package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	red    = "\033[31m"
	green  = "\033[32m"
	reset  = "\033[0m"
	orange = "\033[38;2;255;165;0m"
	blue   = "\033[94m"
	bold   = "\033[1m"
	title  = bold + blue + "●" + reset
	ask    = bold + "-" + reset
	info   = bold + "+" + reset
)

func checkAndroid() {
	path := os.Getenv("PATH")
	if runtime.GOOS == "android" || strings.Contains(path, "com.termux") {
		prefix := os.Getenv("PREFIX")
		certPath := filepath.Join(prefix, "etc/tls/cert.pem")
		if err := os.Setenv("SSL_CERT_FILE", certPath); err != nil {
			failMessage("Failed to set Termux cert file.")
			log.Fatalln(err)
		}
		isAndroid = true
	}
}

func setDNS() {
	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					conn, err := net.Dial("udp", "8.8.8.8:53")
					if err != nil {
						failMessage("Failed to dial DNS. Please disconnect your VPN and try again...")
						log.Fatal(err)
					}
					return conn, nil
				},
			},
		}
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			failMessage("DNS resolution failed. Please disconnect your VPN and try again...")
			log.Fatal(err)
		}
		return conn, nil
	}

}

func renderHeader() {
	fmt.Printf(`
■■■■■■■  ■■■■■■■  ■■■■■■■ 
■■   ■■  ■■   ■■  ■■   ■■
■■■■■■■  ■■■■■■■  ■■■■■■■ 
■■   ■■  ■■       ■■   ■■
■■■■■■■  ■■       ■■■■■■■  %sWizard%s %s%s
`,
		bold+green,
		reset+green,
		version,
		reset)
}

func initPaths() {
	var err error
	srcPath, err = os.MkdirTemp("", ".bpb-wizard")
	if err != nil {
		failMessage("Failed to create temp directory.")
		log.Fatalln(err)
	}

	workerPath = filepath.Join(srcPath, "worker.js")
	cachePath = filepath.Join(srcPath, "tld.cache")
}
