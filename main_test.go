package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/codegangsta/martini"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// forces now() to return a specific time
func nowForce(unix int) {
	now = func() time.Time {
		return time.Unix(int64(unix), 0)
	}
}

func TestNoRecordReturns404(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB(mgoDatabaseName).DropDatabase()

	request, _ := http.NewRequest("GET", "/g", nil)
	response := httptest.NewRecorder()

	ExternalLinkTrackerHandler(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "404", response.Code)
	}
}

func TestExistingUrlIsRedirected(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB("external_link_tracker_test").DropDatabase()

	externalURL := "http://1.example.com"

	collection := mgoSession.DB(mgoDatabaseName).C("links")
	collection.Insert(&ExternalLink{ExternalUrl: externalURL})

	queryParam := url.QueryEscape(externalURL)

	request, _ := http.NewRequest("GET", "/g?url="+queryParam, nil)
	response := httptest.NewRecorder()

	ExternalLinkTrackerHandler(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("Expected 302, got %v", response.Code)
	}

	redirectedTo := response.Header().Get("Location")

	if redirectedTo != externalURL {
		t.Fatalf("Expected '%v', got '%v'", externalURL, redirectedTo)
	}
}

func TestRedirectHasNoCache(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB(mgoDatabaseName).DropDatabase()

	externalURL := "http://2.example.com"

	collection := mgoSession.DB(mgoDatabaseName).C("links")
	collection.Insert(&ExternalLink{ExternalUrl: externalURL})

	queryParam := url.QueryEscape(externalURL)

	request, _ := http.NewRequest("GET", "/g?url="+queryParam, nil)
	response := httptest.NewRecorder()

	ExternalLinkTrackerHandler(response, request)

	cacheControl := response.Header().Get("Cache-control")
	pragma := response.Header().Get("Pragma")
	expires := response.Header().Get("Expires")

	if cacheControl != "no-cache, no-store, must-revalidate" {
		t.Fatalf("Expected no caching, got %v", cacheControl)
	}

	if pragma != "no-cache" {
		t.Fatalf("Expected pragma: no-cache, got pragma: %v", pragma)
	}

	if expires != "0" {
		t.Fatalf("Expected expires: 0, got expires: %v", expires)
	}
}

func TestHitsAreLogged(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB(mgoDatabaseName).DropDatabase()

	externalURL := "http://3.example.com"

	mgoSession.DB(mgoDatabaseName).C("links").Insert(&ExternalLink{ExternalUrl: externalURL})

	queryParam := url.QueryEscape(externalURL)

	request, _ := http.NewRequest("GET", "/g?url="+queryParam, nil)
	response := httptest.NewRecorder()

	// lock time
	nowForce(1388577600) // 2014-01-01T12:00:00z

	ExternalLinkTrackerHandler(response, request)

	// sleep so the goroutine definitely fires
	time.Sleep(100 * time.Millisecond)

	collection := mgoSession.DB(mgoDatabaseName).C("hits")

	result := ExternalLinkHit{}

	err := collection.Find(bson.M{"external_url": externalURL}).One(&result)

	if err != nil {
		if err.Error() == "not found" {
			t.Fatal("Couldn't find record")
		} else {
			t.Fatalf("Mongo error: %v", err.Error())
		}
	}

	expectedDate := time.Unix(int64(1388577600), 0)

	if result.DateTime != expectedDate {
		t.Fatalf("DateTime: Got %v, expected %v", result.DateTime.Unix(), expectedDate.Unix())
	}
}

func TestAPINoURLReturns400(t *testing.T) {
	request, _ := http.NewRequest("PUT", "/url", nil)
	response := httptest.NewRecorder()

	m := martini.Classic()
	m.Put("/url", AddExternalURL)
	m.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("Got status %v, expected %v", response.Code, http.StatusBadRequest)
	}
}

func TestAPIBadURLReturns400(t *testing.T) {
	queryParam := url.QueryEscape("relative-url.com")
	request, _ := http.NewRequest("PUT", "/url?url="+queryParam, nil)
	response := httptest.NewRecorder()

	m := martini.Classic()
	m.Put("/url", AddExternalURL)
	m.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("Got status %v, expected %v", response.Code, http.StatusBadRequest)
	}
}

func TestAPIGoodURLReturns201(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB(mgoDatabaseName).DropDatabase()

	queryParam := url.QueryEscape("http://good-url.com")
	request, _ := http.NewRequest("PUT", "/url?url="+queryParam, nil)
	response := httptest.NewRecorder()

	m := martini.Classic()
	m.Put("/url", AddExternalURL)
	m.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("Got status %v, expected %v", response.Code, http.StatusCreated)
	}

}

func TestAPIGoodURLIsSaved(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB(mgoDatabaseName).DropDatabase()

	queryParam := url.QueryEscape("http://good-url.com")
	request, _ := http.NewRequest("PUT", "/url?url="+queryParam, nil)
	response := httptest.NewRecorder()

	m := martini.Classic()
	m.Put("/url", AddExternalURL)
	m.ServeHTTP(response, request)

	// sleep so the goroutine definitely fires
	time.Sleep(100 * time.Millisecond)

	collection := mgoSession.DB(mgoDatabaseName).C("links")

	result := ExternalLink{}

	err := collection.Find(bson.M{"external_url": "http://good-url.com"}).One(&result)

	if err != nil {
		if err.Error() == "not found" {
			t.Fatal("Couldn't find record")
		} else {
			t.Fatalf("Mongo error: %v", err.Error())
		}
	}

	if result.ExternalUrl != "http://good-url.com" {
		t.Fatalf("Inserted wrong value, %v", result.ExternalUrl)
	}
}
