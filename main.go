package main

import (
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/codegangsta/martini"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var (
	mgoSession      *mgo.Session
	pubAddr         = getenvDefault("LINK_TRACKER_PUBADDR", ":8080")
	apiAddr         = getenvDefault("LINK_TRACKER_APIADDR", ":8081")
	mgoDatabaseName = getenvDefault("LINK_TRACKER_MONGO_DB", "external_link_tracker")
	mgoURL          = getenvDefault("LINK_TRACKER_MONGO_URL", "localhost")
)

var now = time.Now

func getMgoSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial(mgoURL)
		if err != nil {
			panic(err) // no, not really
		}
	}
	return mgoSession.Clone()
}

type ExternalLink struct {
	ExternalURL string `bson:"external_url"`
}

type ExternalLinkHit struct {
	ExternalURL string    `bson:"external_url"`
	DateTime    time.Time `bson:"date_time"`
}

// countHitOnURL logs a request time against the passed in URL
func countHitOnURL(url string, timeOfHit time.Time) {
	session := getMgoSession()
	defer session.Close()
	session.SetMode(mgo.Strong, true)

	collection := session.DB(mgoDatabaseName).C("hits")

	err := collection.Insert(&ExternalLinkHit{
		ExternalURL: url,
		DateTime:    timeOfHit,
	})

	if err != nil {
		panic(err)
	}
}

// ExternalLinkTrackerHandler looks up the `url` against a database whitelist,
// and if it exists redirects to that URL while logging the request in the
// background. It will 404 if the whitelist doesn't pass.
func ExternalLinkTrackerHandler(w http.ResponseWriter, req *http.Request) {
	session := getMgoSession()
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	collection := session.DB(mgoDatabaseName).C("links")

	externalURL := req.URL.Query().Get("url")

	err := collection.Find(bson.M{"external_url": externalURL}).One(&ExternalLink{})

	if err != nil {
		if err.Error() == "not found" {
			http.NotFound(w, req)

		} else {
			panic(err)
		}
	} else {
		go countHitOnURL(externalURL, now().UTC())

		// Make sure this redirect is never cached
		w.Header().Set("Cache-control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		// Explicit 302 because this is a redirection proxy
		http.Redirect(w, req, externalURL, http.StatusFound)
	}
}

func saveExternalURL(url string) {
	session := getMgoSession()
	defer session.Close()
	session.SetMode(mgo.Strong, true)

	collection := session.DB(mgoDatabaseName).C("links")

	err := collection.Find(bson.M{"external_url": url}).One(&ExternalLink{})

	if err != nil {
		if err.Error() == "not found" {
			err1 := collection.Insert(&ExternalLink{
				ExternalURL: url,
			})

			if err1 != nil {
				panic(err)
			}
		}
	}
}

// AddExternalUrl allows an external URL to be added to the database
func AddExternalURL(w http.ResponseWriter, req *http.Request) (int, string) {
	externalURL := req.URL.Query().Get("url")

	if externalURL == "" {
		return http.StatusBadRequest, "URL is required"
	}

	parsedURL, err := url.Parse(externalURL)

	if err != nil {
		panic(err)
	}

	if !parsedURL.IsAbs() {
		return http.StatusBadRequest, "URL is not absolute"
	}

	go saveExternalURL(externalURL)
	return http.StatusCreated, "OK"
}

func getenvDefault(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		val = defaultVal
	}

	return val
}

func main() {
	m := martini.Classic()
	m.Get("/g", ExternalLinkTrackerHandler)
	m.Put("/url", AddExternalURL)
	http.ListenAndServe(pubAddr, m)
}
