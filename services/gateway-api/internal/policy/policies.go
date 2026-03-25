package policy

// Permission represents a fine-grained permission
type Permission string

const (
	PermIncidentsRead  Permission = "incidents:read"
	PermIncidentsWrite Permission = "incidents:write"
	PermAlertsRead     Permission = "alerts:read"
	PermAlertsWrite    Permission = "alerts:write"
	PermRulesWrite     Permission = "rules:write"
	PermAdmin          Permission = "admin:*"
)

// RoutePolicy defines authorization requirements for a route
type RoutePolicy struct {
	Method string
	Path   string
	AnyOf  []Permission // require any
	AllOf  []Permission // require all
}

// Policies is the central policy table
// This is the single source of truth for route authorization
var Policies = []RoutePolicy{
	// Incidents
	{Method: "GET", Path: "/api/v1/incidents", AnyOf: []Permission{PermIncidentsRead, PermAdmin}},
	{Method: "GET", Path: "/api/v1/incidents/{id}", AnyOf: []Permission{PermIncidentsRead, PermAdmin}},
	{Method: "PATCH", Path: "/api/v1/incidents/{id}", AnyOf: []Permission{PermIncidentsWrite, PermAdmin}},

	// Alerts
	{Method: "GET", Path: "/api/v1/alerts", AnyOf: []Permission{PermAlertsRead, PermAdmin}},
	{Method: "GET", Path: "/api/v1/alerts/{id}", AnyOf: []Permission{PermAlertsRead, PermAdmin}},
	{Method: "PATCH", Path: "/api/v1/alerts/{id}", AnyOf: []Permission{PermAlertsWrite, PermAdmin}},

	// Detection Rules (future)
	// {Method: "POST", Path: "/api/v1/rules", AnyOf: []Permission{PermRulesWrite, PermAdmin}},
	// {Method: "PUT", Path: "/api/v1/rules/{id}", AnyOf: []Permission{PermRulesWrite, PermAdmin}},
	// {Method: "DELETE", Path: "/api/v1/rules/{id}", AnyOf: []Permission{PermRulesWrite, PermAdmin}},
}
