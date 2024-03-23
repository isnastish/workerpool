package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"time"
)

// A bare minimum stack implementation used for traversing html nodes iteratively.
type Stack[T any] struct {
	count int
	data  []T
}

// Push element of type T into the stack
func (s *Stack[T]) Push(v T) {
	if s.count == cap(s.data) {
		newCap := max(cap(s.data)<<1, 64)
		if cap(s.data) == 0 {
			s.data = make([]T, newCap)
		} else {
			newData := make([]T, newCap)
			copy(newData, s.data[:s.count])
			s.data = newData
		}
	}
	s.data[s.count] = v
	s.count++
}

// Check if the stack is empty
func (s *Stack[T]) Empty() bool {
	return s.count == 0
}

// Retrieve stack size
func (s *Stack[T]) Size() int {
	return s.count
}

// If stack is not empty, pops the last element and assignes it to v, returns true.
// false otherwise.
func (s *Stack[T]) TryPop(v *T) bool {
	if s.count != 0 {
		*v = s.data[s.count-1]
		s.count--
		return true
	}
	return false
}

// Accumulate all the URL's from the current HTML node.
func getURLs(n *html.Node, response *http.Response) []string {
	urls := []string{}
	for _, atrib := range n.Attr {
		if atrib.Key != "href" {
			continue
		}
		url, err := response.Request.URL.Parse(atrib.Val)
		if err != nil {
			continue
		}
		urls = append(urls, url.String())
	}
	return urls
}

// Traverses html nodes iteratively
func traverseHtmlParseTree(n *html.Node, response *http.Response) []string {
	nodeStack := Stack[*html.Node]{}
	nodeStack.Push(n)

	urls := []string{}
	for !nodeStack.Empty() {
		var node *html.Node
		if nodeStack.TryPop(&node) {
			if node.Type == html.ElementNode && node.Data == "a" {
				urls = append(urls, getURLs(node, response)...)
			}
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				nodeStack.Push(c)
			}
		}
	}
	return urls
}

// A bundle to hold URL name and its depth limit
type UrlInfo struct {
	url   string
	depth int
}

// Core function to traverse all URL's in breadth first search manner and print them to stdout.
func traverseURL_BFS_Concurrent(url string, depth int) {
	urls := make(chan UrlInfo)
	go func() { urls <- UrlInfo{url, 0} }()

	allUrls := make(chan string)
	go func() { allUrls <- url }()

	go func() {
		for url := range allUrls {
			fmt.Printf("url: %s\n", url)
		}
	}()

	p := NewPool()

Loop:
	for {
		select {
		case info := <-urls:
			z := info
			if z.depth < depth {
				p.SubmitTask(func() {
					response, err := http.Get(z.url)
					if err != nil {
						return
					}

					if response.StatusCode != http.StatusOK {
						response.Body.Close()
						return
					}

					root, err := html.Parse(response.Body)
					if err != nil {
						response.Body.Close()
						return
					}

					response.Body.Close()
					for _, url := range traverseHtmlParseTree(root, response) {
						urls <- UrlInfo{url, z.depth + 1}
						allUrls <- url
					}
				})
			}
		case <-time.After(1000 * time.Millisecond):
			break Loop
		}
	}
	p.Wait()
}

type Options struct {
	depth int
	url   string
}

func main() {
	o := Options{}

	flag.IntVar(&o.depth, "depth", 2, "Depth level for traversing URLs")
	flag.StringVar(&o.url, "url", "https://python.org", "URL to travers")

	flag.Parse()

	traverseURL_BFS_Concurrent(o.url, o.depth)
}
