package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/viola/auth/internal/handler"
	"github.com/viola/auth/internal/keys"
	"github.com/viola/auth/internal/token"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	port := getenv("PORT", "8081")
	issuerURL := getenv("ISSUER_URL", "http://auth:8081")
	audience := getenv("AUDIENCE", "viola-api")
	ttlStr := getenv("TOKEN_TTL", "1h")

	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		log.Fatalf("invalid TOKEN_TTL %q: %v", ttlStr, err)
	}

	// Load or generate RSA key pair.
	kp, err := keys.Load()
	if err != nil {
		log.Fatalf("load keys: %v", err)
	}
	log.Printf("auth: signing key loaded (kid=%s)", kp.KID)

	issuer := token.New(token.Config{
		PrivateKey: kp.Private,
		KID:        kp.KID,
		Issuer:     issuerURL,
		Audience:   audience,
		TTL:        ttl,
	})

	h := handler.New(issuer, kp.JWKS)

	mux := http.NewServeMux()
	mux.HandleFunc("/token", h.Token)
	mux.HandleFunc("/.well-known/jwks.json", h.JWKS)
	mux.HandleFunc("/health", h.Health)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		stop()
		log.Println("auth: shutting down")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("auth: listening on :%s (issuer=%s, audience=%s, ttl=%s)", port, issuerURL, audience, ttl)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("auth: %v", err)
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
