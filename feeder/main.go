package main

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	yaml "go.yaml.in/yaml/v4"
)

type article struct {
	BlogName  string     `yaml:"blogName,omitempty"`
	Title     string     `yaml:"title,omitempty"`
	Url       string     `yaml:"url,omitempty"`
	Published *time.Time `yaml:"published,omitempty"`
}

type metadata struct {
	LastModified string
	ETag         string
}

type opmlFile struct {
	Version float64 `xml:"version,attr"`
	Head    head    `xml:"head"`
	Body    body    `xml:"body"`
}

type body struct {
	Outlines []outline `xml:"outline"`
}

type outline struct {
	Text     string    `xml:"text,attr"`
	Title    string    `xml:"title,attr"`
	XMLUrl   string    `xml:"xmlUrl,attr"`
	HTMLUrl  string    `xml:"htmlUrl,attr"`
	Filters  string    `xml:"filters,attr"`
	Outlines []outline `xml:"outline"`
}

type head struct {
	Title string `xml:"title"`
}

var (
	feedParser *gofeed.Parser
	httpClient *http.Client
	cache      map[string]*metadata
	cacheMu    sync.RWMutex
)

var (
	visitedMu sync.Mutex
	visited   = make(map[string]bool)
)

var skip = func(s string) bool {
	s = strings.TrimSpace(s)
	visitedMu.Lock()
	defer visitedMu.Unlock()
	_, found := visited[s]
	if !found {
		visited[s] = true
	}
	return found
}

var userAgent = "giulianopz/feeder/0.1 (+https://github.com/giulianopz/giulianopz.github.io/tree/gh-pages/feeder)"

func init() {
	feedParser = gofeed.NewParser()

	httpClient = &http.Client{Timeout: 30 * time.Second}

	cache = make(map[string]*metadata)
}

func main() {
	var (
		blogrollFilePath = os.Args[1]
		feedYAMLPath     = os.Args[2]
		cachePath        = os.Args[3]
		timeFilter       = os.Args[4]
	)

	dur, err := time.ParseDuration(timeFilter)
	if err != nil {
		panic(err)
	}

	upperBound := time.Now().Add(-dur)

	bs, err := os.ReadFile(blogrollFilePath)
	if err != nil {
		panic(err)
	}

	f := opmlFile{}
	if err := xml.Unmarshal(bs, &f); err != nil {
		panic(err)
	}

	var articlesMu sync.Mutex
	articlesByCategory := make(map[string][]*article)
	if _, err := os.Stat(feedYAMLPath); !os.IsNotExist(err) {
		feedFile, err := os.Open(feedYAMLPath)
		if err != nil {
			panic(err)
		}
		if err := yaml.NewDecoder(feedFile).Decode(&articlesByCategory); err != nil {
			panic(err)
		}
	}

	// remove old articles
	for category, articles := range articlesByCategory {
		filtered := articles[:0]
		for _, a := range articles {
			if !skip(a.Title) && a.Published.After(upperBound) {
				filtered = append(filtered, a)
			}
		}
		articlesByCategory[category] = filtered
		clear(articles[len(filtered):])
	}

	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		cacheFile, err := os.Open(cachePath)
		if err != nil {
			panic(err)
		}
		if err := gob.NewDecoder(cacheFile).Decode(&cache); err != nil {
			if !errors.Is(err, io.EOF) {
				panic(err)
			}
		}
	}

	wg := sync.WaitGroup{}
	for _, o := range f.Body.Outlines {
		if len(o.Outlines) == 0 {
			wg.Go(func() {
				fmt.Println("processing feed:", o.Text)
				articles := getArticles(o, &upperBound)
				articlesMu.Lock()
				articlesByCategory["misc"] = append(articlesByCategory["misc"], articles...)
				articlesMu.Unlock()
			})
		} else {
			for _, child := range o.Outlines {
				wg.Go(func() {
					fmt.Println("processing feed:", child.Text)
					articles := getArticles(child, &upperBound)
					articlesMu.Lock()
					articlesByCategory[o.Text] = append(articlesByCategory[o.Text], articles...)
					articlesMu.Unlock()
				})
			}
		}
	}

	wg.Wait()

	for _, articles := range articlesByCategory {
		slices.SortFunc(articles, func(a, b *article) int {
			sortScore := b.Published.Compare(*a.Published)
			if sortScore == 0 {
				sortScore = strings.Compare(b.Title, a.Title)
			}
			return sortScore
		})
	}

	// limit articles per author to 3
	articlesNum := make(map[string]int)
	for category, articles := range articlesByCategory {
		filtered := articles[:0]
		for _, a := range articles {
			articlesNum[a.BlogName]++
			if articlesNum[a.BlogName] <= 3 {
				filtered = append(filtered, a)
			}
		}
		articlesByCategory[category] = filtered
		clear(articles[len(filtered):])
	}

	bs, err = yaml.Marshal(articlesByCategory)
	if err != nil {
		panic(err)
	}
	os.WriteFile(feedYAMLPath, bs, fs.ModePerm)
	fmt.Println("feed updated")

	cacheFile, err := os.Create(cachePath)
	if err != nil {
		panic(err)
	}
	if err := gob.NewEncoder(cacheFile).Encode(cache); err != nil {
		panic(err)
	}
	fmt.Println("cache updated")
}

func getFeed(feedUrl string) (*gofeed.Feed, error) {
	request, err := http.NewRequest("GET", feedUrl, nil)
	if err != nil {
		return nil, err

	}
	request.Header.Add("User-Agent", userAgent)
	request.Header.Add("Accept-Encoding", "gzip")

	cacheMu.RLock()
	meta, found := cache[feedUrl]
	cacheMu.RUnlock()
	if found {
		if meta.LastModified != "" {
			request.Header.Add("If-Modified-Since", meta.LastModified)
		}
		if meta.ETag != "" {
			request.Header.Add("If-None-Match", meta.ETag)
		}
	}

	var attempts int
retry:
	attempts++
	if attempts > 3 {
		return nil, fmt.Errorf("reached max num of retries")
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified {
		fmt.Println("no updates from:", feedUrl)
		return nil, nil
	}

	if response.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("too many requests: %s", feedUrl)
	}

	if response.StatusCode == http.StatusNotAcceptable {
		acceptableResp, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read resp: %w", err)
		}
		fmt.Printf("req not accepted: %s\n", string(acceptableResp))

		request.Header.Del("Accept-Encoding")
		goto retry
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("err: resp status code: %d: %s", response.StatusCode, feedUrl)
	}

	cacheMu.Lock()
	if _, found := cache[feedUrl]; !found {
		cache[feedUrl] = &metadata{}
	}

	if lastModified := response.Header.Get("Last-Modified"); lastModified != "" {
		cache[feedUrl].LastModified = lastModified
	}
	if eTag := response.Header.Get("ETag"); eTag != "" {
		cache[feedUrl].ETag = eTag
	}
	cacheMu.Unlock()

	var reader io.ReadCloser
	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(response.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
	default:
		reader = response.Body
	}

	return feedParser.Parse(reader)
}

type filter func(*gofeed.Item) bool

func parseFilters(filters string) (funcs []filter) {
	for f := range strings.SplitSeq(filters, ",") {
		kv := strings.Split(f, "=")

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "author":
			funcs = append(funcs, func(i *gofeed.Item) bool {
				return len(i.Authors) != 0 && i.Authors[0].Name == value
			})
		case "category":
			funcs = append(funcs, func(i *gofeed.Item) bool {
				return slices.Contains(i.Categories, value)
			})
		}
	}
	return
}

func recent(publischedTime, upperBound *time.Time) bool {
	return publischedTime != nil && publischedTime.After(*upperBound)
}

func match(filters []filter, article *gofeed.Item) bool {
	for _, f := range filters {
		if !f(article) {
			return false
		}
	}
	return true
}

func getArticles(f outline, upperBound *time.Time) (articles []*article) {
	feed, err := getFeed(f.XMLUrl)
	if err != nil {
		fmt.Printf("err: cannot parse feed %q: %s\n", f.XMLUrl, err)
		return
	}
	if feed == nil {
		return
	}

	var filters []filter
	if f.Filters != "" {
		filters = parseFilters(f.Filters)
	}
	for _, i := range feed.Items {
		title := stripCDATA(i.Title)
		if !skip(title) &&
			recent(i.PublishedParsed, upperBound) &&
			(f.Filters == "" || match(filters, i)) {
			link := i.Link
			if strings.HasPrefix(link, "/") {
				l, err := url.JoinPath(f.HTMLUrl, link)
				if err != nil {
					fmt.Printf("err: cannot join paths: base=%s, elem=%s\n", f.HTMLUrl, link)
					continue
				}
				link = l
			}
			articles = append(articles, &article{
				// override the RSS/Atom title with the UDF title in the OPML,
				// this can help with merging togheter feeds of authors blogging from different sources
				BlogName:  f.Text,
				Title:     title,
				Url:       link,
				Published: i.PublishedParsed,
			})
		}
	}
	return
}

func stripCDATA(s string) string {
	if strings.Contains(s, "<![CDATA[") {
		s = strings.NewReplacer("<![CDATA[", "", "]]>", "").Replace(s)
		s = strings.TrimSpace(s)
	}
	return s
}
