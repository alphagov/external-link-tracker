package main

import (
	"net/http"
)

func externalLinkHandler(w http.ResponseWrite, r *http.Request) {
	external_url := r.URL.Query().Get("url")
	// Explicit 302 because this is a redirection proxy
	http.Redirect(w, r, external_url, http.StatusFound)
}

func main() {
	http.HandleFunc("/g", externalLinkHandler)
	http.ListenAndServe("127.0.0.1:8080", nil)
}
