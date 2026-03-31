package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkResult holds benchmark results
type BenchmarkResult struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalDuration      time.Duration
	RequestsPerSecond  float64
	AvgLatency         time.Duration
	MinLatency         time.Duration
	MaxLatency         time.Duration
	LatencyPercentiles map[string]time.Duration
}

// LoadTester performs load testing
type LoadTester struct {
	client      *http.Client
	url         string
	method      string
	concurrency int
	duration    time.Duration
}

// NewLoadTester creates a new load tester
func NewLoadTester(url, method string, concurrency int, duration time.Duration) *LoadTester {
	return &LoadTester{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        concurrency * 2,
				MaxIdleConnsPerHost: concurrency,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		url:         url,
		method:      method,
		concurrency: concurrency,
		duration:    duration,
	}
}

// Run executes the load test
func (lt *LoadTester) Run() *BenchmarkResult {
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64

	latencies := make([]time.Duration, 0, 100000)
	var latenciesMu sync.Mutex

	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	startTime := time.Now()

	// Start workers
	for i := 0; i < lt.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					reqStart := time.Now()
					req, _ := http.NewRequest(lt.method, lt.url, nil)
					resp, err := lt.client.Do(req)
					latency := time.Since(reqStart)

					atomic.AddInt64(&totalRequests, 1)

					if err != nil || resp.StatusCode >= 400 {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successfulRequests, 1)
					}

					if resp != nil {
						resp.Body.Close()
					}

					latenciesMu.Lock()
					latencies = append(latencies, latency)
					latenciesMu.Unlock()
				}
			}
		}()
	}

	// Run for specified duration
	time.Sleep(lt.duration)
	close(stopCh)

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Calculate results
	return lt.calculateResults(totalRequests, successfulRequests, failedRequests, totalDuration, latencies)
}

func (lt *LoadTester) calculateResults(total, successful, failed int64, duration time.Duration, latencies []time.Duration) *BenchmarkResult {
	rps := float64(total) / duration.Seconds()

	var totalLatency time.Duration
	var minLatency time.Duration
	var maxLatency time.Duration

	if len(latencies) > 0 {
		minLatency = latencies[0]
		maxLatency = latencies[0]

		for _, lat := range latencies {
			totalLatency += lat
			if lat < minLatency {
				minLatency = lat
			}
			if lat > maxLatency {
				maxLatency = lat
			}
		}
	}

	avgLatency := time.Duration(0)
	if len(latencies) > 0 {
		avgLatency = totalLatency / time.Duration(len(latencies))
	}

	return &BenchmarkResult{
		TotalRequests:      total,
		SuccessfulRequests: successful,
		FailedRequests:     failed,
		TotalDuration:      duration,
		RequestsPerSecond:  rps,
		AvgLatency:         avgLatency,
		MinLatency:         minLatency,
		MaxLatency:         maxLatency,
		LatencyPercentiles: lt.calculatePercentiles(latencies),
	}
}

func (lt *LoadTester) calculatePercentiles(latencies []time.Duration) map[string]time.Duration {
	// Simplified percentile calculation
	if len(latencies) == 0 {
		return map[string]time.Duration{}
	}

	// Sort latencies
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	return map[string]time.Duration{
		"p50": latencies[len(latencies)*50/100],
		"p90": latencies[len(latencies)*90/100],
		"p95": latencies[len(latencies)*95/100],
		"p99": latencies[len(latencies)*99/100],
	}
}

func main() {
	fmt.Println("API Cerberus Performance Benchmark")
	fmt.Println("==================================")

	// Test configurations
	configs := []struct {
		name        string
		concurrency int
		duration    time.Duration
	}{
		{"Light Load", 10, 30 * time.Second},
		{"Medium Load", 100, 30 * time.Second},
		{"Heavy Load", 1000, 30 * time.Second},
	}

	targetURL := "http://localhost:8080/health"
	if len(targetURL) == 0 {
		fmt.Println("Please start the API Cerberus gateway first")
		return
	}

	for _, config := range configs {
		fmt.Printf("\nRunning benchmark: %s (concurrency=%d, duration=%v)\n", config.name, config.concurrency, config.duration)
		fmt.Println("------------------------------------------------")

		tester := NewLoadTester(targetURL, "GET", config.concurrency, config.duration)
		result := tester.Run()

		fmt.Printf("Total Requests:      %d\n", result.TotalRequests)
		fmt.Printf("Successful:          %d (%.2f%%)\n", result.SuccessfulRequests, float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
		fmt.Printf("Failed:              %d (%.2f%%)\n", result.FailedRequests, float64(result.FailedRequests)/float64(result.TotalRequests)*100)
		fmt.Printf("Requests/Second:     %.2f\n", result.RequestsPerSecond)
		fmt.Printf("Avg Latency:         %v\n", result.AvgLatency)
		fmt.Printf("Min Latency:         %v\n", result.MinLatency)
		fmt.Printf("Max Latency:         %v\n", result.MaxLatency)
		fmt.Printf("P50:                 %v\n", result.LatencyPercentiles["p50"])
		fmt.Printf("P90:                 %v\n", result.LatencyPercentiles["p90"])
		fmt.Printf("P95:                 %v\n", result.LatencyPercentiles["p95"])
		fmt.Printf("P99:                 %v\n", result.LatencyPercentiles["p99"])

		if result.RequestsPerSecond >= 50000 {
			fmt.Println("\n✅ PASS: Achieved 50K+ req/sec target")
		} else {
			fmt.Printf("\n⚠️  Below 50K req/sec target (achieved %.0f)\n", result.RequestsPerSecond)
		}
	}
}
