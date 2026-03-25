package rule

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Rule represents a detection rule
type Rule struct {
	ID          string      `yaml:"id"`
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version"`
	Severity    string      `yaml:"severity"`    // low|med|high|critical
	Confidence  float64     `yaml:"confidence"`  // 0.0 to 1.0
	Category    string      `yaml:"category"`
	EventType   string      `yaml:"event_type"`
	Description string      `yaml:"description"`
	MITRE       *MITRE      `yaml:"mitre,omitempty"`
	Conditions  []Condition `yaml:"match"` // renamed from Match to avoid collision with Match() method
	Threshold   *Threshold  `yaml:"threshold,omitempty"`
	SuppressIf  []Condition `yaml:"suppress_if,omitempty"`
	Tags        []string    `yaml:"tags,omitempty"`
	Metadata    *Metadata   `yaml:"metadata,omitempty"`

	// Compiled regex patterns (internal use)
	compiledRegex map[int]*regexp.Regexp
}

type MITRE struct {
	Tactic    string `yaml:"tactic"`
	Technique string `yaml:"technique"`
}

type Condition struct {
	Field    string   `yaml:"field"`
	Operator string   `yaml:"operator"`
	Value    string   `yaml:"value,omitempty"`
	Values   []string `yaml:"values,omitempty"`
}

type Threshold struct {
	Count   int    `yaml:"count"`
	Window  string `yaml:"window"`  // e.g., "60s", "5m", "1h"
	GroupBy string `yaml:"group_by"`

	// Parsed duration (internal)
	windowDuration time.Duration
}

type Metadata struct {
	Author     string   `yaml:"author"`
	Date       string   `yaml:"date"`
	References []string `yaml:"references,omitempty"`
}

// LoadRules loads all .yaml rules from a directory
func LoadRules(dir string) ([]*Rule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read rules dir: %w", err)
	}

	var rules []*Rule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		rule, err := LoadRule(path)
		if err != nil {
			// Log error but continue loading other rules
			fmt.Printf("WARN: failed to load rule %s: %v\n", path, err)
			continue
		}

		if err := rule.Validate(); err != nil {
			fmt.Printf("WARN: invalid rule %s: %v\n", path, err)
			continue
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// LoadRule loads a single rule from a YAML file
func LoadRule(path string) (*Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var rule Rule
	if err := yaml.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	// Compile regex patterns
	rule.compiledRegex = make(map[int]*regexp.Regexp)
	for i, cond := range rule.Conditions {
		if cond.Operator == "regex" {
			re, err := regexp.Compile(cond.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid regex in match[%d]: %w", i, err)
			}
			rule.compiledRegex[i] = re
		}
	}

	// Parse threshold window duration
	if rule.Threshold != nil {
		dur, err := time.ParseDuration(rule.Threshold.Window)
		if err != nil {
			return nil, fmt.Errorf("invalid threshold window: %w", err)
		}
		rule.Threshold.windowDuration = dur
	}

	return &rule, nil
}

// Validate checks if the rule is well-formed
func (r *Rule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("missing id")
	}
	if r.Name == "" {
		return fmt.Errorf("missing name")
	}
	if r.Version == "" {
		return fmt.Errorf("missing version")
	}
	if r.EventType == "" {
		return fmt.Errorf("missing event_type")
	}
	if len(r.Conditions) == 0 {
		return fmt.Errorf("missing match conditions")
	}

	// Validate severity
	switch r.Severity {
	case "low", "med", "high", "critical":
	default:
		return fmt.Errorf("invalid severity: %s (must be low|med|high|critical)", r.Severity)
	}

	// Validate confidence
	if r.Confidence < 0.0 || r.Confidence > 1.0 {
		return fmt.Errorf("confidence must be 0.0 to 1.0, got %f", r.Confidence)
	}

	// Validate operators
	for i, cond := range r.Conditions {
		if !isValidOperator(cond.Operator) {
			return fmt.Errorf("match[%d]: invalid operator %s", i, cond.Operator)
		}
		if cond.Field == "" {
			return fmt.Errorf("match[%d]: missing field", i)
		}
		if needsSingleValue(cond.Operator) && cond.Value == "" {
			return fmt.Errorf("match[%d]: operator %s requires 'value'", i, cond.Operator)
		}
		if needsMultipleValues(cond.Operator) && len(cond.Values) == 0 {
			return fmt.Errorf("match[%d]: operator %s requires 'values'", i, cond.Operator)
		}
	}

	// Validate threshold
	if r.Threshold != nil {
		if r.Threshold.Count <= 0 {
			return fmt.Errorf("threshold count must be > 0")
		}
		if r.Threshold.GroupBy == "" {
			return fmt.Errorf("threshold requires group_by")
		}
	}

	return nil
}

func isValidOperator(op string) bool {
	valid := []string{
		"equals", "equals_any", "contains", "contains_any",
		"startswith", "endswith", "regex",
		"not_equals", "not_in",
		"greater_than", "less_than",
	}
	for _, v := range valid {
		if op == v {
			return true
		}
	}
	return false
}

func needsSingleValue(op string) bool {
	return op == "equals" || op == "contains" || op == "startswith" || op == "endswith" ||
		op == "regex" || op == "not_equals" || op == "greater_than" || op == "less_than"
}

func needsMultipleValues(op string) bool {
	return op == "equals_any" || op == "contains_any" || op == "not_in"
}
