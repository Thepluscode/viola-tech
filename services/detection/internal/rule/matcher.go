package rule

import (
	"strconv"
	"strings"
)

// Event represents a normalized telemetry event (simplified for matching)
type Event struct {
	TenantID  string
	EntityID  string
	EventType string
	Fields    map[string]string // Flattened event fields
}

// Match checks if an event matches all rule conditions
func (r *Rule) Match(event *Event) bool {
	// Check event type
	if event.EventType != r.EventType {
		return false
	}

	// Check all match conditions (AND logic)
	for i, cond := range r.Conditions {
		if !r.matchCondition(event, cond, i) {
			return false
		}
	}

	// Check suppression conditions (if any match, suppress the detection)
	for _, cond := range r.SuppressIf {
		if r.matchCondition(event, cond, -1) { // -1 = no compiled regex for suppress
			return false
		}
	}

	return true
}

func (r *Rule) matchCondition(event *Event, cond Condition, regexIndex int) bool {
	fieldValue, exists := event.Fields[cond.Field]
	if !exists {
		// Field not present in event
		return false
	}

	switch cond.Operator {
	case "equals":
		return fieldValue == cond.Value

	case "equals_any":
		for _, v := range cond.Values {
			if fieldValue == v {
				return true
			}
		}
		return false

	case "contains":
		return strings.Contains(fieldValue, cond.Value)

	case "contains_any":
		for _, v := range cond.Values {
			if strings.Contains(fieldValue, v) {
				return true
			}
		}
		return false

	case "startswith":
		return strings.HasPrefix(fieldValue, cond.Value)

	case "endswith":
		return strings.HasSuffix(fieldValue, cond.Value)

	case "regex":
		if regexIndex < 0 || r.compiledRegex[regexIndex] == nil {
			return false
		}
		return r.compiledRegex[regexIndex].MatchString(fieldValue)

	case "not_equals":
		return fieldValue != cond.Value

	case "not_in":
		for _, v := range cond.Values {
			if fieldValue == v {
				return false
			}
		}
		return true

	case "greater_than":
		num, err := strconv.ParseFloat(fieldValue, 64)
		if err != nil {
			return false
		}
		target, err := strconv.ParseFloat(cond.Value, 64)
		if err != nil {
			return false
		}
		return num > target

	case "less_than":
		num, err := strconv.ParseFloat(fieldValue, 64)
		if err != nil {
			return false
		}
		target, err := strconv.ParseFloat(cond.Value, 64)
		if err != nil {
			return false
		}
		return num < target

	default:
		return false
	}
}
