package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/systeminfo"
	"github.com/chromedp/chromedp"
)

func main() {

	dir, err := os.MkdirTemp("", "chromebench")
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

	// get gpu information
	var gpu *systeminfo.GPUInfo
	err = chromedp.Run(taskCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			c := chromedp.FromContext(ctx)
			browser := cdp.WithExecutor(ctx, c.Browser)
			gpuInfo, _, _, _, err := systeminfo.GetInfo().Do(browser)
			if err != nil {
				return err
			}
			gpu = gpuInfo
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nGPU Information:")
	if len(gpu.Devices) > 0 {
		fmt.Printf("  GPU Vendor: %s\n", gpu.Devices[0].VendorString)
		fmt.Printf("  GPU Device: %s\n", gpu.Devices[0].DeviceString)
		fmt.Printf("  GPU Driver Version: %s\n", gpu.Devices[0].DriverVersion)
	} else {
		fmt.Printf("  GPU devices not available\n")
	}
	fmt.Println("\nGPU Feature Status:")
	if gpu.FeatureStatus != nil {
		var featureStatus map[string]string
		err := json.Unmarshal(gpu.FeatureStatus, &featureStatus)
		if err == nil {
			for feature, status := range featureStatus {
				fmt.Printf("  %s: %s\n", feature, status)
			}
		} else {
			fmt.Printf("  Error parsing gpu feature status: %v\n", err)
		}
	}

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
