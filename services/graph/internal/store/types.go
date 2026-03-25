package store

import (
	"time"
)

// NodeType represents the type of node in the graph
type NodeType string

const (
	NodeTypeUser       NodeType = "user"
	NodeTypeService    NodeType = "service"
	NodeTypeEndpoint   NodeType = "endpoint"
	NodeTypeCloud      NodeType = "cloud"
	NodeTypeNetworkSeg NodeType = "network_seg"
)

// EdgeType represents the type of relationship between nodes
type EdgeType string

const (
	EdgeTypeAuth    EdgeType = "auth"       // user → endpoint
	EdgeTypeNetwork EdgeType = "network"    // endpoint → endpoint
	EdgeTypeRole    EdgeType = "role"       // user → role
	EdgeTypeAccess  EdgeType = "access"     // user/service → cloud
	EdgeTypeSpawn   EdgeType = "spawn"      // user → endpoint (process spawn)
)

// Node represents an entity in the attack graph
type Node struct {
	ID          string            `json:"id"`           // e.g., "user:alice", "endpoint:host-123"
	Type        NodeType          `json:"type"`         // user|service|endpoint|cloud|network_seg
	Labels      map[string]string `json:"labels"`       // org_unit, zone, tags
	Criticality int               `json:"criticality"`  // 0-100 (crown jewel = 100)
	FirstSeen   time.Time         `json:"first_seen"`
	LastSeen    time.Time         `json:"last_seen"`

	// Computed properties (cached)
	RiskScore     float64 `json:"risk_score"`      // 0-100
	CrownDistance int     `json:"crown_distance"`  // hops to nearest crown jewel (-1 if unreachable)
	BlastRadius   int     `json:"blast_radius"`    // number of reachable nodes
}

// Edge represents a relationship between two nodes
type Edge struct {
	ID        string            `json:"id"`        // e.g., "auth:alice:host-123"
	Type      EdgeType          `json:"type"`      // auth|network|role|access|spawn
	Source    string            `json:"source"`    // source node ID
	Target    string            `json:"target"`    // target node ID
	Weight    float64           `json:"weight"`    // trust/confidence score (0-1)
	Timestamp time.Time         `json:"timestamp"` // when relationship observed
	TTL       time.Duration     `json:"ttl"`       // how long before edge expires
	Metadata  map[string]string `json:"metadata"`  // type-specific metadata
}

// IsExpired checks if the edge has exceeded its TTL
func (e *Edge) IsExpired() bool {
	if e.TTL == 0 {
		return false // No TTL set
	}
	return time.Since(e.Timestamp) > e.TTL
}

// GraphStats holds statistics about a graph
type GraphStats struct {
	NodeCount     int       `json:"node_count"`
	EdgeCount     int       `json:"edge_count"`
	LastRebuildAt time.Time `json:"last_rebuild_at"`
	LastUpdateAt  time.Time `json:"last_update_at"`
}
