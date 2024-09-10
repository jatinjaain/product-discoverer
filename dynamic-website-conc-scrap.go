package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

// Helper function to extract links from HTML content
func extractLinks(htmlContent string, baseUrl string) []string {
	var links []string

	// Parse the HTML content
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		return links
	}

	// Traverse the HTML nodes to find <a href=""> tags
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			// Find the href attribute
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := attr.Val

					// Check if the link is useful
					if isUsefulUrl(link) {
						// Convert relative URL to absolute URL
						absoluteLink, err := toAbsoluteUrl(baseUrl, link)
						if err != nil {
							if !strings.Contains(err.Error(), "domain not matching") {
								fmt.Println("Error converting to absolute URL:", err)
							}
							continue
						}

						completeAbsoluteLink := absoluteLink
						// if absolute link does not contain prefix https then add it
						if !strings.HasPrefix(absoluteLink, "http") {
							completeAbsoluteLink = "https://" + completeAbsoluteLink
						}
						links = append(links, completeAbsoluteLink)
					}
				}
			}
		}

		// Traverse the child nodes recursively
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	// Start the traversal from the root node
	traverse(doc)

	return links
}

type ProxyResponse struct {
	Proxies []struct {
		Proxy string `json:"proxy"`
	} `json:"proxies"`
}

// Function to fetch proxies from the API
func fetchProxies() ([]string, error) {
	// url := "https://api.proxyscrape.com/v3/free-proxy-list/get?request=displayproxies&protocol=http&proxy_format=protocolipport&format=json&timeout=400"
	url := ""
	if url == "" {
		return make([]string, 0, 5), nil
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var proxyResp ProxyResponse
	err = json.Unmarshal(body, &proxyResp)
	if err != nil {
		return nil, err
	}

	// Fetch maximum of 5 proxies
	proxies := make([]string, 0, 5)
	for i, proxy := range proxyResp.Proxies {
		if i >= 5 {
			break
		}
		proxies = append(proxies, proxy.Proxy)
	}

	return proxies, nil
}

// Function to create a Chrome context with a proxy
func createContextWithProxy(proxyURL string) (context.Context, context.CancelFunc) {
	if proxyURL == "" {
		ctx, cancel := chromedp.NewContext(context.Background())
		return ctx, cancel
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer(proxyURL), // Set the proxy server here
	)
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelBrowser := chromedp.NewContext(allocCtx)
	return ctx, cancelBrowser
}

// Function to get a random proxy from the list
func getRandomProxy(proxies []string) string {
	if len(proxies) == 0 {
		return ""
	}
	randIndex, _ := cryptoRandInt(int64(len(proxies)))
	return proxies[randIndex]
}

func cryptoRandInt(max int64) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

// Function to generate a random delay between min and max seconds
func randomDelay(min, max int64) time.Duration {
	delay, _ := cryptoRandInt(max - min + 1)
	return time.Duration(min+int64(delay)) * time.Second
}

// ProxyData represents a single proxy entry from the API
type ProxyData struct {
	IP        string   `json:"ip"`
	Port      string   `json:"port"`
	Protocols []string `json:"protocols"`
}

// ProxyResponse represents the response structure from the API
type ProxyResponse2 struct {
	Data []ProxyData `json:"data"`
}

// Function to scroll to the bottom of the page
func scrollToBottom(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight);`, nil),
	)
}

// Function to get the current page height
func getPageHeight(ctx context.Context) (int, error) {
	var height int
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.body.scrollHeight`, &height),
	)
	return height, err
}

// Function to handle infinite scroll and wait for new content to load
func handleInfiniteScroll(ctx context.Context, maxRetries int, maxScrolls int, delay time.Duration) error {
	retries := 0
	scrolls := 0
	lastHeight := 0

	for retries < maxRetries && scrolls < maxScrolls {
		// Get current page height
		currentHeight, err := getPageHeight(ctx)
		if err != nil {
			return err
		}

		// Scroll to the bottom of the page
		err = scrollToBottom(ctx)
		if err != nil {
			return err
		}

		// Wait for the new content to load
		time.Sleep(delay)

		// Get the new page height
		newHeight, err := getPageHeight(ctx)
		if err != nil {
			return err
		}

		// If the height hasn't changed, increment retries
		if newHeight == currentHeight {
			retries++
		} else {
			// Reset retries if new content is loaded
			fmt.Println("Successfuly scrolled ....")
			scrolls++
			retries = 0
		}

		// Break if the page height stopped changing
		if newHeight <= lastHeight {
			break
		}

		lastHeight = newHeight
	}

	return nil
}

func scrapeDynamicWebsiteConcurrent(url string) (string, map[string]bool) {
	requiredLinks := getLinksLimitForHeadlessBrowser()
	workers := getHeadlessBrowsingWorkerCount() // Number of concurrent workers

	// Extract the domain of the starting URL
	baseDomain := extractDomain(url)

	queueSize := 2000
	queue := make(chan string, queueSize)
	results := make(chan string, 100)
	done := make(chan bool)
	queueClosed := false
	var wg sync.WaitGroup

	// Semaphore to control concurrency (3 concurrent pages)
	semaphore := make(chan struct{}, workers)

	// Track visited URLs
	visited := map[string]bool{}
	visitedMu := sync.Mutex{}

	// Track product links found
	productLinks := map[string]bool{}
	productLinksMu := sync.Mutex{}

	// Add the first URL to the queue
	queue <- url

	// Start a goroutine to monitor when 100 links have been found
	go func() {
		for productLink := range results {
			productLinksMu.Lock()
			productLinks[productLink] = true
			if len(productLinks)%100 == 0 {
				fmt.Println(len(productLinks), " Product Links fetched for ", baseDomain)
			}
			if len(productLinks) >= requiredLinks {
				done <- true
				queueClosed = true
				fmt.Println("closing queue ....")
				close(queue) // Stop further processing
			}
			productLinksMu.Unlock()
		}
	}()

	// Fetch proxies from the API
	proxies, err := fetchProxies()
	if err != nil {
		log.Fatal("Error fetching proxies:", err)
	}
	fmt.Println("Fetched proxies:", proxies)

	minDelay := int64(1) // Minimum delay in seconds
	maxDelay := int64(3) // Maximum delay in seconds

	// Worker goroutines to process the queue concurrently
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if queueClosed {
				return
			}
			for currentURL := range queue {

				semaphore <- struct{}{} // Acquire a token to start work

				delay := randomDelay(minDelay, maxDelay)

				var htmlContent string

				// Get a random proxy
				proxyURL := getRandomProxy(proxies)
				fmt.Printf("Using proxy: %s\n", proxyURL)

				// Create a Chrome context with the proxy
				ctx, cancel := createContextWithProxy(proxyURL)
				defer cancel()

				for retries := 0; retries < 3; retries++ {
					time.Sleep(delay)
					err := chromedp.Run(ctx,
						chromedp.Navigate(currentURL),
						chromedp.WaitReady("body", chromedp.ByQuery),
						chromedp.Sleep(5*time.Second),
						chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery),
					)
					if err != nil {
						if retries == 2 { // After 3 retries, skip the URL
							log.Printf("Failed to load %s after retries : %v\n", currentURL, err)
							htmlContent = ""
							break
						}
						if strings.Contains(err.Error(), "net::ERR_ABORTED") {
							time.Sleep(5 * time.Second) // Sleep before retrying
							log.Printf("Retrying %s due to error : %v\n", currentURL, err)
							continue
						}
						if strings.Contains(err.Error(), "net::ERR_TUNNEL_CONNECTION_FAILED") ||
							strings.Contains(err.Error(), "net::ERR_TIMED_OUT") ||
							strings.Contains(err.Error(), "net::ERR_PROXY_CONNECTION_FAILED") ||
							strings.Contains(err.Error(), "net::ERR_EMPTY_RESPONSE") {
							proxyURL = getRandomProxy(proxies)
							fmt.Printf("Using proxy: %s\n", proxyURL)
							cancel()
							ctx, cancel = createContextWithProxy(proxyURL)
							defer cancel()
							log.Printf("Retrying %s due to error : %v\n", currentURL, err)
							continue
						}
						log.Printf("Retrying %s due to error : %v\n", currentURL, err)
						time.Sleep(delay) // Sleep before retrying
					} else {
						// Infinite scrolling logic
						err = handleInfiniteScroll(ctx, 2, 5, 3*time.Second)
						if err != nil {
							log.Printf("Failed to handle infinite scroll: %v\n", err)
						}
						break // Exit the retry loop if the page loaded successfully
					}
				}

				if htmlContent == "" {
					<-semaphore // Release token
					cancel()
					if len(queue) == 0 {
						break
					}
					continue
				}

				fmt.Println("Finally VISITING : ", currentURL)
				// Extract links from the page
				links := extractLinks(htmlContent, baseDomain)
				for _, link := range links {
					productLinksMu.Lock()
					if len(productLinks) >= requiredLinks {
						productLinksMu.Unlock()
						return
					}
					productLinksMu.Unlock()

					visitedMu.Lock()
					if visited[link] {
						visitedMu.Unlock()
						continue
					}
					visited[link] = true
					visitedMu.Unlock()

					if isProductUrl(link) {
						productLinksMu.Lock()
						if len(productLinks) < requiredLinks {
							results <- link
						}
						// fmt.Println("Adding to product : ", link, " results length: ", len(results))
						productLinksMu.Unlock()
					}

					if len(queue) < queueSize {
						if len(queue) == 0 {
							queue <- link
							// fmt.Println("Adding to queue : ", link, " queue length: ", len(queue))
						} else {
							if !queueClosed {
								queue <- link
								// fmt.Println("Adding to queue : ", link, " queue length: ", len(queue))
							}
						}
					}

				}
				<-semaphore // Release token
				cancel()
				if len(queue) == 0 {
					break
				}
			}

			// queue becomes empty
			done <- true
			queueClosed = true
			fmt.Println("closing queue ....")
			close(queue) // Stop further processing
		}()
	}

	// Wait for either 100 product links or all pages to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Wait for 100 links or the processing to complete
	<-done

	return baseDomain, productLinks
}
