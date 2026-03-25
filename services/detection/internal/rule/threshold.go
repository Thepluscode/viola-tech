package rule

import (
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
)

// numShards is the number of independent shard maps.
// Power of 2 allows cheap masking: shard = hash & (numShards-1).
// 256 shards means that under 256× parallelism each shard has on average
// one concurrent writer — effectively eliminating lock contention.
const numShards = 256

// ThresholdTracker tracks event counts over time windows for threshold-based detection.
// It uses a 256-shard design so concurrent goroutines (one per Kafka partition)
// each operate on independent shards rather than contending on a single global lock.
type ThresholdTracker struct {
	shards [numShards]thresholdShard
}

type thresholdShard struct {
	mu     sync.Mutex
	counts map[string]*countWindow
	_      [56]byte // pad to 64 bytes to prevent false cache-line sharing
}

type countWindow struct {
	events []time.Time
}

// NewThresholdTracker creates a sharded tracker and starts background cleanup.
func NewThresholdTracker() *ThresholdTracker {
	t := &ThresholdTracker{}
	for i := range t.shards {
		t.shards[i].counts = make(map[string]*countWindow)
	}
	go t.cleanup()
	return t
}

// shardFor returns the shard index for a given key using xxhash.
// xxhash is 5-10× faster than FNV for strings and already in the dependency tree.
func shardFor(key string) uint64 {
	return xxhash.Sum64String(key) & (numShards - 1)
}

// Check records the event occurrence and returns true when the configured
// threshold is met within the time window.
func (t *ThresholdTracker) Check(rule *Rule, event *Event) bool {
	if rule.Threshold == nil {
		return true // No threshold — always fire
	}

	groupValue := event.Fields[rule.Threshold.GroupBy]
	if groupValue == "" {
		groupValue = event.EntityID
	}
	key := rule.ID + ":" + event.TenantID + ":" + groupValue

	shard := &t.shards[shardFor(key)]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	cw, exists := shard.counts[key]
	if !exists {
		cw = &countWindow{}
		shard.counts[key] = cw
	}

	now := time.Now()
	cutoff := now.Add(-rule.Threshold.windowDuration)

	// Filter in-place: slide window forward.
	valid := 0
	for _, ts := range cw.events {
		if ts.After(cutoff) {
			cw.events[valid] = ts
			valid++
		}
	}
	cw.events = cw.events[:valid]

	cw.events = append(cw.events, now)
	return len(cw.events) >= rule.Threshold.Count
}

// cleanup evicts windows with no activity in the past hour.
// Runs every 5 minutes across all shards without a global lock.
func (t *ThresholdTracker) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for i := range t.shards {
			shard := &t.shards[i]
			shard.mu.Lock()
			for key, cw := range shard.counts {
				if len(cw.events) == 0 || now.Sub(cw.events[len(cw.events)-1]) > time.Hour {
					delete(shard.counts, key)
				}
			}
			shard.mu.Unlock()
		}
	}
}
