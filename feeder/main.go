package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
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
		blogrollUrl  = os.Args[1]
		feedYAMLPath = os.Args[2]
	)

	f, err := opml.NewOPMLFromURL(blogrollUrl)
	if err != nil {
		panic(err)
	}

	articlesByCategory := make(map[string][]*article)

	for _, o := range f.Outlines() {
		if len(o.Outlines) == 0 {
			fmt.Println("processing feed:", o.Text)
			articlesByCategory["misc"] = append(articlesByCategory["misc"], getArticles(o.XMLURL)...)
		} else {
			for _, child := range o.Outlines {
				fmt.Println("processing feed:", child.Text)
				articlesByCategory[o.Text] = append(articlesByCategory[o.Text], getArticles(child.XMLURL)...)
			}
		}
	}

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
}

var p = gofeed.NewParser()

func getArticles(feedUrl string) (articles []*article) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	feed, err := p.ParseURLWithContext(feedUrl, ctx)
	if err != nil {
		fmt.Printf("err: cannot parse feed %q: %s\n", feedUrl, err)
		return
	}

	for _, i := range feed.Items {
		if !skip(i.Title) {
			articles = append(articles, &article{
				BlogName:  feed.Title,
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
