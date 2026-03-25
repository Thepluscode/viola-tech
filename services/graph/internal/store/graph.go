package store

import (
	"fmt"
	"sync"
	"time"
)

// Graph represents an in-memory attack graph for a single tenant
type Graph struct {
	tenantID string

	mu sync.RWMutex

	nodes map[string]*Node  // key: node ID
	edges map[string]*Edge  // key: edge ID

	// Adjacency lists for efficient traversal
	outEdges map[string][]string // node ID → list of outgoing edge IDs
	inEdges  map[string][]string // node ID → list of incoming edge IDs

	stats GraphStats
}

// NewGraph creates a new empty graph for a tenant
func NewGraph(tenantID string) *Graph {
	return &Graph{
		tenantID: tenantID,
		nodes:    make(map[string]*Node),
		edges:    make(map[string]*Edge),
		outEdges: make(map[string][]string),
		inEdges:  make(map[string][]string),
		stats: GraphStats{
			LastRebuildAt: time.Now(),
			LastUpdateAt:  time.Now(),
		},
	}
}

// AddNode adds or updates a node in the graph
func (g *Graph) AddNode(node *Node) error {
	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	existing, exists := g.nodes[node.ID]
	if exists {
		// Update existing node
		existing.Labels = node.Labels
		existing.Criticality = node.Criticality
		existing.LastSeen = time.Now()
	} else {
		// Add new node
		node.FirstSeen = time.Now()
		node.LastSeen = time.Now()
		node.CrownDistance = -1 // Unknown until computed
		g.nodes[node.ID] = node
		g.stats.NodeCount++
	}

	g.stats.LastUpdateAt = time.Now()
	return nil
}

// AddEdge adds or updates an edge in the graph
func (g *Graph) AddEdge(edge *Edge) error {
	if edge.ID == "" {
		return fmt.Errorf("edge ID cannot be empty")
	}
	if edge.Source == "" || edge.Target == "" {
		return fmt.Errorf("edge source and target cannot be empty")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Ensure source and target nodes exist
	if _, exists := g.nodes[edge.Source]; !exists {
		return fmt.Errorf("source node %s not found", edge.Source)
	}
	if _, exists := g.nodes[edge.Target]; !exists {
		return fmt.Errorf("target node %s not found", edge.Target)
	}

	existing, exists := g.edges[edge.ID]
	if exists {
		// Update existing edge (refresh timestamp/TTL)
		existing.Timestamp = time.Now()
		existing.Weight = edge.Weight
		existing.Metadata = edge.Metadata
	} else {
		// Add new edge
		edge.Timestamp = time.Now()
		g.edges[edge.ID] = edge

		// Update adjacency lists
		g.outEdges[edge.Source] = append(g.outEdges[edge.Source], edge.ID)
		g.inEdges[edge.Target] = append(g.inEdges[edge.Target], edge.ID)

		g.stats.EdgeCount++
	}

	g.stats.LastUpdateAt = time.Now()
	return nil
}

// GetNode retrieves a node by ID
func (g *Graph) GetNode(nodeID string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.nodes[nodeID]
}

// GetEdge retrieves an edge by ID
func (g *Graph) GetEdge(edgeID string) *Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.edges[edgeID]
}

// GetOutEdges returns all outgoing edges from a node
func (g *Graph) GetOutEdges(nodeID string) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edgeIDs := g.outEdges[nodeID]
	edges := make([]*Edge, 0, len(edgeIDs))
	for _, edgeID := range edgeIDs {
		if edge, exists := g.edges[edgeID]; exists {
			edges = append(edges, edge)
		}
	}
	return edges
}

// GetInEdges returns all incoming edges to a node
func (g *Graph) GetInEdges(nodeID string) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edgeIDs := g.inEdges[nodeID]
	edges := make([]*Edge, 0, len(edgeIDs))
	for _, edgeID := range edgeIDs {
		if edge, exists := g.edges[edgeID]; exists {
			edges = append(edges, edge)
		}
	}
	return edges
}

// RemoveExpiredEdges removes all expired edges from the graph
func (g *Graph) RemoveExpiredEdges() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	removed := 0
	for edgeID, edge := range g.edges {
		if edge.IsExpired() {
			// Remove from adjacency lists
			g.removeEdgeFromAdjacencyLists(edge)
			// Remove from edges map
			delete(g.edges, edgeID)
			g.stats.EdgeCount--
			removed++
		}
	}

	if removed > 0 {
		g.stats.LastUpdateAt = time.Now()
	}

	return removed
}

func (g *Graph) removeEdgeFromAdjacencyLists(edge *Edge) {
	// Remove from outEdges
	if outList, exists := g.outEdges[edge.Source]; exists {
		filtered := make([]string, 0, len(outList))
		for _, id := range outList {
			if id != edge.ID {
				filtered = append(filtered, id)
			}
		}
		g.outEdges[edge.Source] = filtered
	}

	// Remove from inEdges
	if inList, exists := g.inEdges[edge.Target]; exists {
		filtered := make([]string, 0, len(inList))
		for _, id := range inList {
			if id != edge.ID {
				filtered = append(filtered, id)
			}
		}
		g.inEdges[edge.Target] = filtered
	}
}

// GetAllNodes returns all nodes in the graph
func (g *Graph) GetAllNodes() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]*Node, 0, len(g.nodes))
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNodesByType returns all nodes of a specific type
func (g *Graph) GetNodesByType(nodeType NodeType) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]*Node, 0)
	for _, node := range g.nodes {
		if node.Type == nodeType {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GetCrownJewels returns all nodes marked as crown jewels (criticality >= 90)
func (g *Graph) GetCrownJewels() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	jewels := make([]*Node, 0)
	for _, node := range g.nodes {
		if node.Criticality >= 90 {
			jewels = append(jewels, node)
		}
	}
	return jewels
}

// Stats returns graph statistics
func (g *Graph) Stats() GraphStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.stats
}

// TenantID returns the tenant ID this graph belongs to
func (g *Graph) TenantID() string {
	return g.tenantID
}

// Clear removes all nodes and edges (used for full rebuild)
func (g *Graph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes = make(map[string]*Node)
	g.edges = make(map[string]*Edge)
	g.outEdges = make(map[string][]string)
	g.inEdges = make(map[string][]string)
	g.stats.NodeCount = 0
	g.stats.EdgeCount = 0
	g.stats.LastRebuildAt = time.Now()
	g.stats.LastUpdateAt = time.Now()
}
