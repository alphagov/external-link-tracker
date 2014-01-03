package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
)

var (
	mgoSession      *mgo.Session
	mgoDatabaseName = "external_link_tracker"
)

func getMgoSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial("localhost")
		if err != nil {
			panic(err) // no, not really
		}
	}
	return mgoSession.Clone()
}

type ExternalLink struct {
	ExternalUrl string `bson:"external_url"`
	HitCount    int32  `bson:"hit_count"`
}

func countHit(url string) {
	session := getMgoSession()
	defer session.Close()
	session.SetMode(mgo.Strong, true)

	collection := session.DB(mgoDatabaseName).C("links")

	err := collection.Update(bson.M{"external_url": url}, bson.M{
		"$inc": bson.M{"hit_count": 1},
	})

	if err != nil {
		panic(err)
	}
}

func externalLinkTrackerHandler(mongoUrl string, mongoDbName string) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		session := getMgoSession()
		defer session.Close()
		session.SetMode(mgo.Monotonic, true)

		collection := session.DB(mgoDatabaseName).C("links")

		externalUrl := req.URL.Query().Get("url")

		result := ExternalLink{}
		err := collection.Find(bson.M{"external_url": externalUrl}).One(&result)

		if err != nil {
			if err.Error() == "not found" {
				http.NotFound(w, req)
			} else {
				panic(err)
			}
		} else {
			go countHit(externalUrl)

			// Make sure this redirect is never cached
			w.Header().Set("Cache-control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			// Explicit 302 because this is a redirection proxy
			http.Redirect(w, req, externalUrl, http.StatusFound)
		}
	}
}

func main() {
	http.HandleFunc("/g", externalLinkTrackerHandler("localhost", "external_link_tracker"))
	http.ListenAndServe("127.0.0.1:8080", nil)
}
