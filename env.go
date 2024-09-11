package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func getInputLinkHandlingWorkerCount() int {
	defaultWorkers := 2
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	workerCountEnv := os.Getenv("INPUT_LINK_HANDLING_WORKERS")

	if workerCountEnv != "" {
		workers, err := strconv.Atoi(workerCountEnv)
		if err == nil && workers > 0 {
			return workers
		}
	}

	return defaultWorkers
}

func getHeadlessBrowsingWorkerCount() int {
	defaultWorkers := 3
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	workerCountEnv := os.Getenv("HEADLESS_BROWSING_WORKERS")

	if workerCountEnv != "" {
		workers, err := strconv.Atoi(workerCountEnv)
		if err == nil && workers > 0 {
			return workers
		}
	}

	return defaultWorkers
}

func getLinksLimitForHeadlessBrowser() int {
	defaultWorkers := 200
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	workerCountEnv := os.Getenv("LINKS_LIMIT_FOR_HEADLESS_BROWSER")

	if workerCountEnv != "" {
		workers, err := strconv.Atoi(workerCountEnv)
		if err == nil && workers > 0 {
			return workers
		}
	}

	return defaultWorkers
}
