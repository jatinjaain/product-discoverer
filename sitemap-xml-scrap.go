package main

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// SitemapIndex represents the structure of a sitemap index file
type SitemapIndex struct {
	Sitemaps []Sitemap `xml:"sitemap"`
}

// Sitemap represents an individual sitemap in a sitemap index
type Sitemap struct {
	Loc string `xml:"loc"`
}

// URLSet represents a set of URLs in a sitemap file
type URLSet struct {
	URLs []URL `xml:"url"`
}

// URL represents an individual URL entry in a sitemap file
type URL struct {
	Loc string `xml:"loc"`
}

var client = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,  // Ensure no persistent connections
		ForceAttemptHTTP2: false, // Disable HTTP/2
	},
}

func fetchSitemapURL(website string) (string, error) {
	robotsURL := website + "/robots.txt"

	req, err := http.NewRequest("GET", robotsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not fetch robots.txt: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read robots.txt: %v", err)
	}

	body := string(bodyBytes)
	sitemapRegex := regexp.MustCompile(`(?i)sitemap:\s*(\S+)`)
	matches := sitemapRegex.FindStringSubmatch(body)

	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("no sitemap found in robots.txt")
}

func fetchProductURLs(sitemapURL string) (map[string]bool, error) {
	req, err := http.NewRequest("GET", sitemapURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch sitemap: %v", err)
	}
	defer resp.Body.Close()

	var bodyBytes []byte

	if strings.HasSuffix(sitemapURL, ".xml.gz") {
		fmt.Println("decompressing...")
		bodyBytes, err = decompressGzip(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("could not decompress sitemap: %v", err)
		}
	} else {
		bodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read sitemap: %v", err)
		}
	}

	if strings.Contains(string(bodyBytes), "<sitemapindex") {
		return processSitemapIndex(bodyBytes)
	} else if strings.Contains(string(bodyBytes), "<urlset") {
		return processURLSet(bodyBytes)
	}

	return nil, fmt.Errorf("unsupported sitemap format")
}

func decompressGzip(reader io.Reader) ([]byte, error) {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not create gzip reader: %v", err)
	}
	defer gzReader.Close()

	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, gzReader)
	if err != nil {
		return nil, fmt.Errorf("could not decompress gzip data: %v", err)
	}

	return buffer.Bytes(), nil
}

func processSitemapIndex(body []byte) (map[string]bool, error) {
	var sitemapIndex SitemapIndex
	err := xml.Unmarshal(body, &sitemapIndex)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal sitemap index: %v", err)
	}

	productURLs := make(map[string]bool)

	for _, sitemap := range sitemapIndex.Sitemaps {
		fmt.Println("sitemap.Loc s: ", sitemap.Loc)
		if strings.Contains(sitemap.Loc, "product") || strings.Contains(sitemap.Loc, "products") {
			urls, err := fetchProductURLs(sitemap.Loc)
			if err == nil {
				for url := range urls {
					productURLs[url] = true
				}
			}
		}
	}

	// If no sitemap containing "product" or "products" is found, try all sitemaps
	if len(productURLs) == 0 {
		for _, sitemap := range sitemapIndex.Sitemaps {
			urls, err := fetchProductURLs(sitemap.Loc)
			if err == nil {
				for url := range urls {
					productURLs[url] = true
				}
			}
		}
	}

	return productURLs, nil
}

func processURLSet(body []byte) (map[string]bool, error) {
	var urlSet URLSet
	err := xml.Unmarshal(body, &urlSet)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal URL set: %v", err)
	}

	productURLs := make(map[string]bool)

	for _, url := range urlSet.URLs {
		if isProductUrl(url.Loc) {
			if len(productURLs)%10000 == 0 {
				fmt.Println("Product urls length", len(productURLs))
			}
			productURLs[url.Loc] = true
		}
	}

	return productURLs, nil
}

func outputProductUrlsUsingSitemap(website string) (string, map[string]bool) {
	baseDomain := extractDomain(website)
	// Step 1: Fetch Sitemap URL from robots.txt
	sitemapURL, err := fetchSitemapURL("https://" + baseDomain)
	if err != nil {
		fmt.Println(err)
		return baseDomain, make(map[string]bool)
	}
	fmt.Printf("Sitemap URL: %s\n", sitemapURL)

	// Step 2: Fetch and process sitemap for product URLs
	productLinks, err := fetchProductURLs(sitemapURL)
	if err != nil {
		fmt.Println(err)
		return baseDomain, make(map[string]bool)
	}

	return baseDomain, productLinks
}
