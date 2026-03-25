package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	apimiddleware "github.com/viola/gateway-api/internal/api/middleware"
	"github.com/viola/gateway-api/internal/audit"
	"github.com/viola/gateway-api/internal/auth"
	"github.com/viola/gateway-api/internal/authz"
	"github.com/viola/gateway-api/internal/store"
)

// TestHelpers provides test utilities
type TestHelpers struct {
	t          *testing.T
	router     *chi.Mux
	testTenant string
	jwtSecret  []byte
}

// NewTestHelpers creates test helper instance
func NewTestHelpers(t *testing.T) *TestHelpers {
	return &TestHelpers{
		t:          t,
		testTenant: "test-tenant-123",
		jwtSecret:  []byte("test-secret-key-DO-NOT-USE-IN-PRODUCTION"),
	}
}

// CreateTestJWT generates a test JWT token
func (h *TestHelpers) CreateTestJWT(claims map[string]interface{}) string {
	// Default claims
	defaultClaims := jwt.MapClaims{
		"sub":   "test-user",
		"email": "test@example.com",
		"tid":   h.testTenant,
		"iss":   "https://test.auth.com",
		"aud":   "api://test",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{},
		"scp":   "",
	}

	// Override with provided claims
	for k, v := range claims {
		defaultClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, defaultClaims)
	signedToken, err := token.SignedString(h.jwtSecret)
	if err != nil {
		h.t.Fatalf("Failed to create test JWT: %v", err)
	}

	return signedToken
}

// SetupTestRouter creates a test router with middleware
// Using interface{} to accept both real and mock stores
func (h *TestHelpers) SetupTestRouter(incidentStore interface{}, alertStore interface{}, auditor *audit.Emitter) *chi.Mux {
	r := chi.NewRouter()

	// Basic middleware (applied to all routes)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Health endpoint (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Protected API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth middleware only for API routes
		r.Use(h.mockAuthMiddleware())

		// RBAC middleware
		rbacMiddleware := &apimiddleware.RBACMiddleware{
			Authz: authz.SimpleAuthorizer{},
		}
		r.Use(rbacMiddleware.Handler)
		// Type assert to mock stores for testing
		mockIncStore := incidentStore.(*MockIncidentStore)
		mockAlertStore := alertStore.(*MockAlertStore)

		// Create test handlers with mock stores
		incidentHandlers := &TestIncidentHandlers{store: mockIncStore, auditor: auditor}
		alertHandlers := &TestAlertHandlers{store: mockAlertStore, auditor: auditor}

		// Incidents
		r.Get("/incidents", incidentHandlers.List)
		r.Get("/incidents/{id}", incidentHandlers.Get)
		r.Patch("/incidents/{id}", incidentHandlers.Update)

		// Alerts
		r.Get("/alerts", alertHandlers.List)
		r.Get("/alerts/{id}", alertHandlers.Get)
		r.Patch("/alerts/{id}", alertHandlers.Update)
	})

	h.router = r
	return r
}

// mockAuthMiddleware creates a mock authentication middleware for testing
func (h *TestHelpers) mockAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			// Extract token
			tokenString := authHeader[len("Bearer "):]

			// Parse token (without verification for tests)
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return h.jwtSecret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims := token.Claims.(jwt.MapClaims)

			// Convert to auth.Claims
			authClaims := &auth.Claims{
				Subject:  getString(claims, "sub"),
				Email:    getString(claims, "email"),
				TenantID: getString(claims, "tid"),
				Issuer:   getString(claims, "iss"),
				Audience: getString(claims, "aud"),
				Roles:    getStringSlice(claims, "roles"),
				Scopes:   parseScopes(getString(claims, "scp")),
				Raw:      claims,
			}

			if exp, ok := claims["exp"].(float64); ok {
				authClaims.Expiry = time.Unix(int64(exp), 0)
			}

			// Add claims to context
			ctx := apimiddleware.WithClaims(r.Context(), authClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Helper functions
func getString(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}

func getStringSlice(claims jwt.MapClaims, key string) []string {
	if v, ok := claims[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}

func parseScopes(scp string) []string {
	if scp == "" {
		return []string{}
	}
	return []string{scp} // Simplified
}

// Request makes an HTTP request and returns the response
func (h *TestHelpers) Request(method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rr := httptest.NewRecorder()
	h.router.ServeHTTP(rr, req)

	return rr
}

// AssertStatus checks HTTP status code
func (h *TestHelpers) AssertStatus(rr *httptest.ResponseRecorder, expected int) {
	if rr.Code != expected {
		h.t.Errorf("Expected status %d, got %d. Body: %s", expected, rr.Code, rr.Body.String())
	}
}

// AssertJSON checks if response is valid JSON
func (h *TestHelpers) AssertJSON(rr *httptest.ResponseRecorder, target interface{}) {
	if err := json.NewDecoder(rr.Body).Decode(target); err != nil {
		h.t.Errorf("Failed to decode JSON response: %v. Body: %s", err, rr.Body.String())
	}
}

// MockIncidentStore creates an in-memory incident store for testing
type MockIncidentStore struct {
	incidents map[string]*store.Incident
}

func NewMockIncidentStore() *MockIncidentStore {
	return &MockIncidentStore{
		incidents: make(map[string]*store.Incident),
	}
}

func (m *MockIncidentStore) List(ctx context.Context, tenantID string, filter store.ListIncidentsFilter) ([]*store.Incident, error) {
	result := []*store.Incident{}
	for _, inc := range m.incidents {
		if inc.TenantID == tenantID {
			result = append(result, inc)
		}
	}
	return result, nil
}

func (m *MockIncidentStore) Get(ctx context.Context, tenantID, incidentID string) (*store.Incident, error) {
	key := tenantID + ":" + incidentID
	if inc, ok := m.incidents[key]; ok {
		return inc, nil
	}
	return nil, nil
}

func (m *MockIncidentStore) Update(ctx context.Context, tenantID, incidentID string, updates map[string]interface{}) error {
	key := tenantID + ":" + incidentID
	if inc, ok := m.incidents[key]; ok {
		if status, ok := updates["status"].(string); ok {
			inc.Status = status
		}
		if assignedTo, ok := updates["assigned_to"].(string); ok {
			inc.AssignedTo = &assignedTo
		}
		if closureReason, ok := updates["closure_reason"].(string); ok {
			inc.ClosureReason = &closureReason
		}
		inc.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockIncidentStore) Upsert(ctx context.Context, inc *store.Incident) error {
	key := inc.TenantID + ":" + inc.IncidentID
	m.incidents[key] = inc
	return nil
}

// MockAlertStore creates an in-memory alert store for testing
type MockAlertStore struct {
	alerts map[string]*store.Alert
}

func NewMockAlertStore() *MockAlertStore {
	return &MockAlertStore{
		alerts: make(map[string]*store.Alert),
	}
}

func (m *MockAlertStore) List(ctx context.Context, tenantID string, filter store.ListAlertsFilter) ([]*store.Alert, error) {
	result := []*store.Alert{}
	for _, alert := range m.alerts {
		if alert.TenantID == tenantID {
			result = append(result, alert)
		}
	}
	return result, nil
}

func (m *MockAlertStore) Get(ctx context.Context, tenantID, alertID string) (*store.Alert, error) {
	key := tenantID + ":" + alertID
	if alert, ok := m.alerts[key]; ok {
		return alert, nil
	}
	return nil, nil
}

func (m *MockAlertStore) Update(ctx context.Context, tenantID, alertID string, updates map[string]interface{}) error {
	key := tenantID + ":" + alertID
	if alert, ok := m.alerts[key]; ok {
		if status, ok := updates["status"].(string); ok {
			alert.Status = status
		}
		if assignedTo, ok := updates["assigned_to"].(string); ok {
			alert.AssignedTo = &assignedTo
		}
		if closureReason, ok := updates["closure_reason"].(string); ok {
			alert.ClosureReason = &closureReason
		}
		alert.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockAlertStore) Upsert(ctx context.Context, alert *store.Alert) error {
	key := alert.TenantID + ":" + alert.AlertID
	m.alerts[key] = alert
	return nil
}

// MockAuditor creates a no-op auditor for testing
type MockAuditor struct{}

func NewMockAuditor() *audit.Emitter {
	// Return nil for tests (handlers should handle nil auditor)
	return nil
}

// TestIncidentHandlers wraps incident handlers for testing with mock store
type TestIncidentHandlers struct {
	store   *MockIncidentStore
	auditor *audit.Emitter
}

func (h *TestIncidentHandlers) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := apimiddleware.ClaimsFrom(r.Context())
	if !ok || claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	incidents, err := h.store.List(r.Context(), claims.TenantID, store.ListIncidentsFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"incidents": incidents})
}

func (h *TestIncidentHandlers) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := apimiddleware.ClaimsFrom(r.Context())
	if !ok || claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	incidentID := chi.URLParam(r, "id")
	incident, err := h.store.Get(r.Context(), claims.TenantID, incidentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if incident == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "incident not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incident)
}

func (h *TestIncidentHandlers) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := apimiddleware.ClaimsFrom(r.Context())
	if !ok || claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	incidentID := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status if provided
	if status, ok := updates["status"].(string); ok {
		validStatuses := map[string]bool{
			"open":   true,
			"ack":    true,
			"closed": true,
		}
		if !validStatuses[status] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid status value"})
			return
		}
	}

	if err := h.store.Update(r.Context(), claims.TenantID, incidentID, updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// TestAlertHandlers wraps alert handlers for testing with mock store
type TestAlertHandlers struct {
	store   *MockAlertStore
	auditor *audit.Emitter
}

func (h *TestAlertHandlers) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := apimiddleware.ClaimsFrom(r.Context())
	if !ok || claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	alerts, err := h.store.List(r.Context(), claims.TenantID, store.ListAlertsFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"alerts": alerts})
}

func (h *TestAlertHandlers) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := apimiddleware.ClaimsFrom(r.Context())
	if !ok || claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	alertID := chi.URLParam(r, "id")
	alert, err := h.store.Get(r.Context(), claims.TenantID, alertID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if alert == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "alert not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alert)
}

func (h *TestAlertHandlers) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := apimiddleware.ClaimsFrom(r.Context())
	if !ok || claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	alertID := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status if provided
	if status, ok := updates["status"].(string); ok {
		validStatuses := map[string]bool{
			"open":   true,
			"ack":    true,
			"closed": true,
		}
		if !validStatuses[status] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid status value"})
			return
		}
	}

	if err := h.store.Update(r.Context(), claims.TenantID, alertID, updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
