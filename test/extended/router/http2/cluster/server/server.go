package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	defaultPort   = "8443"
	defaultTLSCrt = "/etc/service-certs/tls.crt"
	defaultTLSKey = "/etc/service-certs/tls.key"
)

func lookupEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func main() {
	crt := lookupEnv("TLS_CRT", defaultTLSCrt)
	key := lookupEnv("TLS_KEY", defaultTLSKey)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(fmt.Sprintf("Hello %q, request protocol is %q\n", req.RemoteAddr, req.Proto)))
	})

	log.Printf("Listening on port 8443")

	if err := http.ListenAndServeTLS(":"+lookupEnv("PORT", defaultPort), crt, key, nil); err != nil {
		log.Fatal(err)
	}
}
