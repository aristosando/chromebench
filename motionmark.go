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

	var scoreStr, confidenceStr string
	var subtestNames, subtestScores, subtestConfidences []string

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://browserbench.org/MotionMark/"),
		chromedp.WaitVisible(`#intro`),
		chromedp.Evaluate(`benchmarkController.startBenchmark()`, nil),
		chromedp.WaitVisible(`#results`, chromedp.ByID),

		// Extract overall score and confidence
		chromedp.Text(`#results .score-container .score`, &scoreStr, chromedp.NodeVisible),
		chromedp.Text(`#results .score-container .confidence`, &confidenceStr, chromedp.NodeVisible),

		// Extract subtest names, scores, and confidences (skip the first .suites-separator row)
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#results-header td')).slice(1).map(td => td.textContent.trim())`, &subtestNames),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#results-score td')).slice(1).map(td => td.textContent.trim())`, &subtestScores),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('#results-data td span')).slice(1).map(span => span.textContent.trim())`, &subtestConfidences),
	)

	// Parse overall score
	var score float64
	fmt.Sscanf(scoreStr, "%f", &score)

	// Parse subscores
	subscores := make(map[string]interface{})
	for i := 0; i < len(subtestNames) && i < len(subtestScores); i++ {
		var val float64
		fmt.Sscanf(subtestScores[i], "%f", &val)
		subscores[subtestNames[i]] = val
		// Optionally, also store confidence intervals
		if i < len(subtestConfidences) {
			subscores[subtestNames[i]+"_confidence"] = subtestConfidences[i]
		}
	}

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
