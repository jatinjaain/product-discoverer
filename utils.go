package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func extractDomain(link string) string {
	parsedUrl, err := url.Parse(link)
	if err != nil {
		fmt.Println("Error parsing URL:", link)
		return ""
	}
	return parsedUrl.Host
}

func isProductUrl(url string) bool {
	productSubstrings := []string{
		"/products/",
		"/p/",
		"/product-detail",
		"/productpage",
		"/item/",
		"/t/",
		"/buy",
		"/product/"}
	matchesAny := false

	for _, subs := range productSubstrings {
		if strings.Contains(url, subs) {
			matchesAny = true
			break
		}
	}

	return matchesAny
}

func isImageUrl(url string) bool {
	imageSubstrings := []string{
		".jpg",
		".jpeg",
		".png",
		".webp",
		"/cdn/",
		"cdn.",
		"assets.",
		"/image/",
		"asset.",
		"image.",
		"/static"}

	isImageUrl := false
	for _, subs := range imageSubstrings {
		if strings.Contains(url, subs) {
			isImageUrl = true
			break
		}
	}

	return isImageUrl
}

func isUsefulUrl(url string) bool {
	isUsefulUrl := true
	// do not visit if not slash, only hash eg. #MainContent
	if strings.Contains(url, "#") && !strings.Contains(url, "/") {
		isUsefulUrl = false
	}

	if isUsefulUrl {
		isUsefulUrl = !isImageUrl(url)
	}

	notUsefulPages := []string{
		"/account",
		"/login",
		"/contact-us",
		"/contactus",
		"/cart",
		"/search",
		"/faq",
		"/faqs",
		"/about-us",
		"/terms-of-use",
		"/t-cs",
		"/tac",
		"/privacy-policy",
		"/privacypolicy",
		"/returns-exchange-policy",
		"/news",
		"/wishlist"}

	if isUsefulUrl {
		for _, subs := range notUsefulPages {
			if strings.Contains(url, subs) {
				isUsefulUrl = false
			}
		}
	}

	return isUsefulUrl
}

func toAbsoluteUrl(baseUrl string, href string) (string, error) {
	base, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}
	relative, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	if relative.Hostname() == "" {
		return base.String() + relative.String(), nil
	} else if relative.Hostname() == base.String() {
		return relative.String(), nil
	}

	return "", errors.New("domain not matching for : " + relative.String())
}
