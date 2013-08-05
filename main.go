package main

import (
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"newsserver/rssparse"
)

// TODO: Config
const (
	updateInterval = 300
	newsCount      = 20
)

var (
	newsList  NewsList
	cache     []byte           // Stores the data that will be served on each request.
	cacheLock sync.RWMutex     // Mutex for reading/writing to cache.
	dataSize  int              // Byte count of compressed data, for the protocol.
	done      = make(chan int) // Channel that signals if the webserver stops.

	sources = [...]FeedSource{
		{"NRK", "http://www.nrk.no/nyheiter/siste.rss", false},
		{"VG", "http://www.vg.no/rss/create.php?categories=10,12&keywords=&limit=10", true},
	}
)

// FeedSource represents an RSS feed.
type FeedSource struct {
	Name             string
	URL              string
	UseCharsetReader bool
}

// News represents a collection of news items.
type NewsList struct {
	News []NewsItem `json:"news"`
}

// NewsItem represents a single news item, corresponds with a single article.
type NewsItem struct {
	Title   string    `json:"title"`
	Text    string    `json:"text"`
	Link    string    `json:"link"`
	PubDate time.Time `json:"pubDate"`
}

func (n *NewsList) Len() int {
	return len(n.News)
}

// Swap() for the News list
func (n *NewsList) Swap(i, j int) {
	n.News[i], n.News[j] = n.News[j], n.News[i]
}

// Comparison function for the sorting interface
func (n *NewsList) Less(i, j int) bool {
	return n.News[i].PubDate.After(n.News[j].PubDate)
}

// UpdateMultiple updates the cache with new content
func UpdateMultiple() (data []byte, wait int64, err error) {
	var r *rssparse.Rss
	var jsonData []byte

	wait = updateInterval

	newsList.News = make([]NewsItem, 0, 25)

	log.Printf("Next update in %d seconds\n", wait)

	for _, source := range sources {
		if r, err = rssparse.GetRssFrom(source.URL, source.UseCharsetReader); err != nil {
			return
		}

		for _, item := range r.Channel.Items {
			const format = "Mon, 2 Jan 2006 15:04:05 -0700"
			var pubDate time.Time

			if pubDate, err = time.Parse(format, item.PubDate); err != nil {
				log.Printf("Error parsing time format:", err)
				return
			}

			newsList.News = append(newsList.News,
				NewsItem{
					"[" + source.Name + "] " + strings.Trim(item.Title, " "),
					strings.Trim(item.Description, " "),
					item.Link,
					pubDate,
				})
		}
	}

	// Sort to get the most recent news first
	sort.Sort(&newsList)

	if newsList.Len() > newsCount {
		newsList.News = newsList.News[:newsCount]
	}

	if jsonData, err = json.Marshal(newsList); err != nil {
		log.Printf("Error marshaling json data:", err)
		return
	}

	data = jsonData

	return
}

func main() {
	var err error
	var wait int64

	log.SetOutput(os.Stderr)

	newsList = NewsList{}

	log.Println("Fetching initial update...")
	cache, wait, err = UpdateMultiple()

	if err != nil {
		log.Println("Updating failed:", err)
		return
	}

	log.Println("Starting web server...")

	go webserver()

	for {
		select {
		case <-time.After(time.Duration(wait * 1e9)):
			log.Println("Updating...")

			cacheLock.Lock()
			cache, wait, err = UpdateMultiple()
			cacheLock.Unlock()

			if err != nil {
				log.Println("Updating failed:", err)
				return
			}

			log.Println("Finished updating")
		case <-done:
			log.Println("Webserver has stopped.")
			return
		}
	}
}
