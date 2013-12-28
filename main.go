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
	println("Counted hit:", url)
}

func externalLinkTrackerHandler(mongoUrl string, mongoDbName string) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		session := getMgoSession()
		defer session.Close()

		collection := session.DB(mgoDatabaseName).C("links")

		externalUrl := req.URL.Query().Get("url")

		result := ExternalLink{}
		err1 := collection.Find(bson.M{"external_url": externalUrl}).One(&result)

		if err1 != nil {
			if err1.Error() == "not found" {
				http.NotFound(w, req)
			} else {
				panic(err1)
			}
		} else {
			println("Found:", result.ExternalUrl)
			// Explicit 302 because this is a redirection proxy
			go countHit(externalUrl)
			http.Redirect(w, req, externalUrl, http.StatusFound)
		}
	}
}

func main() {
	http.HandleFunc("/g", externalLinkTrackerHandler("localhost", "external_link_tracker"))
	http.ListenAndServe("127.0.0.1:8080", nil)

}
