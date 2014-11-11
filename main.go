package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/alext/tablecloth"
)

var (
	pubAddr = getenvDefault("LINK_TRACKER_PUBADDR", ":8080")
	apiAddr = getenvDefault("LINK_TRACKER_APIADDR", ":8081")
)

func getenvDefault(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		val = defaultVal
	}

	return val
}

func catchListenAndServe(addr string, handler http.Handler, ident string, wg *sync.WaitGroup) {
	defer wg.Done()
	err := tablecloth.ListenAndServe(addr, handler, ident)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if wd := os.Getenv("GOVUK_APP_ROOT"); wd != "" {
		tablecloth.WorkingDir = wd
	}

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("/g", ExternalLinkTrackerHandler)

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/url", AddExternalURL)
	apiMux.HandleFunc("/healthcheck", healthcheck)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go catchListenAndServe(pubAddr, publicMux, "redirects", wg)
	log.Println("external-link-tracker: listening for redirects on " + pubAddr)

	go catchListenAndServe(apiAddr, apiMux, "api", wg)
	log.Println("external-link-tracker: listening for writes on " + apiAddr)

	wg.Wait()
}
