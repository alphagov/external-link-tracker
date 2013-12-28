package main

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
)

type ExternalLink struct {
	ExternalUrl string `bson:"external_url"`
	HitCount    int32  `bson:"hit_count"`
}

func externalLinkTrackerHandler(mongoUrl string, mongoDbName string) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		sess, err := mgo.Dial(mongoUrl)
		if err != nil {
			panic(fmt.Sprintln("mgo:", err))
		}
		defer sess.Close()
		sess.SetMode(mgo.Monotonic, true)

		db := sess.DB(mongoDbName)
		collection := db.C("links")

		external_url := req.URL.Query().Get("url")
		println(external_url)
		result := ExternalLink{}
		err1 := collection.Find(bson.M{"external_url": external_url}).One(&result)

		if err1 != nil {
			if err1.Error() == "not found" {
				http.NotFound(w, req)
			} else {
				panic(err1)
			}
		} else {
			println("Found:", result.ExternalUrl)
			// Explicit 302 because this is a redirection proxy
			http.Redirect(w, req, external_url, http.StatusFound)
		}
	}
}

func main() {
	http.HandleFunc("/g", externalLinkTrackerHandler("localhost", "external_link_tracker"))
	http.ListenAndServe("127.0.0.1:8080", nil)

}
