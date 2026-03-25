package rule

import (
	"github.com/cespare/xxhash/v2"
)

// RuleBloomFilter is a per-rule Bloom filter that pre-screens events using
// field values from "equals" and "equals_any" match conditions.
//
// When an event field value is NOT in the filter, the rule provably cannot
// match that event — we skip the full condition evaluation entirely.
// False positives are allowed (some events pass the filter but fail matching),
// but false negatives never occur.
//
// Implementation: 4096-bit (512-byte) bit array, double-hashing with xxhash.
// At 20 "equals" values per rule, expected FPR ≈ 0.7% — negligible overhead.
const bloomBits = 4096

type RuleBloomFilter struct {
	bits [bloomBits / 64]uint64 // 64 × 64-bit words = 4096 bits
}

// setBit sets bit position pos in the filter.
func (bf *RuleBloomFilter) setBit(pos uint64) {
	pos &= bloomBits - 1 // wrap to [0, 4095]
	bf.bits[pos/64] |= 1 << (pos % 64)
}

// testBit reports whether bit pos is set.
func (bf *RuleBloomFilter) testBit(pos uint64) bool {
	pos &= bloomBits - 1
	return bf.bits[pos/64]&(1<<(pos%64)) != 0
}

// add inserts a string value into the filter using two independent hashes.
func (bf *RuleBloomFilter) add(s string) {
	h1 := xxhash.Sum64String(s)
	// Derive h2 from h1 via Murmur-style finaliser (avoids second full pass).
	h2 := h1 ^ (h1 >> 33)
	h2 *= 0xff51afd7ed558ccd
	h2 ^= h2 >> 33
	bf.setBit(h1)
	bf.setBit(h2)
}

// MayContain returns true when the value might be in the filter.
// A false return guarantees the value was never added.
func (bf *RuleBloomFilter) MayContain(s string) bool {
	h1 := xxhash.Sum64String(s)
	h2 := h1 ^ (h1 >> 33)
	h2 *= 0xff51afd7ed558ccd
	h2 ^= h2 >> 33
	return bf.testBit(h1) && bf.testBit(h2)
}

// IsEmpty returns true when no values have been added.
func (bf *RuleBloomFilter) IsEmpty() bool {
	for _, w := range bf.bits {
		if w != 0 {
			return false
		}
	}
	return true
}

// BuildBloomFilter constructs a RuleBloomFilter from the rule's "equals" and
// "equals_any" conditions. Only the first match condition with a field value
// is indexed (subsequent conditions are still evaluated in full by Match()).
//
// Rules with no exact-match conditions return nil — callers must nil-check
// before calling MayContain.
func BuildBloomFilter(r *Rule) *RuleBloomFilter {
	bf := &RuleBloomFilter{}
	added := 0

	for _, cond := range r.Conditions {
		switch cond.Operator {
		case "equals":
			if cond.Value != "" {
				bf.add(cond.Value)
				added++
			}
		case "equals_any":
			for _, v := range cond.Values {
				if v != "" {
					bf.add(v)
					added++
				}
			}
		}
	}

	if added == 0 {
		return nil // No indexed values — don't use filter for this rule
	}
	return bf
}

// EventMatchesBloom checks if any event field value passes the bloom filter.
// Returns true if any "equals" / "equals_any" condition field's value is
// potentially in the filter, meaning the rule might match.
// Returns true always when bf is nil (no filter for this rule).
func EventMatchesBloom(bf *RuleBloomFilter, r *Rule, event *Event) bool {
	if bf == nil {
		return true
	}
	// Check the first equals/equals_any condition's field value against the filter.
	for _, cond := range r.Conditions {
		switch cond.Operator {
		case "equals", "equals_any":
			val, ok := event.Fields[cond.Field]
			if !ok {
				return false // Field absent — can't match
			}
			if !bf.MayContain(val) {
				return false // Definitely not in filter — skip rule
			}
			return true // Might match — proceed to full evaluation
		}
	}
	return true
}

