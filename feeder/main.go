package main

import (
	"compress/gzip"
	"encoding/gob"
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

	"github.com/gilliek/go-opml/opml"
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

var (
	feedParser *gofeed.Parser
	httpClient *http.Client
	cache      map[string]*metadata
)

var visited = make(map[string]bool)
var skip = func(s string) bool {
	s = strings.TrimSpace(s)
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

	f, err := opml.NewOPMLFromFile(blogrollFilePath)
	if err != nil {
		panic(err)
	}

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

	var mu sync.Mutex
	wg := sync.WaitGroup{}
	for _, o := range f.Outlines() {
		if len(o.Outlines) == 0 {
			wg.Go(func() {
				fmt.Println("processing feed:", o.Text)
				mu.Lock()
				articlesByCategory["misc"] = append(articlesByCategory["misc"], getArticles(o, &upperBound)...)
				mu.Unlock()
			})
		} else {
			for _, child := range o.Outlines {
				wg.Go(func() {
					fmt.Println("processing feed:", child.Text)
					mu.Lock()
					articlesByCategory[o.Text] = append(articlesByCategory[o.Text], getArticles(child, &upperBound)...)
					mu.Unlock()
				})
			}
		}
	}

	wg.Wait()

	for _, articles := range articlesByCategory {
		slices.SortFunc(articles, func(a, b *article) int {
			return b.Published.Compare(*a.Published)
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

	bs, err := yaml.Marshal(articlesByCategory)
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

	if meta, found := cache[feedUrl]; found {
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

	if _, found := cache[feedUrl]; !found {
		cache[feedUrl] = &metadata{}
	}

	if lastModified := response.Header.Get("Last-Modified"); lastModified != "" {
		cache[feedUrl].LastModified = lastModified
	}
	if eTag := response.Header.Get("ETag"); eTag != "" {
		cache[feedUrl].ETag = eTag
	}

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

func getArticles(f opml.Outline, upperBound *time.Time) (articles []*article) {
	feed, err := getFeed(f.XMLURL)
	if err != nil {
		fmt.Printf("err: cannot parse feed %q: %s\n", f.XMLURL, err)
		return
	}
	if feed == nil {
		return
	}

	for _, i := range feed.Items {
		if !skip(i.Title) && i.PublishedParsed.After(*upperBound) {
			link := i.Link
			if strings.HasPrefix(link, "/") {
				l, err := url.JoinPath(f.HTMLURL, link)
				if err != nil {
					fmt.Printf("err: cannot join paths: base=%s, elem=%s\n", f.HTMLURL, link)
					continue
				}
				link = l
			}
			articles = append(articles, &article{
				// override the RSS/Atom title with the UDF title in the OPML,
				// this can help with merging togheter feeds of authors blogging from different sources
				BlogName:  f.Text,
				Title:     i.Title,
				Url:       link,
				Published: i.PublishedParsed,
			})
		}
	}
	if len(articles) > 3 {
		articles = articles[:3]
	}
	return
}
