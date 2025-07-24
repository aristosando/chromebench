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

type ChromeCPUMonitor struct {
	samples  []CPUSample
	mu       sync.Mutex
	stop     chan bool
	wg       sync.WaitGroup
	interval time.Duration
}

func NewChromeCPUMonitor() *ChromeCPUMonitor {
	return &ChromeCPUMonitor{
		interval: 500 * time.Millisecond,
		stop:     make(chan bool),
	}
}

func (m *ChromeCPUMonitor) Start() {
	m.wg.Add(1)
	go m.monitor()
}

func (m *ChromeCPUMonitor) Stop() {
	close(m.stop)
	m.wg.Wait()
}

func (m *ChromeCPUMonitor) GetSamples() []CPUSample {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	result := make([]CPUSample, len(m.samples))
	copy(result, m.samples)
	return result
}

func (m *ChromeCPUMonitor) monitor() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			usage, err := m.getChromeCPUUsage()
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

func (m *ChromeCPUMonitor) getChromeCPUUsage() (float64, error) {
	switch runtime.GOOS {
	case "darwin":
		return m.getChromeCPUUsageDarwin()
	case "linux":
		return m.getChromeCPUUsageLinux()
	case "windows":
		return m.getChromeCPUUsageWindows()
	default:
		return 0, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (m *ChromeCPUMonitor) getChromeCPUUsageDarwin() (float64, error) {
	// Get all Chrome-related processes
	cmd := exec.Command("ps", "-A", "-o", "pid,ppid,%cpu,command")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var totalCPU float64
	chromeProcesses := make(map[int]bool)
	
	// First pass: identify Chrome processes
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	// Find main Chrome process and its children
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			pid, _ := strconv.Atoi(fields[0])
			ppid, _ := strconv.Atoi(fields[1])
			
			// Check if this is a Chrome process
			cmdLine := strings.Join(fields[3:], " ")
			if strings.Contains(cmdLine, "Chromium") || strings.Contains(cmdLine, "Google Chrome") ||
			   strings.Contains(cmdLine, "chrome") {
				chromeProcesses[pid] = true
			}
			
			// If parent is Chrome, this is also Chrome
			if chromeProcesses[ppid] {
				chromeProcesses[pid] = true
			}
		}
	}
	
	// Second pass: sum up CPU usage
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, _ := strconv.Atoi(fields[0])
			if chromeProcesses[pid] {
				cpu, err := strconv.ParseFloat(fields[2], 64)
				if err == nil {
					totalCPU += cpu
				}
			}
		}
	}
	
	return totalCPU, nil
}

func (m *ChromeCPUMonitor) getChromeCPUUsageLinux() (float64, error) {
	// Use top in batch mode to get accurate CPU usage for Chrome processes
	// The -b flag runs in batch mode, -n 1 means one iteration
	cmd := exec.Command("sh", "-c", "top -b -n 1 | grep -E '(chrome|chromium)' | grep -v grep | awk '{sum += $9} END {print sum}'")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative with ps
		cmd = exec.Command("sh", "-c", "ps aux | grep -E '(chrome|chromium)' | grep -v grep | awk '{sum += $3} END {print sum}'")
		output, err = cmd.Output()
		if err != nil {
			return 0, err
		}
	}
	
	totalCPU, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}
	
	// The value from ps/top is already the total across all Chrome processes
	// It represents the percentage of total CPU capacity
	return totalCPU, nil
}


func (m *ChromeCPUMonitor) getChromeCPUUsageWindows() (float64, error) {
	// Use wmic to get Chrome process CPU usage
	cmd := exec.Command("wmic", "process", "where", "name like '%chrome%'", "get", "ProcessId,PercentProcessorTime", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var totalCPU float64
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ",") {
			fields := strings.Split(line, ",")
			if len(fields) >= 3 {
				cpu, err := strconv.ParseFloat(fields[2], 64)
				if err == nil {
					totalCPU += cpu
				}
			}
		}
	}
	
	return totalCPU, nil
}