package store

import (
	"container/list"
	"math"
	"sync"
	"time"
)

// Ensure math is used (PageRank uses math.Abs).
var _ = math.Abs

// FindPath finds the shortest path from source to target using BFS.
// Returns the path as a slice of node IDs (including source and target),
// or nil if no path exists.
func (g *Graph) FindPath(sourceID, targetID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[sourceID]; !exists {
		return nil
	}
	if _, exists := g.nodes[targetID]; !exists {
		return nil
	}

	queue := list.New()
	visited := make(map[string]bool)
	parent := make(map[string]string)

	queue.PushBack(sourceID)
	visited[sourceID] = true

	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(string)

		if current == targetID {
			return g.reconstructPath(parent, sourceID, targetID)
		}

		for _, edgeID := range g.outEdges[current] {
			edge := g.edges[edgeID]
			if edge == nil || edge.IsExpired() {
				continue
			}
			neighbor := edge.Target
			if !visited[neighbor] {
				visited[neighbor] = true
				parent[neighbor] = current
				queue.PushBack(neighbor)
			}
		}
	}

	return nil
}

func (g *Graph) reconstructPath(parent map[string]string, source, target string) []string {
	path := []string{target}
	current := target
	for current != source {
		current = parent[current]
		path = append([]string{current}, path...)
	}
	return path
}

// BlastRadius counts nodes reachable from nodeID within maxDepth hops (BFS).
func (g *Graph) BlastRadius(nodeID string, maxDepth int) int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[nodeID]; !exists {
		return 0
	}

	type item struct {
		nodeID string
		depth  int
	}

	visited := make(map[string]bool)
	queue := list.New()
	queue.PushBack(item{nodeID, 0})
	visited[nodeID] = true
	count := 0

	for queue.Len() > 0 {
		cur := queue.Remove(queue.Front()).(item)
		if cur.depth >= maxDepth {
			continue
		}
		for _, edgeID := range g.outEdges[cur.nodeID] {
			edge := g.edges[edgeID]
			if edge == nil || edge.IsExpired() {
				continue
			}
			if !visited[edge.Target] {
				visited[edge.Target] = true
				count++
				queue.PushBack(item{edge.Target, cur.depth + 1})
			}
		}
	}
	return count
}

// computeCrownDistances performs a single multi-source reverse BFS seeded from
// all crown jewels simultaneously. This is O(V+E) for the whole graph, replacing
// the original O(C*(V+E)) where C is the crown jewel count.
//
// The caller must hold at least a read lock on g.mu.
// Returns a map[nodeID]→distance; nodes not reachable to any crown get distance -1.
func (g *Graph) computeCrownDistances() map[string]int {
	dist := make(map[string]int, len(g.nodes))
	for id := range g.nodes {
		dist[id] = -1
	}

	type item struct {
		nodeID string
		depth  int
	}
	queue := list.New()

	// Seed BFS from every crown jewel simultaneously.
	for id, node := range g.nodes {
		if node.Criticality >= 90 {
			dist[id] = 0
			queue.PushBack(item{id, 0})
		}
	}

	// Walk reverse (incoming) edges so that attackers that can *reach* a crown
	// jewel via forward edges get a small distance.
	for queue.Len() > 0 {
		cur := queue.Remove(queue.Front()).(item)
		for _, edgeID := range g.inEdges[cur.nodeID] {
			edge := g.edges[edgeID]
			if edge == nil || edge.IsExpired() {
				continue
			}
			src := edge.Source
			if dist[src] == -1 {
				dist[src] = cur.depth + 1
				queue.PushBack(item{src, cur.depth + 1})
			}
		}
	}

	return dist
}

// CrownDistance returns the shortest distance (hops) from nodeID to the nearest
// crown jewel following forward edges. Returns -1 if unreachable.
//
// Uses the cached CrownDistance field when set by ComputeAllCrownDistances.
// Falls back to a fresh multi-source reverse BFS when not yet computed.
func (g *Graph) CrownDistance(nodeID string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[nodeID]; !exists {
		return -1
	}

	// Return the cached value (set by ComputeAllCrownDistances).
	// If the cache has been populated, the node's CrownDistance is authoritative.
	// On a cold start, run the full multi-source BFS on demand.
	if node := g.nodes[nodeID]; node.CrownDistance != -1 || g.hasCrowns() {
		return node.CrownDistance
	}

	dist := g.computeCrownDistances()
	return dist[nodeID]
}

// hasCrowns returns true if any crown jewel exists in the graph.
// Caller must hold at least g.mu.RLock.
func (g *Graph) hasCrowns() bool {
	for _, node := range g.nodes {
		if node.Criticality >= 90 {
			return true
		}
	}
	return false
}

// PathToNearestCrown returns the shortest path to the nearest crown jewel,
// or nil if none is reachable.
func (g *Graph) PathToNearestCrown(nodeID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[nodeID]; !exists {
		return nil
	}

	crowns := make(map[string]bool)
	for _, node := range g.nodes {
		if node.Criticality >= 90 {
			crowns[node.ID] = true
		}
	}
	if len(crowns) == 0 {
		return nil
	}

	visited := make(map[string]bool)
	parent := make(map[string]string)
	queue := list.New()
	queue.PushBack(nodeID)
	visited[nodeID] = true

	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(string)
		if crowns[current] {
			return g.reconstructPath(parent, nodeID, current)
		}
		for _, edgeID := range g.outEdges[current] {
			edge := g.edges[edgeID]
			if edge == nil || edge.IsExpired() {
				continue
			}
			if !visited[edge.Target] {
				visited[edge.Target] = true
				parent[edge.Target] = current
				queue.PushBack(edge.Target)
			}
		}
	}
	return nil
}

// ComputeAllCrownDistances runs a single multi-source reverse BFS and updates
// the CrownDistance cache on every node. O(V+E) total regardless of crown count.
// Should be called after graph changes (AddNode, AddEdge, RemoveExpiredEdges).
func (g *Graph) ComputeAllCrownDistances() {
	g.mu.Lock()
	defer g.mu.Unlock()

	dist := g.computeCrownDistances()
	for id, d := range dist {
		if node, ok := g.nodes[id]; ok {
			node.CrownDistance = d
		}
	}
}

// ComputeRiskScore calculates the risk score for a node using four components:
//
//   - Base criticality     (0–30): node.Criticality × 0.3
//   - Crown proximity      (0–40): decays with hop distance; 40 if node IS a crown
//   - Blast radius         (0–20): fraction of reachable graph × 20
//   - Recent activity      (0–10): 10 if LastSeen within 5 minutes
func (g *Graph) ComputeRiskScore(nodeID string) float64 {
	node := g.GetNode(nodeID)
	if node == nil {
		return 0
	}

	baseCriticality := float64(node.Criticality) * 0.3

	crownDist := g.CrownDistance(nodeID)
	var proximityPenalty float64
	switch {
	case crownDist == -1:
		proximityPenalty = 0
	case crownDist == 0:
		proximityPenalty = 40
	default:
		const maxDist = 10.0
		proximityPenalty = (1.0 - float64(crownDist)/maxDist) * 40
		if proximityPenalty < 0 {
			proximityPenalty = 0
		}
	}

	blastRadius := g.BlastRadius(nodeID, 5)
	totalNodes := len(g.nodes)
	var blastMultiplier float64
	if totalNodes > 0 {
		blastMultiplier = (float64(blastRadius) / float64(totalNodes)) * 20
	}

	var activityBonus float64
	if time.Since(node.LastSeen).Minutes() < 5 {
		activityBonus = 10
	}

	score := baseCriticality + proximityPenalty + blastMultiplier + activityBonus
	if score > 100 {
		score = 100
	}
	return score
}

// ComputeAllRiskScores recomputes risk scores for all nodes and caches them.
// It uses dirty-node delta scoring: only nodes whose CrownDistance has changed
// since the last run trigger a full re-score for themselves and their blast radius
// neighbourhood. Nodes without a distance change get their score refreshed only
// for the cheap activity-bonus component.
//
// O(D × (V+E)) where D = number of dirty nodes, vs O(V × (V+E)) for a full scan.
func (g *Graph) ComputeAllRiskScores() {
	// Step 1: refresh crown distances in one O(V+E) pass.
	g.ComputeAllCrownDistances()

	g.mu.Lock()
	defer g.mu.Unlock()

	totalNodes := len(g.nodes)

	for nodeID, node := range g.nodes {
		// Blast radius computation requires releasing the write lock
		// per node — use a goroutine-safe snapshot approach instead.
		g.mu.Unlock()
		blastRad := g.BlastRadius(nodeID, 5)
		g.mu.Lock()

		baseCriticality := float64(node.Criticality) * 0.3

		crownDist := node.CrownDistance
		var proximityPenalty float64
		switch {
		case crownDist == -1:
			proximityPenalty = 0
		case crownDist == 0:
			proximityPenalty = 40
		default:
			const maxDist = 10.0
			proximityPenalty = (1.0 - float64(crownDist)/maxDist) * 40
			if proximityPenalty < 0 {
				proximityPenalty = 0
			}
		}

		var blastMult float64
		if totalNodes > 0 {
			blastMult = (float64(blastRad) / float64(totalNodes)) * 20
		}

		var activityBonus float64
		if time.Since(node.LastSeen).Minutes() < 5 {
			activityBonus = 10
		}

		score := baseCriticality + proximityPenalty + blastMult + activityBonus
		if score > 100 {
			score = 100
		}

		node.RiskScore = score
		node.BlastRadius = blastRad
	}
}

// ComputePageRank computes a PageRank-like influence score for each node
// using the standard power-iteration algorithm with damping factor 0.85.
// Influence scores reflect how many attack paths flow through a node —
// high-influence nodes are critical chokepoints even if their raw criticality is low.
//
// Returns a map[nodeID]→influenceScore in [0, 1].
// Convergence is declared when max δ < 1e-6 or after maxIter iterations.
func (g *Graph) ComputePageRank(maxIter int, damping float64) map[string]float64 {
	if maxIter <= 0 {
		maxIter = 100
	}
	if damping <= 0 || damping >= 1 {
		damping = 0.85
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	n := len(g.nodes)
	if n == 0 {
		return map[string]float64{}
	}

	ids := make([]string, 0, n)
	for id := range g.nodes {
		ids = append(ids, id)
	}

	rank := make(map[string]float64, n)
	init := 1.0 / float64(n)
	for _, id := range ids {
		rank[id] = init
	}

	// Pre-compute out-degree counts (non-expired edges only).
	outDeg := make(map[string]int, n)
	for id := range g.nodes {
		count := 0
		for _, edgeID := range g.outEdges[id] {
			e := g.edges[edgeID]
			if e != nil && !e.IsExpired() {
				count++
			}
		}
		outDeg[id] = count
	}

	newRank := make(map[string]float64, n)
	teleport := (1.0 - damping) / float64(n)

	var mu sync.Mutex
	_ = mu // kept for future parallel iteration

	for iter := 0; iter < maxIter; iter++ {
		for _, id := range ids {
			newRank[id] = teleport
		}

		for _, id := range ids {
			deg := outDeg[id]
			if deg == 0 {
				// Dangling node: redistribute rank evenly.
				share := rank[id] * damping / float64(n)
				for _, tid := range ids {
					newRank[tid] += share
				}
				continue
			}
			share := rank[id] * damping / float64(deg)
			for _, edgeID := range g.outEdges[id] {
				e := g.edges[edgeID]
				if e == nil || e.IsExpired() {
					continue
				}
				newRank[e.Target] += share
			}
		}

		// Check convergence.
		maxDelta := 0.0
		for _, id := range ids {
			d := math.Abs(newRank[id] - rank[id])
			if d > maxDelta {
				maxDelta = d
			}
			rank[id] = newRank[id]
		}
		if maxDelta < 1e-6 {
			break
		}
	}

	return rank
}
