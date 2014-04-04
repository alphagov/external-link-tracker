package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var (
	mgoSession      *mgo.Session
	mgoDatabaseName = getenvDefault("LINK_TRACKER_MONGO_DB", "external_link_tracker")
	mgoURL          = getenvDefault("LINK_TRACKER_MONGO_URL", "localhost")
)

// Store function in a variable so it can be overridden in the tests.
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
	Referrer    string    `bson:"referrer"`
}

// countHitOnURL logs a request time and HTTP 'referer' against the passed in
// URL
func countHitOnURL(url string, timeOfHit time.Time, referrer string) {
	session := getMgoSession()
	defer session.Close()
	session.SetMode(mgo.Strong, true)

	collection := session.DB(mgoDatabaseName).C("hits")

	err := collection.Insert(&ExternalLinkHit{
		ExternalURL: url,
		DateTime:    timeOfHit,
		Referrer:    referrer,
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
		go countHitOnURL(externalURL, now().UTC(), req.Referer())

		// Make sure this redirect is never cached
		w.Header().Set("Cache-control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		// Explicit 302 because this is a redirection proxy
		http.Redirect(w, req, externalURL, http.StatusFound)
	}
}

func saveExternalURL(url string) error {
	session := getMgoSession()
	defer session.Close()
	session.SetMode(mgo.Strong, true)

	collection := session.DB(mgoDatabaseName).C("links")

	err := collection.Find(bson.M{"external_url": url}).One(&ExternalLink{})

	if err != nil {
		if err.Error() != "not found" {
			return err
		}
		err1 := collection.Insert(&ExternalLink{
			ExternalURL: url,
		})

		if err1 != nil {
			return err1
		}
	}
	return nil
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

	err1 := saveExternalURL(externalURL)

	if err1 != nil {
		panic(err1)
	}

	return http.StatusCreated, "OK"
}

func healthcheck(w http.ResponseWriter, req *http.Request) (int, string) {
	return http.StatusOK, "OK"
}

// reports on the last 7 days of hits by URL
func ReportHitsByURL(w http.ResponseWriter, req *http.Request) (int, string) {
	session := getMgoSession()
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	collection := session.DB(mgoDatabaseName).C("hits")

	externalURL := req.URL.Query().Get("url")

	if externalURL == "" {
		return http.StatusNotFound, "URL is required"
	}

	today := now().UTC()
	oneWeekAgo := today.Add(-(24 * 7 * time.Hour))

	query := []bson.D{
		bson.D{
			{"$match",
				bson.M{
					"external_url": externalURL,
					"date_time": bson.M{
						"$gte": oneWeekAgo,
						"$lte": today,
					},
				},
			},
		},
		bson.D{
			{"$project",
				bson.M{
					"external_url": 1,
					"date": bson.M{
						"y": bson.M{
							"$year": "$date_time",
						},
						"m": bson.M{
							"$month": "$date_time",
						},
						"d": bson.M{
							"$dayOfMonth": "$date_time",
						},
					},
				},
			},
		},
		bson.D{
			{"$group",
				bson.M{
					"_id": bson.M{
						"p": "$external_url",
						"y": "$date.y",
						"m": "$date.m",
						"d": "$date.d",
					},
					"hits": bson.M{
						"$sum": 1,
					},
				},
			},
		},
	}

	results := collection.Pipe(query).Iter()

	w.Header().Set("Content-Type", "application/json")
	jsonPayload, err := json.Marshal(results)
	return http.StatusOK, jsonPayload

}
