package main

import (
	"context"
	"fmt"
	"io/fs"
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

var mu sync.Mutex

var visited = make(map[string]bool)
var skip = func(s string) bool {
	s = strings.TrimSpace(s)
	if _, found := visited[s]; found {
		return true
	}
	visited[s] = true
	return false
}

func main() {
	var (
		blogrollFilePath = os.Args[1]
		feedYAMLPath     = os.Args[2]
		timeFilter       = os.Args[3]
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

	wg := sync.WaitGroup{}

	articlesByCategory := make(map[string][]*article)
	for _, o := range f.Outlines() {
		if len(o.Outlines) == 0 {
			wg.Go(func() {
				fmt.Println("processing feed:", o.Text)
				articlesByCategory["misc"] = append(articlesByCategory["misc"], getArticles(o.Title, o.XMLURL, &upperBound)...)
			})
		} else {
			for _, child := range o.Outlines {
				wg.Go(func() {
					fmt.Println("processing feed:", child.Text)
					articlesByCategory[o.Text] = append(articlesByCategory[o.Text], getArticles(child.Title, child.XMLURL, &upperBound)...)
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

	bs, err := yaml.Marshal(articlesByCategory)
	if err != nil {
		panic(err)
	}
	os.WriteFile(feedYAMLPath, bs, fs.ModePerm)
	fmt.Println("feed updated")
}

var p *gofeed.Parser

func init() {
	p = gofeed.NewParser()
	p.UserAgent = "giulianopz/feeder/0.1 (https://github.com/giulianopz/giulianopz.github.io/feeder)"
	// TODO: implement polite feeder:
	// - https://rachelbythebay.com/w/2022/03/07/get/
	// - https://rachelbythebay.com/w/2023/01/18/http/
	// - https://rachelbythebay.com/w/2023/06/03/feed/
}

func getArticles(feedName, feedUrl string, upperBound *time.Time) (articles []*article) {
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	feed, err := p.ParseURLWithContext(feedUrl, ctx)
	if err != nil {
		fmt.Printf("err: cannot parse feed %q: %s\n", feedUrl, err)
		return
	}

	for _, i := range feed.Items {
		if !skip(i.Title) && i.PublishedParsed.After(*upperBound) {
			articles = append(articles, &article{
				// override the RSS/Atom title with the UDF title in the OPML,
				// this can help with merging togheter feeds of authors blogging from different sources
				BlogName:  feedName,
				Title:     i.Title,
				Url:       i.Link,
				Published: i.PublishedParsed,
			})
		}
	}
	if len(articles) > 3 {
		articles = articles[:3]
	}
	return
}
