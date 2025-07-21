package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/chromedp/chromedp"
)

func main() {

	dir, err := os.MkdirTemp("", "chromedp-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", false), // Disable headless mode
		// chromedp.Flag("kiosk", true),
		chromedp.Flag("disable-gpu", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// also set up a custom logger
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Define the URL for the MotionMark benchmark
	benchmarkURL := "https://browserbench.org/MotionMark/"

	// Define variables to store the results
	var results string

	// Run the tasks
	err = chromedp.Run(taskCtx,
		// Navigate to the benchmark URL
		chromedp.Navigate(benchmarkURL),

		// Wait for the "Run Benchmark" button to be visible
		chromedp.WaitVisible(`#intro`),

		// Click the "Run Benchmark" button
		chromedp.Evaluate(`benchmarkController.startBenchmark()`, nil),

		// Extract the results text
		chromedp.WaitVisible(`#results`),
		chromedp.Text(`#results`, &results, chromedp.NodeVisible),
	)

	if err != nil {
		log.Fatalf("Failed to run benchmark: %v", err)
	}

	// Print the results to stdout
	fmt.Println("MotionMark Benchmark Results:")
	fmt.Println(results)
}
