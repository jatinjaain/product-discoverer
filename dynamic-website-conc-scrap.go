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

func extractLinks(htmlContent string, baseUrl string) []string {
	var links []string

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		return links
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := attr.Val

					if isUsefulUrl(link) {
						absoluteLink, err := toAbsoluteUrl(baseUrl, link)
						if err != nil {
							if !strings.Contains(err.Error(), "domain not matching") {
								fmt.Println("Error converting to absolute URL:", err)
							}
							continue
						}

						completeAbsoluteLink := absoluteLink
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
	// above is the URL for freen proxies, which is not working. can use paid proxies link to use this feature
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

func createContextWithProxy(proxyURL string) (context.Context, context.CancelFunc) {
	if proxyURL == "" {
		ctx, cancel := chromedp.NewContext(context.Background())
		return ctx, cancel
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer(proxyURL),
	)
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelBrowser := chromedp.NewContext(allocCtx)
	return ctx, cancelBrowser
}

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

func scrollToBottom(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight);`, nil),
	)
}

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
		currentHeight, err := getPageHeight(ctx)
		if err != nil {
			return err
		}

		err = scrollToBottom(ctx)
		if err != nil {
			return err
		}

		time.Sleep(delay)

		newHeight, err := getPageHeight(ctx)
		if err != nil {
			return err
		}

		if newHeight == currentHeight {
			retries++
		} else {
			fmt.Println("Successfuly scrolled ....")
			scrolls++
			retries = 0
		}

		if newHeight <= lastHeight {
			break
		}

		lastHeight = newHeight
	}

	return nil
}

func scrapeDynamicWebsiteConcurrent(url string) (string, map[string]bool) {
	requiredLinks := getLinksLimitForHeadlessBrowser()
	workers := getHeadlessBrowsingWorkerCount()

	baseDomain := extractDomain(url)

	queueSize := 2000
	queue := make(chan string, queueSize)
	results := make(chan string, 100)
	done := make(chan bool)
	queueClosed := false
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, workers)

	visited := map[string]bool{}
	visitedMu := sync.Mutex{}

	productLinks := map[string]bool{}
	productLinksMu := sync.Mutex{}

	queue <- url

	// Start a goroutine to monitor when required links have been found
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
				close(queue)
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

				semaphore <- struct{}{}

				delay := randomDelay(minDelay, maxDelay)

				var htmlContent string

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
						if strings.Contains(err.Error(), "net::ERR_TUNNEL_CONNECTION_FAILED") ||
							strings.Contains(err.Error(), "net::ERR_TIMED_OUT") ||
							strings.Contains(err.Error(), "net::ERR_ABORTED") ||
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
						time.Sleep(delay)
					} else {
						// Scrolling logic
						err = handleInfiniteScroll(ctx, 2, 5, delay)
						if err != nil {
							log.Printf("Failed to handle infinite scroll: %v\n", err)
						}
						break // Exit the retry loop if the page loaded successfully
					}
				}

				if htmlContent == "" {
					<-semaphore
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
				<-semaphore
				cancel()
				if len(queue) == 0 {
					break
				}
			}

			done <- true
			queueClosed = true
			fmt.Println("closing queue ....")
			close(queue)
		}()
	}

	// Wait for either required product links or all pages to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	<-done

	return baseDomain, productLinks
}
