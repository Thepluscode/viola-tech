// Package executor implements response action execution.
//
// The Executor interface is intentionally thin: it receives an action request
// and returns an error. Concrete implementations can call an EDR agent API,
// cloud provider firewall API, SOAR webhook, etc.
//
// The LogExecutor bundled here records actions to the structured logger and
// marks them as succeeded — useful for dev/demo where no real EDR is present.
package executor

import (
	"context"
	"fmt"
	"log"
)

// Request describes a single response action to execute.
type Request struct {
	ActionID   string
	TenantID   string
	ActionType string // "isolate_host" | "block_ip" | "kill_process" | "contain_user"
	Target     string // hostname, IP, process name, username, etc.
	Reason     string
}

// Executor executes a response action and returns an error on failure.
type Executor interface {
	Execute(ctx context.Context, req Request) error
}

// LogExecutor logs the action and reports success.
// Used in dev / when no real execution backend is configured.
type LogExecutor struct{}

func (LogExecutor) Execute(_ context.Context, req Request) error {
	log.Printf("[RESPONSE] action_id=%s type=%s target=%q tenant=%s reason=%q → SIMULATED OK",
		req.ActionID, req.ActionType, req.Target, req.TenantID, req.Reason)
	return nil
}

// ValidateType returns an error if the action type is unknown.
func ValidateType(actionType string) error {
	switch actionType {
	case "isolate_host", "block_ip", "kill_process", "contain_user":
		return nil
	default:
		return fmt.Errorf("unknown action type %q; valid: isolate_host, block_ip, kill_process, contain_user", actionType)
	}
}
