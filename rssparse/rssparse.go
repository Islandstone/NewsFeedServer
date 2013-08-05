package rssparse

import (
	"bytes"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Rss represents the contents of an rss feed
type Rss struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Channel represents a single channel in the rss feed
type Channel struct {
	Items        []Item        `xml:"item"`
	PubDate      string        `xml:"pubDate"`
	UpdateFreq   time.Duration `xml:"updateFrequency"`
	UpdatePeriod string        `xml:"updatePeriod"`
}

// Item represents a single entry in a channel
type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	Enclosure   Enclosure `xml:"enclosure"`
}

// Enclosure TODO
type Enclosure struct {
	URL string `xml:"url,attr"`
}

// GetRssFrom will get the RSS feed from the server at uri and parse it, using
// a charset reader for ISO-88591 if useCharsetReader is true.
func GetRssFrom(uri string, useCharsetReader bool) (r *Rss, err error) {
	var body []byte
	var resp *http.Response
	r = nil

	resp, err = http.Get(uri)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	r = &Rss{}

	b := bytes.NewReader(body)

	p := xml.NewDecoder(b)

	if useCharsetReader {
		p.CharsetReader = CharsetReader
	}

	err = p.Decode(r)

	return
}

type CharsetISO88591er struct {
	r   io.ByteReader
	buf *bytes.Buffer
}

func NewCharsetISO88591(r io.Reader) *CharsetISO88591er {
	buf := bytes.Buffer{}
	return &CharsetISO88591er{r.(io.ByteReader), &buf}
}

func (cs *CharsetISO88591er) Read(p []byte) (n int, err error) {
	for _ = range p {
		if r, err := cs.r.ReadByte(); err != nil {
			break
		} else {
			if _, err := cs.buf.WriteRune(rune(r)); err != nil {
				return 0, err
			}
		}
	}
	return cs.buf.Read(p)
}

func isCharset(charset string, names []string) bool {
	charset = strings.ToLower(charset)
	for _, n := range names {
		if charset == strings.ToLower(n) {
			return true
		}
	}
	return false
}

func IsCharsetISO88591(charset string) bool {
	// http://www.iana.org/assignments/character-sets
	// (last updated 2010-11-04)
	names := []string{
		// Name
		"ISO_8859-1:1987",
		// Alias (preferred MIME name)
		"ISO-8859-1",
		// Aliases
		"iso-ir-100",
		"ISO_8859-1",
		"latin1",
		"l1",
		"IBM819",
		"CP819",
		"csISOLatin1",
	}

	return isCharset(charset, names)
}

func CharsetReader(charset string, input io.Reader) (io.Reader, error) {
	if IsCharsetISO88591(charset) {
		return NewCharsetISO88591(input), nil
	}

	return input, nil
}
