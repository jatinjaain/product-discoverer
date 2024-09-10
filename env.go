package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Function to get the number of workers from the environment, or fall back to default.
func getInputLinkHandlingWorkerCount() int {
	defaultWorkers := 2
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	workerCountEnv := os.Getenv("INPUT_LINK_HANDLING_WORKERS") // Get the value from the environment variable

	if workerCountEnv != "" {
		workers, err := strconv.Atoi(workerCountEnv) // Convert to integer
		if err == nil && workers > 0 {
			return workers
		}
	}
	// Fallback to default if the environment variable is not set or invalid
	return defaultWorkers
}

func getHeadlessBrowsingWorkerCount() int {
	defaultWorkers := 3
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	workerCountEnv := os.Getenv("HEADLESS_BROWSING_WORKERS") // Get the value from the environment variable

	if workerCountEnv != "" {
		workers, err := strconv.Atoi(workerCountEnv) // Convert to integer
		if err == nil && workers > 0 {
			return workers
		}
	}
	// Fallback to default if the environment variable is not set or invalid
	return defaultWorkers
}

func getLinksLimitForHeadlessBrowser() int {
	defaultWorkers := 200
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	workerCountEnv := os.Getenv("LINKS_LIMIT_FOR_HEADLESS_BROWSER") // Get the value from the environment variable

	if workerCountEnv != "" {
		workers, err := strconv.Atoi(workerCountEnv) // Convert to integer
		if err == nil && workers > 0 {
			return workers
		}
	}
	// Fallback to default if the environment variable is not set or invalid
	return defaultWorkers
}
