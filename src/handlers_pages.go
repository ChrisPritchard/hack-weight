package main

import "net/http"

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}
	renderView(w, r, nil, "index.html", "Home")
}
