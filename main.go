package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

func outputProductUrls(inputLinks []string) {
	// Maximum number of workers (parallel tasks) to run at once
	maxWorkers := getInputLinkHandlingWorkerCount()
	sem := make(chan struct{}, maxWorkers) // Semaphore to limit concurrent workers

	var wg sync.WaitGroup

	// Iterate over the inputLinks
	for _, link := range inputLinks {
		// Add a worker
		wg.Add(1)
		sem <- struct{}{} // Acquire a worker slot

		// Run in parallel for each link
		go func(link string) {
			defer wg.Done()
			defer func() { <-sem }() // Release the worker slot

			fmt.Printf("Processing link: %s\n", link)

			// Try fetching products using sitemap first
			baseDomain, productLinks := outputProductUrlsUsingSitemap(link)

			// Check if we got any product links from sitemap
			if len(productLinks) == 0 {
				fmt.Printf("No product links found in Sitemap for %s, attempting dynamic scrape...\n", baseDomain)

				// If sitemap didn't provide product links, use dynamic scraping
				_, productLinks = scrapeDynamicWebsiteConcurrent(link)
			}

			// Save product links to a file
			if len(productLinks) > 0 {
				keys := make([]string, 0, len(productLinks))
				for productLink := range productLinks {
					keys = append(keys, productLink)
				}

				combinedString := strings.Join(keys, "\n")
				filename := "./" + baseDomain + ".txt"
				err := os.WriteFile(filename, []byte(combinedString), 0644)
				if err != nil {
					fmt.Println("Error writing to file:", err)
				} else {
					fmt.Printf("Product links saved to %s\n", filename)
				}
			} else {
				fmt.Printf("No product links found for %s.\n", baseDomain)
			}

		}(link) // Pass the link to the goroutine
	}

	// Wait for all workers to finish
	wg.Wait()
	fmt.Println("Finished processing all links.")
}

func main() {
	fmt.Println("in main")

	start := time.Now()
	inputLinks := []string{
		// "https://www.myntra.com/",
		"https://littleboxindia.com/",
		"https://www.thesouledstore.com",
		"https://www.snitch.co.in/",

		"https://www.uniqlo.com/in/en/"}

	outputProductUrls(inputLinks)

	fmt.Println("time taken: ", time.Since(start))
}
