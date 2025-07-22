package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

type VideoTest struct {
	name       string
	videoURL   string
	resolution string
}

func (t *VideoTest) Name() string {
	return t.name
}

func (t *VideoTest) Run(ctx context.Context) (*TestResult, error) {
	result := &TestResult{
		TestName:  t.Name(),
		StartTime: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	// Create HTML page with video element and monitoring
	htmlContent := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Video Test - %s</title>
			<style>
				body { margin: 0; padding: 20px; background: #000; }
				video { width: 100%%; max-width: 1920px; display: block; margin: 0 auto; }
				#stats { color: white; font-family: monospace; margin-top: 20px; }
			</style>
		</head>
		<body>
			<video id="video" controls autoplay muted></video>
			<div id="stats"></div>
			<script>
				const video = document.getElementById('video');
				const stats = document.getElementById('stats');
				
				let frameCount = 0;
				let droppedFrames = 0;
				let decodedFrames = 0;
				let lastCheck = performance.now();
				let playbackStarted = false;
				let errors = [];
				
				// Set video source
				video.src = '%s';
				video.play().catch(e => console.error('Autoplay failed:', e));
				
				// Monitor video quality
				function updateStats() {
					if (video.getVideoPlaybackQuality) {
						const quality = video.getVideoPlaybackQuality();
						droppedFrames = quality.droppedVideoFrames;
						decodedFrames = quality.totalVideoFrames;
					}
					
					const now = performance.now();
					const elapsed = (now - lastCheck) / 1000;
					
					stats.innerHTML = 
						'Duration: ' + video.duration.toFixed(2) + 's<br>' +
						'Current Time: ' + video.currentTime.toFixed(2) + 's<br>' +
						'Decoded Frames: ' + decodedFrames + '<br>' +
						'Dropped Frames: ' + droppedFrames + '<br>' +
						'Drop Rate: ' + (decodedFrames > 0 ? (droppedFrames/decodedFrames*100).toFixed(2) : 0) + '%%<br>' +
						'Errors: ' + errors.length;
					
					if (!video.ended && playbackStarted) {
						requestAnimationFrame(updateStats);
					}
				}
				
				video.addEventListener('loadeddata', () => {
					console.log('Video loaded');
				});
				
				video.addEventListener('playing', () => {
					playbackStarted = true;
					updateStats();
				});
				
				video.addEventListener('error', (e) => {
					const error = {
						type: 'video_error',
						message: video.error ? video.error.message : 'Unknown error',
						code: video.error ? video.error.code : -1
					};
					errors.push(error);
					console.error('Video error:', error);
				});
				
				video.addEventListener('stalled', () => {
					errors.push({type: 'stalled', time: video.currentTime});
				});
				
				video.addEventListener('waiting', () => {
					errors.push({type: 'waiting', time: video.currentTime});
				});
				
				// Expose results for extraction
				window.getVideoStats = () => {
					const quality = video.getVideoPlaybackQuality ? video.getVideoPlaybackQuality() : {};
					return {
						duration: video.duration,
						currentTime: video.currentTime,
						ended: video.ended,
						paused: video.paused,
						readyState: video.readyState,
						networkState: video.networkState,
						decodedFrames: quality.totalVideoFrames || 0,
						droppedFrames: quality.droppedVideoFrames || 0,
						corruptedFrames: quality.corruptedVideoFrames || 0,
						errors: errors,
						videoWidth: video.videoWidth,
						videoHeight: video.videoHeight
					};
				};
			</script>
		</body>
		</html>
	`, t.resolution, t.videoURL)

	var videoStats map[string]interface{}

	// Create a temporary HTML file
	tmpFile, err := os.CreateTemp("", "video-test-*.html")
	if err != nil {
		result.EndTime = time.Now()
		result.Success = false
		result.Error = err
		return result, err
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(htmlContent); err != nil {
		result.EndTime = time.Now()
		result.Success = false
		result.Error = err
		return result, err
	}
	tmpFile.Close()

	err = chromedp.Run(ctx,
		// Navigate to the temporary HTML file
		chromedp.Navigate("file://"+tmpFile.Name()),
		chromedp.WaitReady("body"),
		
		// Wait for video to start playing
		chromedp.WaitVisible("#video", chromedp.ByID),
		chromedp.Sleep(2*time.Second),
		
		// Play video for at least 30 seconds or until it ends
		chromedp.ActionFunc(func(ctx context.Context) error {
			timeout := time.After(30 * time.Second)
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			
			for {
				select {
				case <-timeout:
					return nil
				case <-ticker.C:
					var ended bool
					if err := chromedp.Evaluate(`document.getElementById('video').ended`, &ended).Do(ctx); err != nil {
						return err
					}
					if ended {
						return nil
					}
				}
			}
		}),
		
		// Get final stats
		chromedp.Evaluate(`window.getVideoStats()`, &videoStats),
	)

	result.EndTime = time.Now()

	if err != nil {
		result.Success = false
		result.Error = err
		return result, err
	}

	// Extract metrics from stats
	if videoStats != nil {
		result.Metrics["video_url"] = t.videoURL
		result.Metrics["resolution"] = t.resolution
		result.Metrics["duration"] = videoStats["duration"]
		result.Metrics["decoded_frames"] = videoStats["decodedFrames"]
		result.Metrics["dropped_frames"] = videoStats["droppedFrames"]
		result.Metrics["corrupted_frames"] = videoStats["corruptedFrames"]
		
		// Calculate drop rate
		decoded := getFloat64(videoStats["decodedFrames"])
		dropped := getFloat64(videoStats["droppedFrames"])
		if decoded > 0 {
			result.Metrics["drop_rate_percent"] = (dropped / decoded) * 100
		} else {
			result.Metrics["drop_rate_percent"] = 0.0
		}
		
		result.Metrics["video_width"] = videoStats["videoWidth"]
		result.Metrics["video_height"] = videoStats["videoHeight"]
		result.Metrics["errors"] = videoStats["errors"]
		
		// Check for any errors (ignore waiting events at start)
		if errors, ok := videoStats["errors"].([]interface{}); ok && len(errors) > 0 {
			// Filter out benign errors
			var criticalErrors []interface{}
			for _, err := range errors {
				if errMap, ok := err.(map[string]interface{}); ok {
					errType := errMap["type"]
					// Waiting at time 0 is normal for video loading
					if errType == "waiting" && getFloat64(errMap["time"]) == 0 {
						continue
					}
					// Video errors are critical
					if errType == "video_error" {
						criticalErrors = append(criticalErrors, err)
					}
				}
			}
			
			if len(criticalErrors) > 0 {
				result.Success = false
				result.Error = fmt.Errorf("video playback errors: %v", criticalErrors)
			} else {
				result.Success = true
			}
		} else {
			result.Success = true
		}
	} else {
		result.Success = false
		result.Error = fmt.Errorf("failed to get video stats")
	}

	return result, nil
}

func getFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}