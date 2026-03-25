package auth

import "time"

// Claims represents parsed JWT claims
type Claims struct {
	TenantID string
	Subject  string
	Email    string
	Roles    []string
	Scopes   []string

	Issuer   string
	Audience string
	Expiry   time.Time

	Raw map[string]any
}
