package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type VideoCache struct {
	cacheDir string
}

type VideoInfo struct {
	Name       string
	URL        string
	LocalPath  string
	Resolution string
	Size       int64
}

var testVideos = []VideoInfo{
	{
		Name:       "video-640x360-h264",
		URL:        "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4",
		Resolution: "640x360",
	},
	{
		Name:       "video-1080p-h264",
		URL:        "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/1080/Big_Buck_Bunny_1080_10s_10MB.mp4",
		Resolution: "1920x1080",
	},
	{
		Name:       "video-4k-h264",
		URL:        "https://sample-videos.com/video321/mp4/720/big_buck_bunny_720p_10mb.mp4",
		Resolution: "3840x2160",
	},
}

func NewVideoCache() (*VideoCache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	cacheDir := filepath.Join(homeDir, ".chromebench", "videos")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}
	
	return &VideoCache{cacheDir: cacheDir}, nil
}

func (vc *VideoCache) GetVideoPath(videoInfo VideoInfo) string {
	// Extract filename from URL
	parts := strings.Split(videoInfo.URL, "/")
	filename := parts[len(parts)-1]
	return filepath.Join(vc.cacheDir, filename)
}

func (vc *VideoCache) IsVideoCached(videoInfo VideoInfo) bool {
	localPath := vc.GetVideoPath(videoInfo)
	info, err := os.Stat(localPath)
	return err == nil && info.Size() > 0
}

func (vc *VideoCache) DownloadVideo(videoInfo VideoInfo) error {
	localPath := vc.GetVideoPath(videoInfo)
	
	// Check if already exists
	if vc.IsVideoCached(videoInfo) {
		fmt.Printf("Video %s already cached at %s\n", videoInfo.Name, localPath)
		return nil
	}
	
	fmt.Printf("Downloading %s from %s...\n", videoInfo.Name, videoInfo.URL)
	
	// Create temporary file
	tmpPath := localPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()
	
	// Download the file
	resp, err := http.Get(videoInfo.URL)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		os.Remove(tmpPath)
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	
	// Get the size
	size := resp.ContentLength
	
	// Create progress reader
	pr := &progressReader{
		Reader: resp.Body,
		Total:  size,
		Name:   videoInfo.Name,
	}
	
	// Copy the file
	_, err = io.Copy(out, pr)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}
	
	// Close the file
	out.Close()
	
	// Rename to final path
	if err := os.Rename(tmpPath, localPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	
	fmt.Printf("\nDownloaded %s successfully\n", videoInfo.Name)
	return nil
}

func (vc *VideoCache) EnsureAllVideos() error {
	fmt.Println("Checking video cache...")
	
	for _, video := range testVideos {
		if err := vc.DownloadVideo(video); err != nil {
			return fmt.Errorf("failed to download %s: %v", video.Name, err)
		}
	}
	
	fmt.Println("All videos cached successfully")
	return nil
}

type progressReader struct {
	io.Reader
	Total      int64
	Downloaded int64
	Name       string
	lastPrint  int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Downloaded += int64(n)
	
	// Print progress every 1MB
	if pr.Downloaded-pr.lastPrint > 1024*1024 || pr.Downloaded == pr.Total || err == io.EOF {
		pr.lastPrint = pr.Downloaded
		if pr.Total > 0 {
			percent := float64(pr.Downloaded) / float64(pr.Total) * 100
			fmt.Printf("\r%s: %.1f%% (%.1f MB / %.1f MB)", 
				pr.Name,
				percent,
				float64(pr.Downloaded)/(1024*1024),
				float64(pr.Total)/(1024*1024))
		}
	}
	
	return n, err
}