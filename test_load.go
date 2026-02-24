//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

func main() {
	var (
		requests    int
		concurrency int
		model       string
		url         string
		authKey     string
	)

	flag.IntVar(&requests, "n", 1_000_000, "Number of requests to run")
	flag.IntVar(&concurrency, "c", 100_000, "Number of multiple requests to make at a time")
	flag.StringVar(&model, "m", "llama3.2:1b", "Model parameter for completions")
	flag.StringVar(&url, "url", "http://localhost:8000/v1/chat/completions", "Target endpoint")
	flag.StringVar(&authKey, "key", "sk-alice-001", "Bearer token to use")
	flag.Parse()

	fmt.Printf("Starting %d requests to %s with concurrency %d for %s...\n", requests, url, concurrency, model)

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
		"max_tokens": 5, // keep it extremely small for load testing
	}
	body, _ := json.Marshal(payload)

	reqs := make(chan int, requests)
	for i := 0; i < requests; i++ {
		reqs <- i
	}
	close(reqs)

	type result struct {
		duration time.Duration
		status   int
		err      error
	}

	results := make(chan result, requests)
	var wg sync.WaitGroup

	startTime := time.Now()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency,
			MaxIdleConnsPerHost: concurrency,
		},
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range reqs {
				req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+authKey)

				start := time.Now()
				resp, err := client.Do(req)
				dur := time.Since(start)

				if err != nil {
					results <- result{duration: dur, err: err}
					continue
				}

				// Must read and close body to reuse connection
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				results <- result{duration: dur, status: resp.StatusCode}
			}
		}()
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(startTime)

	var latencies []time.Duration
	statusCounts := make(map[int]int)
	errors := 0

	for r := range results {
		if r.err != nil {
			errors++
			continue
		}
		latencies = append(latencies, r.duration)
		statusCounts[r.status]++
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Total time:       %.2fs\n", totalTime.Seconds())
	fmt.Printf("Requests/sec:     %.2f\n", float64(requests)/totalTime.Seconds())
	if errors > 0 {
		fmt.Printf("Errors:           %d\n", errors)
	}

	fmt.Printf("\n=== Status Codes ===\n")
	for code, count := range statusCounts {
		fmt.Printf("[%d] %d\n", code, count)
	}

	if len(latencies) > 0 {
		fmt.Printf("\n=== Latency (%d successes) ===\n", len(latencies))
		fmt.Printf("Fastest:          %.4fs\n", latencies[0].Seconds())
		fmt.Printf("Average:          %.4fs\n", avg(latencies).Seconds())
		fmt.Printf("Slowest:          %.4fs\n", latencies[len(latencies)-1].Seconds())

		fmt.Printf("\n--- Distribution ---\n")
		p50 := latencies[len(latencies)*50/100]
		p90 := latencies[len(latencies)*90/100]
		p99 := latencies[len(latencies)*99/100]
		fmt.Printf("p50:              %.4fs\n", p50.Seconds())
		fmt.Printf("p90:              %.4fs\n", p90.Seconds())
		fmt.Printf("p99:              %.4fs\n", p99.Seconds())
	}
}

func avg(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	var total time.Duration
	for _, v := range d {
		total += v
	}
	return total / time.Duration(len(d))
}
