package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/time/rate"
)

const (
	driveDownloadPrefix = "https://drive.google.com/uc?export=download&id="
	fileID              = "1TF4RPuj8iFwpa-lyhxG67V8NDlktmTGi"
	testFileID          = "1TF6vDa7bTU_v814UmCPJQkIKeDqR3DoH"
)

var errEssayUnavailable = errors.New("unable to retrieve essay from provided url")

type essayCollector struct {
	urlbank     []string
	rpm         int
	routines    int
	destination chan *html.Node
}

type worker struct {
	wordbank        map[string]struct{}
	wordOccurrences map[string]int
}

func newWorker(wb map[string]struct{}) *worker {
	return &worker{
		wordbank:        wb,
		wordOccurrences: make(map[string]int),
	}
}

type htmlParser struct {
	wordbank   map[string]struct{}
	routines   int
	htmlStream chan *html.Node
}

// urlBank parses the essays urls file
func urlBank() ([]string, error) {
	path := driveDownloadPrefix + fileID

	resp, err := http.Get(path)
	if err != nil {
		return nil, fmt.Errorf("unable to access raw url bank: %w", err)
	}
	defer resp.Body.Close()

	// Create a scanner to read the file line by line
	var urls []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		url := scanner.Text()
		urls = append(urls, url)
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to scan urls source file: %w", err)
	}

	return urls, nil
}

func (w *worker) extractArticleContent(doc *html.Node) {
	// Find the main content <div> with class "caas-body"
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "caas-body") {
					// Extract text content from paragraphs within this div
					w.extractParagraphs(n)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findContent(c)
		}
	}

	// Start finding content from the root node
	findContent(doc)
}

func (w *worker) extractParagraphs(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "p" {
		// Extract text content from this paragraph
		text := w.extractText(n)
		// Count occurrences of valid words
		w.filterAndCount(text)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.extractParagraphs(c)
	}
}

func (w *worker) extractText(n *html.Node) string {
	var text string
	if n.Type == html.TextNode {
		text = n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += w.extractText(c)
	}
	return text
}

func (w *worker) filterAndCount(s string) {
	for _, v := range strings.Fields(s) {
		v = strings.ToLower(v)
		if _, ok := w.wordbank[v]; !ok {
			continue
		}
		w.wordOccurrences[v]++
	}
}

func (ec *essayCollector) retrieveHTMLEssays(ctx context.Context, debug bool) {
	// push urls to channel, so they can be consumed by http client
	urlsChan := make(chan string, ec.routines*5)
	go func() {
		for _, v := range ec.urlbank {
			urlsChan <- v
		}
		close(urlsChan)
	}()

	wg := sync.WaitGroup{}
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(ec.rpm)), 1)

	for i := 0; i < ec.routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range urlsChan {
				// error is returned if the context is cancelled.
				if err := limiter.Wait(ctx); err != nil {
					return
				}
				if debug {
					log.Printf("[DEBUG] Requesting article %s from remote server\n", u)
				}
				h, err := urlToHTML(u)
				if err != nil {
					log.Printf("[WARNING] Skipping the following essay url due to an error: %s\n", err)
					continue
				}
				ec.destination <- h
			}
		}()
	}

	wg.Wait()
	close(ec.destination)
}

func urlToHTML(url string) (*html.Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errEssayUnavailable, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w. url: %s; response code: %s", errEssayUnavailable, url, resp.Status)
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read essay: %w", err)
	}
	return doc, nil
}

// dispatch workers to parse and count words in the htmls sent to them
func (p *htmlParser) parseAndCount() ([]map[string]int, error) {
	wg := sync.WaitGroup{}
	results := make([]map[string]int, p.routines)

	for i := 0; i < p.routines; i++ {
		wg.Add(1)
		go func(i int) {
			w := newWorker(p.wordbank)
			for h := range p.htmlStream {
				w.extractArticleContent(h)
			}
			results[i] = w.wordOccurrences
			wg.Done()
		}(i)
	}

	wg.Wait()
	return results, nil
}
