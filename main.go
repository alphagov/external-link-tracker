package main

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/martini"
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

func catchListenAndServe(addr string, handler http.Handler) {
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	m := martini.Classic()
	m.Get("/g", ExternalLinkTrackerHandler)
	mApi := martini.Classic()
	mApi.Put("/url", AddExternalURL)
	mApi.Get("/url", ReportHitsByURL)
	mApi.Get("/healthcheck", healthcheck)

	go catchListenAndServe(pubAddr, m)
	log.Println("external-link-tracker: listening for redirects on " + pubAddr)

	go catchListenAndServe(apiAddr, mApi)
	log.Println("external-link-tracker: listening for writes on " + apiAddr)

	dontQuit := make(chan int)
	<-dontQuit
}
