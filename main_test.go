package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestNoRecordReturns404(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB("external_link_tracker_test").DropDatabase()

	request, _ := http.NewRequest("GET", "/g", nil)
	response := httptest.NewRecorder()

	ExternalLinkTrackerHandler("localhost", "external_link_tracker_test")(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "404", response.Code)
	}
}

func TestExistingUrlIsRedirected(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB("external_link_tracker_test").DropDatabase()

	collection := mgoSession.DB("external_link_tracker_test").C("links")
	collection.Insert(&ExternalLink{ExternalUrl: "http://example.com"})

	queryParam := url.QueryEscape("http://example.com")

	request, _ := http.NewRequest("GET", "/g?url="+queryParam, nil)
	response := httptest.NewRecorder()

	ExternalLinkTrackerHandler("localhost", "external_link_tracker_test")(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("Expected 302, got %v", response.Code)
	}

	redirectedTo := response.Header().Get("Location")

	if redirectedTo != "http://example.com" {
		t.Fatalf("Expected 'http://example.com', got %v", redirectedTo)
	}
}

func TestRedirectHasNoCache(t *testing.T) {
	mgoSession, _ := mgo.Dial("localhost")
	defer mgoSession.DB("external_link_tracker_test").DropDatabase()

	collection := mgoSession.DB("external_link_tracker_test").C("links")
	collection.Insert(&ExternalLink{ExternalUrl: "http://example.com"})

	queryParam := url.QueryEscape("http://example.com")

	request, _ := http.NewRequest("GET", "/g?url="+queryParam, nil)
	response := httptest.NewRecorder()

	ExternalLinkTrackerHandler("localhost", "external_link_tracker_test")(response, request)

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
	defer mgoSession.DB("external_link_tracker_test").DropDatabase()

	mgoSession.DB("external_link_tracker_test").C("links").Insert(&ExternalLink{ExternalUrl: "http://example.com"})

	queryParam := url.QueryEscape("http://example.com")

	request, _ := http.NewRequest("GET", "/g?url="+queryParam, nil)
	response := httptest.NewRecorder()

	ExternalLinkTrackerHandler("localhost", "external_link_tracker_test")(response, request)

	// sleep so the goroutine definitely fires
	time.Sleep(1)

	collection := mgoSession.DB(mgoDatabaseName).C("hits")

	result := ExternalLinkHit{}

	err := collection.Find(bson.M{"external_url": "http://example.com"}).One(&result)

	if err != nil {
		if err.Error() == "not found" {
			t.Fatal("Couldn't find record")
		} else {
			t.Fatalf("Mongo error: %v", err.Error())
		}
	}

	if result.ExternalUrl != "http://example.com" {
		t.Fatalf("Inserted wrong value, %v", result.ExternalUrl)
	}
}
