package main

import (
	"context"
	"dht-crawler/dht"
	"html/template"
	"log"
	"net/http"
	"time"
)

func Crawler() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(120 * time.Second)
		cancel()
	}()
	dht.CrawlDHT(ctx)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("template/search.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	data := dht.Query(query)

	t, err := template.ParseFiles("template/search.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	t.Execute(w, struct{ SearchResult []dht.SearchResult }{data})
}

func main() {
	dht.InitDB()
	defer dht.CloseDB()

	// Crawler()

	http.HandleFunc("/", searchHandler)
	http.HandleFunc("/search", queryHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
