package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CPUMonitor struct {
	samples  []CPUSample
	mu       sync.Mutex
	stop     chan bool
	wg       sync.WaitGroup
	interval time.Duration
}

func NewCPUMonitor() *CPUMonitor {
	return &CPUMonitor{
		interval: 500 * time.Millisecond,
		stop:     make(chan bool),
	}
}

func (m *CPUMonitor) Start() {
	m.wg.Add(1)
	go m.monitor()
}

func (m *CPUMonitor) Stop() {
	close(m.stop)
	m.wg.Wait()
}

func (m *CPUMonitor) GetSamples() []CPUSample {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	result := make([]CPUSample, len(m.samples))
	copy(result, m.samples)
	return result
}

func (m *CPUMonitor) monitor() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			usage, err := getCPUUsage()
			if err == nil {
				m.mu.Lock()
				m.samples = append(m.samples, CPUSample{
					Timestamp: time.Now(),
					Usage:     usage,
				})
				m.mu.Unlock()
			}
		}
	}
}

func getCPUUsage() (float64, error) {
	switch runtime.GOOS {
	case "darwin":
		return getCPUUsageDarwin()
	case "linux":
		return getCPUUsageLinux()
	case "windows":
		return getCPUUsageWindows()
	default:
		return 0, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func getCPUUsageDarwin() (float64, error) {
	cmd := exec.Command("ps", "-A", "-o", "%cpu")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var total float64
	lineCount := 0
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "%CPU" {
			continue
		}
		
		cpu, err := strconv.ParseFloat(line, 64)
		if err == nil {
			total += cpu
			lineCount++
		}
	}
	
	// Darwin ps shows per-core percentages, so we need to divide by core count
	numCPU := runtime.NumCPU()
	return total / float64(numCPU), nil
}

func getCPUUsageLinux() (float64, error) {
	cmd := exec.Command("sh", "-c", "top -bn1 | grep 'Cpu(s)' | awk '{print 100 - $8}'")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}
	
	return usage, nil
}

func getCPUUsageWindows() (float64, error) {
	cmd := exec.Command("wmic", "cpu", "get", "loadpercentage", "/value")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "LoadPercentage=") {
			value := strings.TrimPrefix(line, "LoadPercentage=")
			value = strings.TrimSpace(value)
			return strconv.ParseFloat(value, 64)
		}
	}
	
	return 0, fmt.Errorf("could not parse CPU usage")
}