# ChromeBench

A Chrome performance test harness for benchmarking graphics and video playback performance.

## Features

- **MotionMark Benchmark**: Runs the MotionMark 1.3 graphics benchmark
- **Video Playback Tests**: Tests video playback with frame drop detection at 24fps, 30fps, and 60fps for 240p, 720p, 1080p, and 2160p (4K)
- **CPU Monitoring**: Tracks CPU usage during all tests
- **Flexible Test Selection**: Include/exclude specific tests
- **Chrome Flag Support**: Pass custom Chrome flags for testing different configurations
- **Video Caching**: Automatically downloads and caches test videos locally to eliminate network variability

## Usage

### List available tests
```bash
go run . -list
```

### Download test videos
```bash
go run . -download-videos
```

Videos are cached in `~/.chromebench/videos/`

### Run all tests
```bash
go run .
```

### Run specific tests
```bash
go run . -include "motionmark,video-1080p-h264"
```

### Exclude tests
```bash
go run . -exclude "video-4k-h264"
```

### Run with custom Chrome flags
```bash
# Test with hardware video decode disabled
go run . -- --disable-accelerated-video-decode

# Test with multiple flags
go run . -- --disable-gpu-rasterization --disable-gpu

# Test with flags containing commas
go run . -- --enable-features=VaapiVideoEncoder,Vulkan --disable-features=UseChromeOSDirectVideoDecoder
```

### Run in headless mode
```bash
go run . -headless
```

## Example Output

```
GPU: ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Max, Version 15.5)

Running test: motionmark
Running test: video-1080p-h264

=== Test Summary ===

Test: motionmark
  Duration: 2m15s
  Success: true
  Average CPU Usage: 45.23%
  Metrics:
    overall_score: 1523.45
    subscore_Canvas: 1623.12
    subscore_WebGL: 1423.78

Test: video-1080p-h264  
  Duration: 30s
  Success: true
  Average CPU Usage: 12.45%
  Metrics:
    resolution: 1920x1080
    decoded_frames: 900
    dropped_frames: 2
    drop_rate_percent: 0.22%
```

## Chrome Flags for Video Testing

Common flags for testing video decode paths:
- `disable-accelerated-video-decode`: Force software video decode
- `enable-accelerated-video-decode`: Ensure hardware decode is enabled
- `use-angle=metal`: Force ANGLE Metal backend (macOS)
- `use-angle=d3d11`: Force ANGLE D3D11 backend (Windows)
- `disable-features=UseChromeOSDirectVideoDecoder`: Disable direct video decoder