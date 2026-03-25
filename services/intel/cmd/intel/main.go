package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/viola/intel/internal/enrichment"
	"github.com/viola/intel/internal/handler"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	port := getenv("PORT", "8082")
	apiKey := os.Getenv("OTX_API_KEY") // optional; empty = unauthenticated lookups
	cacheTTLStr := getenv("CACHE_TTL", "10m")

	cacheTTL, err := time.ParseDuration(cacheTTLStr)
	if err != nil {
		log.Fatalf("invalid CACHE_TTL %q: %v", cacheTTLStr, err)
	}

	otx := enrichment.NewClient(apiKey)
	cached := enrichment.NewCachedClient(otx, cacheTTL)

	// Background cache purge every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cached.Purge()
			}
		}
	}()

	h := handler.New(cached)

	mux := http.NewServeMux()
	mux.HandleFunc("/enrich", h.Enrich)
	mux.HandleFunc("/health", h.Health)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		<-ctx.Done()
		stop()
		log.Println("intel: shutting down")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if apiKey != "" {
		log.Printf("intel: listening on :%s (OTX authenticated, cache_ttl=%s)", port, cacheTTL)
	} else {
		log.Printf("intel: listening on :%s (OTX unauthenticated — set OTX_API_KEY for higher rate limits, cache_ttl=%s)", port, cacheTTL)
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("intel: %v", err)
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
