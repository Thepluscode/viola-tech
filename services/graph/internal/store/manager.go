package store

import (
	"fmt"
	"sync"
)

// GraphManager manages per-tenant graphs
type GraphManager struct {
	mu     sync.RWMutex
	graphs map[string]*Graph // key: tenant ID
}

// NewGraphManager creates a new graph manager
func NewGraphManager() *GraphManager {
	return &GraphManager{
		graphs: make(map[string]*Graph),
	}
}

// GetOrCreateGraph gets an existing graph or creates a new one for the tenant
func (m *GraphManager) GetOrCreateGraph(tenantID string) *Graph {
	m.mu.RLock()
	graph, exists := m.graphs[tenantID]
	m.mu.RUnlock()

	if exists {
		return graph
	}

	// Need to create
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check (another goroutine may have created it)
	if graph, exists := m.graphs[tenantID]; exists {
		return graph
	}

	graph = NewGraph(tenantID)
	m.graphs[tenantID] = graph
	return graph
}

// GetGraph returns a graph for a tenant (nil if not found)
func (m *GraphManager) GetGraph(tenantID string) *Graph {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.graphs[tenantID]
}

// AddNode adds a node to a tenant's graph
func (m *GraphManager) AddNode(tenantID string, node *Node) error {
	if tenantID == "" {
		return fmt.Errorf("tenant ID cannot be empty")
	}

	graph := m.GetOrCreateGraph(tenantID)
	return graph.AddNode(node)
}

// AddEdge adds an edge to a tenant's graph
func (m *GraphManager) AddEdge(tenantID string, edge *Edge) error {
	if tenantID == "" {
		return fmt.Errorf("tenant ID cannot be empty")
	}

	graph := m.GetOrCreateGraph(tenantID)
	return graph.AddEdge(edge)
}

// GetNode gets a node from a tenant's graph
func (m *GraphManager) GetNode(tenantID string, nodeID string) *Node {
	graph := m.GetGraph(tenantID)
	if graph == nil {
		return nil
	}
	return graph.GetNode(nodeID)
}

// CleanupExpiredEdges removes expired edges from all tenant graphs
func (m *GraphManager) CleanupExpiredEdges() map[string]int {
	m.mu.RLock()
	tenants := make([]string, 0, len(m.graphs))
	for tid := range m.graphs {
		tenants = append(tenants, tid)
	}
	m.mu.RUnlock()

	results := make(map[string]int)
	for _, tid := range tenants {
		graph := m.GetGraph(tid)
		if graph != nil {
			removed := graph.RemoveExpiredEdges()
			if removed > 0 {
				results[tid] = removed
			}
		}
	}

	return results
}

// TenantCount returns the number of tenants with graphs
func (m *GraphManager) TenantCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.graphs)
}

// AllStats returns statistics for all tenant graphs
func (m *GraphManager) AllStats() map[string]GraphStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]GraphStats)
	for tid, graph := range m.graphs {
		stats[tid] = graph.Stats()
	}
	return stats
}
