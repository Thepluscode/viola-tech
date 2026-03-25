package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apimiddleware "github.com/viola/gateway-api/internal/api/middleware"
	"github.com/viola/gateway-api/internal/audit"
	"github.com/viola/gateway-api/internal/auth"
	"github.com/viola/gateway-api/internal/authz"
	"github.com/viola/gateway-api/internal/db"
	"github.com/viola/gateway-api/internal/handlers"
	"github.com/viola/gateway-api/internal/store"
	"github.com/viola/shared/observability/logging"
	obsmiddleware "github.com/viola/shared/observability/middleware"
	"github.com/viola/shared/observability/metrics"
	"github.com/viola/shared/observability/tracing"
)

func main() {
	// Root context — cancelled on SIGTERM / SIGINT.
	// All background goroutines (JWKS refresh, etc.) must use this context
	// so they exit cleanly on shutdown (H4 fix).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	env := getenv("VIOLA_ENV", "dev")

	// Initialize observability
	metricsRegistry := metrics.NewRegistry("gateway_api")
	logger := logging.New("gateway-api", logging.INFO)
	tracer := tracing.Tracer("gateway-api")

	// Initialize OpenTelemetry tracing (optional)
	tracingEnabled := getenv("TRACING_ENABLED", "false") == "true"
	if tracingEnabled {
		otlpEndpoint := getenv("OTLP_ENDPOINT", "localhost:4317")
		shutdown, err := tracing.InitTracer(tracing.Config{
			ServiceName:    "gateway-api",
			ServiceVersion: "1.0.0",
			Environment:    env,
			OTLPEndpoint:   otlpEndpoint,
			Enabled:        true,
		})
		if err != nil {
			log.Printf("WARN: failed to initialize tracing: %v", err)
		} else {
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = shutdown(ctx)
			}()
			log.Printf("Tracing enabled, exporting to %s", otlpEndpoint)
		}
	}

	// Database connection
	database, err := db.New(ctx, db.Config{
		Host:     getenv("PG_HOST", "localhost"),
		Port:     getenvInt("PG_PORT", 5432),
		User:     getenv("PG_USER", "postgres"),
		Password: getenv("PG_PASSWORD", "postgres"),
		Database: getenv("PG_DATABASE", "viola_gateway"),
		SSLMode:  getenv("PG_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Database connected")

	// Initialize OIDC authentication (with JWKS)
	requireBearer := getenv("AUTH_REQUIRE_BEARER", "true") == "true"
	var authMiddleware *apimiddleware.AuthMiddleware
	var rbacMiddleware *apimiddleware.RBACMiddleware

	if requireBearer {
		issuerURL := getenv("OIDC_ISSUER_URL", "")
		if issuerURL == "" {
			log.Fatal("OIDC_ISSUER_URL required when AUTH_REQUIRE_BEARER=true")
		}

		audience := getenv("OIDC_AUDIENCE", "")
		if audience == "" {
			log.Fatal("OIDC_AUDIENCE required when AUTH_REQUIRE_BEARER=true")
		}

		// Discover or use explicit JWKS URL
		jwksURL := getenv("OIDC_JWKS_URL", "")
		if jwksURL == "" {
			log.Printf("Discovering JWKS URL from %s", issuerURL)
			discovered, err := auth.DiscoverJWKSURL(ctx, issuerURL)
			if err != nil {
				log.Fatalf("Failed to discover JWKS URL: %v", err)
			}
			jwksURL = discovered
			log.Printf("Discovered JWKS URL: %s", jwksURL)
		}

		// Create JWKS cache
		jwksCache, err := auth.NewJWKSCache(auth.JWKSCacheConfig{
			JWKSURL:      jwksURL,
			RefreshEvery: 10 * time.Minute,
		})
		if err != nil {
			log.Fatalf("Failed to create JWKS cache: %v", err)
		}

		// Initial refresh — populate keys before serving traffic.
		if err := jwksCache.Refresh(ctx); err != nil {
			log.Fatalf("Failed to refresh JWKS: %v", err)
		}
		// StartBackground ties the periodic refresh goroutine to the app
		// lifecycle context so it exits on SIGTERM (H4 fix).
		jwksCache.StartBackground(ctx)
		log.Println("JWKS cache initialized")

		// Parse allowed algorithms
		allowedAlgos := map[string]bool{}
		for _, alg := range strings.Split(getenv("AUTH_ALLOW_ALGOS", "RS256"), ",") {
			allowedAlgos[strings.TrimSpace(alg)] = true
		}

		// Clock skew tolerance
		skewSeconds := 120
		if v := getenv("OIDC_CLOCK_SKEW_SECONDS", ""); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				skewSeconds = parsed
			}
		}

		// Create JWT verifier
		verifier := &auth.Verifier{
			Issuer:   issuerURL,
			Audience: audience,
			Algos:    allowedAlgos,
			Skew:     time.Duration(skewSeconds) * time.Second,
			JWKS:     jwksCache,
		}

		authMiddleware = &apimiddleware.AuthMiddleware{
			RequireBearer: true,
			Verifier:      verifier,
			Auditor:       nil, // Set below after audit emitter is created
			AuditSample:   0.1, // 10% sampling for failed auth
		}

		// Initialize RBAC authorizer
		authorizer := authz.SimpleAuthorizer{}
		rbacMiddleware = &apimiddleware.RBACMiddleware{
			Authz: authorizer,
		}

		log.Printf("OIDC authentication enabled (issuer=%s, audience=%s)", issuerURL, audience)
	} else {
		log.Println("WARNING: OIDC disabled - authentication bypassed")
	}

	// Initialize audit emitter
	var auditor *audit.Emitter
	if auditBroker := getenv("AUDIT_KAFKA_BROKER", ""); auditBroker != "" {
		auditor, err = audit.New(audit.Config{
			Service: getenv("SERVICE_NAME", "gateway-api"),
			Brokers: []string{auditBroker},
			Topic:   getenv("AUDIT_TOPIC", "viola.dev.audit.v1.event"),
		})
		if err != nil {
			log.Fatalf("Failed to initialize audit emitter: %v", err)
		}
		defer auditor.Close()
		log.Println("Audit emitter initialized")

		// Set auditor in auth middleware for failed auth logging
		if authMiddleware != nil {
			authMiddleware.Auditor = auditor
		}
	} else {
		log.Println("WARNING: Audit disabled (AUDIT_KAFKA_BROKER not set)")
	}

	// Initialize rate limiting (optional)
	var rateLimitMiddleware *apimiddleware.RateLimitMiddleware
	if getenv("RATE_LIMIT_ENABLED", "false") == "true" {
		maxRequests := 120 // Default
		if v := getenv("RATE_LIMIT_PER_MIN_DEFAULT", ""); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				maxRequests = parsed
			}
		}

		keyClaims := []string{"sub", "tid"} // Default
		if v := getenv("RATE_LIMIT_KEY_CLAIMS", ""); v != "" {
			keyClaims = strings.Split(v, ",")
		}

		rateLimitMiddleware = apimiddleware.NewRateLimitMiddleware(apimiddleware.RateLimitConfig{
			Enabled:          true,
			MaxRequests:      maxRequests,
			WindowMinutes:    1,
			KeyClaims:        keyClaims,
			CustomLimitClaim: getenv("RATE_LIMIT_PER_MIN_CLAIM", ""),
		})
		log.Printf("Rate limiting enabled (max=%d req/min, key=%v)", maxRequests, keyClaims)
	}

	// Initialize token introspection (optional, for revocation checks)
	var introspectionMiddleware *apimiddleware.IntrospectionMiddleware
	if getenv("INTROSPECTION_ENABLED", "false") == "true" {
		introspectionEndpoint := getenv("INTROSPECTION_ENDPOINT", "")
		introspectionClientID := getenv("INTROSPECTION_CLIENT_ID", "")
		introspectionClientSecret := getenv("INTROSPECTION_CLIENT_SECRET", "")

		if introspectionEndpoint != "" && introspectionClientID != "" && introspectionClientSecret != "" {
			introspectionClient, err := auth.NewIntrospectionClient(auth.IntrospectionConfig{
				Endpoint:     introspectionEndpoint,
				ClientID:     introspectionClientID,
				ClientSecret: introspectionClientSecret,
			})
			if err != nil {
				log.Fatalf("Failed to create introspection client: %v", err)
			}

			introspectionMiddleware = apimiddleware.NewIntrospectionMiddleware(introspectionClient, true)
			log.Println("Token introspection enabled")
		}
	}

	// Initialize MFA enforcement (optional)
	var mfaMiddleware *apimiddleware.MFAMiddleware
	if getenv("MFA_REQUIRED", "false") == "true" {
		mfaIndicators := []string{"mfa", "otp", "totp", "sms", "duo", "fido", "phone"}
		if v := getenv("MFA_INDICATORS", ""); v != "" {
			mfaIndicators = strings.Split(v, ",")
		}

		mfaMiddleware = apimiddleware.NewMFAMiddleware(false, mfaIndicators) // false = not required by default
		log.Printf("MFA enforcement available (indicators=%v)", mfaIndicators)
	}

	// Initialize stores
	incidentStore := store.NewIncidentStore(database.Pool())
	alertStore := store.NewAlertStore(database.Pool())

	// Initialize handlers (with audit emitter)
	incidentHandlers := handlers.NewIncidentHandlers(incidentStore, auditor)
	alertHandlers := handlers.NewAlertHandlers(alertStore, auditor)

	// Setup router
	r := chi.NewRouter()

	// CORS (must be before auth to handle preflight OPTIONS)
	corsOrigins := getenv("CORS_ORIGINS", "http://localhost:3000")
	corsCfg := apimiddleware.DefaultCORSConfig()
	corsCfg.AllowedOrigins = strings.Split(corsOrigins, ",")
	corsCfg.AllowCredentials = true
	r.Use(apimiddleware.CORSMiddleware(corsCfg))

	// Middleware (order matters!)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimiddleware.SecurityHeadersMiddleware)               // Security headers
	r.Use(apimiddleware.RequestValidationMiddleware(1 << 20))    // 1MB max body
	r.Use(obsmiddleware.LoggingMiddleware(logger))               // Structured logging
	r.Use(obsmiddleware.TracingMiddleware(tracer))               // OpenTelemetry tracing
	r.Use(obsmiddleware.MetricsMiddleware(metricsRegistry))      // Prometheus metrics
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Metrics endpoint
	r.Handle("/metrics", metricsRegistry.Handler())

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := database.Pool().Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("DB unavailable"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	// API routes (protected by OIDC + RBAC)
	r.Route("/api/v1", func(r chi.Router) {
		// Apply authentication middleware (if enabled)
		if authMiddleware != nil {
			r.Use(authMiddleware.Handler)
		}

		// Apply token introspection (if enabled)
		if introspectionMiddleware != nil {
			r.Use(introspectionMiddleware.Handler)
		}

		// Apply MFA enforcement (if enabled)
		if mfaMiddleware != nil {
			r.Use(mfaMiddleware.Handler)
		}

		// Apply rate limiting (if enabled)
		if rateLimitMiddleware != nil {
			r.Use(rateLimitMiddleware.Handler)
		}

		// Apply RBAC middleware (if enabled)
		if rbacMiddleware != nil {
			r.Use(rbacMiddleware.Handler)
		}

		// Incidents
		r.Get("/incidents", incidentHandlers.List)
		r.Get("/incidents/{id}", incidentHandlers.Get)
		r.Patch("/incidents/{id}", incidentHandlers.Update)

		// Alerts
		r.Get("/alerts", alertHandlers.List)
		r.Get("/alerts/{id}", alertHandlers.Get)
		r.Patch("/alerts/{id}", alertHandlers.Update)
	})

	// Start server
	port := getenv("PORT", "8080")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown: when the signal.NotifyContext is cancelled (SIGTERM/SIGINT),
	// drain in-flight requests with a 10-second deadline.
	go func() {
		<-ctx.Done() // cancelled by signal.NotifyContext on SIGTERM / Ctrl-C
		stop()       // release signal resources
		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("gateway-api listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		// Simple atoi - use strconv for production
		var i int
		for _, c := range v {
			if c < '0' || c > '9' {
				return fallback
			}
			i = i*10 + int(c-'0')
		}
		return i
	}
	return fallback
}
