package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/alext/graceful_listener"
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

func main() {
	listener, apiListener, err := createListeners()
	if err != nil {
		log.Fatal(err)
	}

	m := martini.Classic()
	m.Get("/g", ExternalLinkTrackerHandler)
	mApi := martini.Classic()
	mApi.Put("/url", AddExternalURL)

	var wg sync.WaitGroup
	wg.Add(2)

	go serve(listener, m, &wg)
	log.Println("external-link-tracker: listening for redirects on " + listener.Addr().String())

	go serve(apiListener, mApi, &wg)
	log.Println("external-link-tracker: listening for writes on " + apiListener.Addr().String())

	wg.Wait()
}

func createListeners() (listener, apiListener *graceful_listener.Listener, err error) {
	listenFD, err := strconv.Atoi(getenvDefault("LISTEN_FD", "0"))
	if err != nil {
		log.Println("Non-integer LISTEN_FD, ignoring:", err)
		listenFD = 0
	}
	apiListenFD, err := strconv.Atoi(getenvDefault("API_LISTEN_FD", "0"))
	if err != nil {
		log.Println("Non-integer API_LISTEN_FD, ignoring:", err)
		apiListenFD = 0
	}

	listener, err = graceful_listener.ResumeOrStart(listenFD, pubAddr)
	if err != nil {
		return nil, nil, err
	}
	apiListener, err = graceful_listener.ResumeOrStart(apiListenFD, apiAddr)
	if err != nil {
		return nil, nil, err
	}

	return
}

func serve(l *graceful_listener.Listener, handler http.Handler, wg *sync.WaitGroup) {
	err := http.Serve(l, handler)
	if err != nil {
		log.Fatal("serve error: ", err)
	}
	wg.Done()
}
