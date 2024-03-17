package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Stack[T any] struct {
	count int
	data  []T
	mu    sync.Mutex
}

func (s *Stack[T]) Push(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *Stack[T]) Empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count == 0
}

func (s *Stack[T]) Size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

func (s *Stack[T]) TryPop(v *T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.count != 0 {
		*v = s.data[s.count-1]
		s.count--
		return true
	}
	return false
}

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

func traverseHtmlParseTree(n *html.Node, response *http.Response) []string {
	nodeStack := Stack[*html.Node]{}
	nodeStack.Push(n)

	urls := []string{}
	for !nodeStack.Empty() {
		var node *html.Node
		if nodeStack.TryPop(&node) { // Just use pop?
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

type UrlInfo struct {
	url   string
	depth int
}

func traverseURL_BFS_StackBased(url string, depthLevel int) {
	urlStack := Stack[UrlInfo]{}
	urlStack.Push(UrlInfo{url, 0})

	var totalUrls uint32
	totalUrls++

	start := time.Now()
	for !urlStack.Empty() {
		var z UrlInfo
		if urlStack.TryPop(&z) {
			if z.depth < depthLevel {
				start := time.Now()
				response, err := http.Get(z.url)
				took := time.Since(start)

				if err != nil {
					continue
				}

				if response.StatusCode != http.StatusOK {
					response.Body.Close()
					continue
				}

				root, err := html.Parse(response.Body)
				if err != nil {
					response.Body.Close()
					continue
				}

				response.Body.Close()

				fmt.Printf("StackSize: %d, took: %v, url: %s[%d]\n", urlStack.Size(), took, z.url, z.depth)
				for _, url := range traverseHtmlParseTree(root, response) {
					totalUrls++
					urlStack.Push(UrlInfo{url, z.depth + 1})
				}
			}
		}
	}
	fmt.Printf("Total time: %v\n", time.Since(start))
	fmt.Printf("Total urls: %d\n", totalUrls)
}

func traverseURL_BFS_Concurrent(url string, depth int) {
	urls := make(chan UrlInfo)
	go func() { urls <- UrlInfo{url, 0} }()

	allUrls := make(chan string)
	go func() { allUrls <- url }() // avoid deadlock

	var index uint32
	go func() {
		for url := range allUrls {
			fmt.Printf("%d: url: %s\n", atomic.LoadUint32(&index), url)
			atomic.AddUint32(&index, 1)
		}
	}()
	for info := range urls {
		if info.depth < depth {
			go func(info UrlInfo) {
				response, err := http.Get(info.url)
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
					allUrls <- url
					urls <- UrlInfo{url, info.depth + 1}
				}
			}(info)
		}
	}
}

func main() {
	var depth int
	flag.IntVar(&depth, "depth", 2, "Depth level for traversing URLs")

	traverseURL_BFS_StackBased("https://python.org", depth)
	// traverseURL_BFS_Concurrent("https://python.org", depth)
}
