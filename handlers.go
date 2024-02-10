package main

import (
	"net/http"
)

func handleIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://github.com/bndw/nostr-relay-proxy", http.StatusTemporaryRedirect)
}
