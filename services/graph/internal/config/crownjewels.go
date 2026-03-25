package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/viola/graph/internal/store"
)

// CrownJewelConfig defines crown jewels per tenant
type CrownJewelConfig struct {
	Tenants map[string]TenantCrownJewels `yaml:"tenants"`
}

type TenantCrownJewels struct {
	CrownJewels []CrownJewel `yaml:"crown_jewels"`
}

type CrownJewel struct {
	ID          string `yaml:"id"`           // Node ID (e.g., "endpoint:dc-01")
	Reason      string `yaml:"reason"`       // Why it's a crown jewel
	Criticality int    `yaml:"criticality"`  // 0-100 (default: 100)
}

// LoadCrownJewels loads crown jewel configuration from a YAML file
func LoadCrownJewels(path string) (*CrownJewelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read crown jewels file: %w", err)
	}

	var config CrownJewelConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal crown jewels: %w", err)
	}

	// Set default criticality
	for tid, tenant := range config.Tenants {
		for i := range tenant.CrownJewels {
			if tenant.CrownJewels[i].Criticality == 0 {
				tenant.CrownJewels[i].Criticality = 100
			}
		}
		config.Tenants[tid] = tenant
	}

	return &config, nil
}

// ApplyToManager applies crown jewel configuration to a graph manager
func (c *CrownJewelConfig) ApplyToManager(manager *store.GraphManager) error {
	for tenantID, tenant := range c.Tenants {
		for _, jewel := range tenant.CrownJewels {
			// Get or create the node
			node := manager.GetNode(tenantID, jewel.ID)
			if node == nil {
				// Node doesn't exist yet, we'll create it with high criticality
				// When telemetry arrives, it will update LastSeen
				node = &store.Node{
					ID:          jewel.ID,
					Type:        inferNodeType(jewel.ID),
					Labels:      map[string]string{"crown_jewel": "true", "reason": jewel.Reason},
					Criticality: jewel.Criticality,
				}
				if err := manager.AddNode(tenantID, node); err != nil {
					return fmt.Errorf("add crown jewel node %s: %w", jewel.ID, err)
				}
			} else {
				// Update existing node criticality
				node.Criticality = jewel.Criticality
				node.Labels["crown_jewel"] = "true"
				node.Labels["reason"] = jewel.Reason
			}
		}
	}
	return nil
}

func inferNodeType(nodeID string) store.NodeType {
	// Simple inference based on ID prefix
	if len(nodeID) > 5 {
		switch nodeID[:5] {
		case "user:":
			return store.NodeTypeUser
		case "endpo": // "endpoint:"
			return store.NodeTypeEndpoint
		case "servi": // "service:"
			return store.NodeTypeService
		case "cloud":
			return store.NodeTypeCloud
		}
	}
	return store.NodeTypeEndpoint // Default
}
