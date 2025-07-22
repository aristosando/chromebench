package main

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

type MotionMarkTest struct{}

func (t *MotionMarkTest) Name() string {
	return "motionmark"
}

func (t *MotionMarkTest) Run(ctx context.Context) (*TestResult, error) {
	result := &TestResult{
		TestName:  t.Name(),
		StartTime: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	var score float64
	var subscores map[string]interface{}

	err := chromedp.Run(ctx,
		// Navigate to the benchmark URL
		chromedp.Navigate("https://browserbench.org/MotionMark/"),

		// Wait for the "Run Benchmark" button to be visible
		chromedp.WaitVisible(`#intro`),

		// Click "Run Benchmark" button
		chromedp.Evaluate(`benchmarkController.startBenchmark()`, nil),

		// Wait for test to complete (this can take several minutes)
		chromedp.WaitVisible(`#results`, chromedp.ByID),

		// Extract overall score
		chromedp.Evaluate(`
			(() => {
				const scoreElement = document.querySelector('.score-text');
				if (scoreElement) {
					return parseFloat(scoreElement.textContent);
				}
				// Fallback to looking for score in results
				const resultsElement = document.querySelector('#results');
				if (resultsElement) {
					const text = resultsElement.textContent;
					const match = text.match(/Score:\s*(\d+\.?\d*)/);
					if (match) return parseFloat(match[1]);
				}
				return 0;
			})()
		`, &score),

		// Extract subscores
		chromedp.Evaluate(`
			(() => {
				const scores = {};
				const rows = document.querySelectorAll('tr.test-row');
				rows.forEach(row => {
					const name = row.querySelector('.test-name');
					const score = row.querySelector('.score');
					if (name && score) {
						scores[name.textContent.trim()] = parseFloat(score.textContent);
					}
				});
				
				// Also check for any detailed results
				const detailRows = document.querySelectorAll('.detailed-results tr');
				detailRows.forEach(row => {
					const cells = row.querySelectorAll('td');
					if (cells.length >= 2) {
						const name = cells[0].textContent.trim();
						const value = cells[cells.length - 1].textContent.trim();
						const numValue = parseFloat(value);
						if (!isNaN(numValue) && name) {
							scores[name] = numValue;
						}
					}
				});
				
				return scores;
			})()
		`, &subscores),
	)

	result.EndTime = time.Now()

	if err != nil {
		result.Success = false
		result.Error = err
		return result, err
	}

	result.Success = true
	result.Metrics["overall_score"] = score

	// Add subscores to metrics
	for key, value := range subscores {
		result.Metrics[fmt.Sprintf("subscore_%s", key)] = value
	}

	return result, nil
}
