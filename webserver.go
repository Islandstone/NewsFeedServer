package main

import (
	"bytes"
	"log"
	"time"

	"encoding/json"
	"net/http"
)

// TODO: DRY up the code a little.

func writeHttpHeader(w http.ResponseWriter, contentType string, contentLength int) {
	header := w.Header()

	if contentType != "" {
		header.Add("Content-Type", contentType)
	}

	if contentLength != 0 {
		header.Add("Content-Length", string(contentLength))
	}
}

func handleRaw(w http.ResponseWriter, r *http.Request) {
	var cached, indented = &bytes.Buffer{}, &bytes.Buffer{}
	var err error

	log.Println("Serving raw data:", r.Host)

	cacheLock.RLock()
	defer cacheLock.RUnlock()

	writeHttpHeader(w, "", 0)

	cacheReader := bytes.NewReader(cache)
	if _, err = cached.ReadFrom(cacheReader); err != nil {
		log.Println("Failed to read from cache:", err)
		return
	}

	if err = json.Indent(indented, cached.Bytes(), "", "    "); err != nil {
		log.Println("Error indenting JSON:", err)
		log.Printf("JSON data:\n%s", cached.String())
		return
	}

	if _, err = w.Write(indented.Bytes()); err != nil {
		log.Println("Err writing result:", err)
		return
	}
}

func handleCached(w http.ResponseWriter, r *http.Request) {
	var err error

	cacheLock.RLock()
	defer cacheLock.RUnlock()

	log.Println("Serving:", r.Host)

	writeHttpHeader(w, "text/text", dataSize)

	if _, err = w.Write(cache); err != nil {
		log.Println("Err writing:", err)
		return
	}
}

func handleUpdates(w http.ResponseWriter, r *http.Request) {
	var jsonData []byte
	var dataSize int
	var last_update time.Time
	var err error

	time_str := r.URL.Query().Get("time")

	if len(time_str) == 0 {
		handleCached(w, r)
		return
	}

	newsUpdates := NewsList{}
	newsUpdates.News = make([]NewsItem, 0, 20)

	// TODO: Change this and the app to include seconds in the timestamp
	// TODO: Change the protocol to not include seconds
	// Note: + and - need to be escaped in the URL.
	const format = "20060102150405-0700"
	if last_update, err = time.Parse(format, time_str); err != nil {
		log.Println("Error parsing time string:", err)
		return
	}

	// Include only news that are more recent than the supplied timestamp.
	for _, news := range newsList.News {
		if !news.PubDate.After(last_update) {
			break
		}

		newsUpdates.News = append(newsUpdates.News, news)
	}

	// TODO: Consider sending a 304 instead?
	// If there's no new content, just send a 0, indicating nothing new.
	if len(newsUpdates.News) == 0 {
		if _, err = w.Write([]byte("0")); err != nil {
			log.Println("Failed to write a 0 to response")
			return
		}

		return
	}

	// TODO: This should not be indented if we are compressing the content.
	// Check the supplied HTTP header for it.
	if jsonData, err = json.MarshalIndent(newsUpdates, "", "    "); err != nil {
		log.Printf("Error marshaling json data:", err)
		return
	}

	var i int
	if i, err = w.Write(jsonData); err != nil {
		log.Println("Failed to write response:", err)
		return
	}

	dataSize += i

	writeHttpHeader(w, "text/text", dataSize)

	return
}

func webserver() {
	http.HandleFunc("/", makeGzipHandler(handleCached))
	http.HandleFunc("/raw", handleRaw)
	http.HandleFunc("/updates/raw", handleUpdates)
	http.HandleFunc("/updates", makeGzipHandler(handleUpdates))

	err := http.ListenAndServe(":9000", nil)
	log.Println("Listen and Serve Error:", err)

	done <- 0
}
