// Package handler exposes the intel enrichment API over HTTP.
//
// Endpoints:
//
//	GET /enrich?type=ip&value=1.2.3.4
//	GET /enrich?type=domain&value=evil.com
//	GET /enrich?type=hash&value=<md5|sha1|sha256>
//	GET /health
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/viola/intel/internal/enrichment"
)

// Enricher is the interface satisfied by both enrichment.Client and
// enrichment.CachedClient.
type Enricher interface {
	LookupIP(ctx context.Context, ip string) (*enrichment.Result, error)
	LookupDomain(ctx context.Context, domain string) (*enrichment.Result, error)
	LookupHash(ctx context.Context, hash string) (*enrichment.Result, error)
}

// Handler holds the enricher dependency.
type Handler struct {
	enricher Enricher
}

// New creates a Handler.
func New(e Enricher) *Handler {
	return &Handler{enricher: e}
}

// Enrich handles GET /enrich — looks up a single IOC.
func (h *Handler) Enrich(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	iocType := q.Get("type")
	value := q.Get("value")

	if value == "" {
		writeError(w, http.StatusBadRequest, "value query param is required")
		return
	}

	var (
		result *enrichment.Result
		err    error
	)

	switch iocType {
	case "ip":
		result, err = h.enricher.LookupIP(r.Context(), value)
	case "domain":
		result, err = h.enricher.LookupDomain(r.Context(), value)
	case "hash":
		result, err = h.enricher.LookupHash(r.Context(), value)
	default:
		writeError(w, http.StatusBadRequest, "type must be one of: ip, domain, hash")
		return
	}

	if err != nil {
		// OTX unreachable / rate limited — return partial result so callers can
		// continue without enrichment data.
		writeError(w, http.StatusBadGateway, "enrichment lookup failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Health handles GET /health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
