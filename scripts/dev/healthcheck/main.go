// healthcheck — Quick service-level health verification for Viola XDR.
//
// Usage:
//
//	go run ./scripts/dev/healthcheck/ [--timeout 60s] [--verbose]
//
// Checks: postgres, kafka, auth, gateway-api, intel, response, detection metrics,
// graph metrics, and an authenticated gateway round-trip.
//
// Exit 0 if all pass, exit 1 if any fail.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// ANSI colors
// ---------------------------------------------------------------------------

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	bold   = "\033[1m"
	dim    = "\033[2m"
)

// ---------------------------------------------------------------------------
// Check framework
// ---------------------------------------------------------------------------

type checkFunc func(ctx context.Context) error

type checkResult struct {
	name     string
	passed   bool
	duration time.Duration
	err      error
}

func main() {
	timeout := flag.Duration("timeout", 60*time.Second, "maximum time for all checks")
	verbose := flag.Bool("verbose", false, "show error details")
	flag.Parse()

	checks := []struct {
		name string
		fn   checkFunc
	}{
		{"postgres connectivity", checkPostgres},
		{"kafka connectivity", checkKafka},
		{"auth service health", checkHealth("http://localhost:8081/health")},
		{"auth token issuance", checkAuthToken},
		{"gateway-api health", checkHealth("http://localhost:8080/health")},
		{"gateway-api authenticated", checkGatewayAuth},
		{"intel service health", checkHealth("http://localhost:8082/health")},
		{"response service health", checkHealth("http://localhost:8083/health")},
		{"detection metrics", checkHealth("http://localhost:9090/metrics")},
		{"graph metrics", checkHealth("http://localhost:9091/metrics")},
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	fmt.Printf("\n%s%s Viola Platform — Health Check%s\n", bold, yellow, reset)
	fmt.Printf("%s%s%s\n\n", dim, strings.Repeat("─", 50), reset)

	results := make([]checkResult, 0, len(checks))
	for _, c := range checks {
		start := time.Now()
		err := c.fn(ctx)
		elapsed := time.Since(start)

		r := checkResult{name: c.name, passed: err == nil, duration: elapsed, err: err}
		results = append(results, r)

		icon := green + "✓" + reset
		if !r.passed {
			icon = red + "✗" + reset
		}
		fmt.Printf("  %s  %-35s %s%6dms%s", icon, r.name, dim, elapsed.Milliseconds(), reset)
		if !r.passed && *verbose {
			fmt.Printf("  %s%s%s", red, err.Error(), reset)
		}
		fmt.Println()
	}

	// Summary
	fmt.Printf("\n%s%s%s\n", dim, strings.Repeat("─", 50), reset)
	passed, failed := 0, 0
	var total time.Duration
	for _, r := range results {
		total += r.duration
		if r.passed {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("  %s%d passed%s", green, passed, reset)
	if failed > 0 {
		fmt.Printf("  %s%d failed%s", red, failed, reset)
	}
	fmt.Printf("  %s(%dms total)%s\n\n", dim, total.Milliseconds(), reset)

	if failed > 0 && *verbose {
		fmt.Printf("%s%sFailed:%s\n", bold, red, reset)
		for _, r := range results {
			if !r.passed {
				fmt.Printf("  %s✗ %s%s: %s\n", red, r.name, reset, r.err)
			}
		}
		fmt.Println()
	}

	if failed > 0 {
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Check implementations
// ---------------------------------------------------------------------------

func checkPostgres(ctx context.Context) error {
	return dialTCP("localhost:5435", 3*time.Second)
}

func checkKafka(ctx context.Context) error {
	return dialTCP("localhost:9094", 3*time.Second)
}

func checkHealth(url string) checkFunc {
	return func(ctx context.Context) error {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("status %d: %s", resp.StatusCode, body)
		}
		return nil
	}
}

func checkAuthToken(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:8081/token", nil)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}
	var tr map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &tr)
	if tr["token"] == nil && tr["access_token"] == nil {
		return fmt.Errorf("no token in response")
	}
	return nil
}

func checkGatewayAuth(ctx context.Context) error {
	// Get token
	treq, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:8081/token", nil)
	tresp, err := (&http.Client{Timeout: 5 * time.Second}).Do(treq)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}
	defer tresp.Body.Close()
	var tr map[string]interface{}
	tbody, _ := io.ReadAll(tresp.Body)
	json.Unmarshal(tbody, &tr)
	token := ""
	if t, ok := tr["token"].(string); ok {
		token = t
	} else if t, ok := tr["access_token"].(string); ok {
		token = t
	}
	if token == "" {
		return fmt.Errorf("no token from auth")
	}

	// Use token
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/api/v1/alerts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func dialTCP(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
