package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/viola/cloud-connector/internal/normalizer"
)

// CloudTrailConfig configures the CloudTrail poller.
type CloudTrailConfig struct {
	Region       string
	PollInterval time.Duration
	LookbackMin  int
	TenantID     string
}

// CloudTrailPoller polls AWS CloudTrail for new events.
type CloudTrailPoller struct {
	cfg  CloudTrailConfig
	norm *normalizer.Normalizer

	lastPoll   time.Time
	eventCount int64
	errorCount int64
	running    bool
}

// NewCloudTrailPoller creates a poller.
func NewCloudTrailPoller(cfg CloudTrailConfig, norm *normalizer.Normalizer) *CloudTrailPoller {
	return &CloudTrailPoller{cfg: cfg, norm: norm}
}

// Run starts the polling loop.
func (p *CloudTrailPoller) Run(ctx context.Context) {
	p.running = true
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	// Initial poll
	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			p.running = false
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *CloudTrailPoller) poll(ctx context.Context) {
	p.lastPoll = time.Now()

	// In production, this uses the AWS SDK:
	//   cfg, _ := config.LoadDefaultConfig(ctx, config.WithRegion(p.cfg.Region))
	//   client := cloudtrail.NewFromConfig(cfg)
	//   output, err := client.LookupEvents(ctx, &cloudtrail.LookupEventsInput{
	//       StartTime: aws.Time(time.Now().Add(-time.Duration(p.cfg.LookbackMin) * time.Minute)),
	//       EndTime:   aws.Time(time.Now()),
	//       MaxResults: aws.Int32(50),
	//   })
	//
	// For now, we provide the integration structure and event normalization.

	events := p.fetchEvents(ctx)
	for _, event := range events {
		if err := p.norm.Publish(ctx, event); err != nil {
			p.errorCount++
			fmt.Printf("cloud-connector: publish error: %v\n", err)
			continue
		}
		p.eventCount++
	}
}

// fetchEvents retrieves CloudTrail events. In production this calls the AWS API.
// This stub demonstrates the normalization pipeline.
func (p *CloudTrailPoller) fetchEvents(ctx context.Context) []normalizer.CloudEvent {
	// Stub: In production, iterate over CloudTrail LookupEvents response.
	// Each CloudTrail event is normalized to a CloudEvent.
	//
	// Example normalization for a real CloudTrail event:
	//   CloudTrailEvent{
	//     EventName: "AssumeRole",
	//     EventSource: "sts.amazonaws.com",
	//     Username: "user@company.com",
	//     Resources: [{ARN: "arn:aws:iam::123:role/admin"}],
	//   }
	//   →
	//   CloudEvent{
	//     EventType: "cloud_assume_role",
	//     Principal: "user@company.com",
	//     Resource:  "arn:aws:iam::123:role/admin",
	//     Action:    "AssumeRole",
	//   }

	return nil
}

// NormalizeCloudTrailEvent converts a raw CloudTrail event to a CloudEvent.
// This is the production normalization function.
func NormalizeCloudTrailEvent(tenantID string, raw json.RawMessage) (*normalizer.CloudEvent, error) {
	var ct struct {
		EventName    string `json:"eventName"`
		EventSource  string `json:"eventSource"`
		EventTime    string `json:"eventTime"`
		AWSRegion    string `json:"awsRegion"`
		SourceIP     string `json:"sourceIPAddress"`
		UserIdentity struct {
			Type        string `json:"type"`
			ARN         string `json:"arn"`
			AccountID   string `json:"accountId"`
			PrincipalID string `json:"principalId"`
			UserName    string `json:"userName"`
		} `json:"userIdentity"`
		Resources []struct {
			ARN  string `json:"ARN"`
			Type string `json:"resourceType"`
		} `json:"resources"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
	}

	if err := json.Unmarshal(raw, &ct); err != nil {
		return nil, fmt.Errorf("unmarshal cloudtrail event: %w", err)
	}

	observedAt, _ := time.Parse(time.RFC3339, ct.EventTime)
	if observedAt.IsZero() {
		observedAt = time.Now()
	}

	// Determine entity (primary resource ARN)
	entityID := ct.UserIdentity.ARN
	if len(ct.Resources) > 0 {
		entityID = ct.Resources[0].ARN
	}

	// Normalize event type
	eventType := normalizeEventName(ct.EventName, ct.EventSource)

	// Determine result
	result := "success"
	if ct.ErrorCode != "" {
		result = "failure"
	}

	labels := map[string]string{
		"source_ip":    ct.SourceIP,
		"account_id":   ct.UserIdentity.AccountID,
		"event_source": ct.EventSource,
	}
	if ct.ErrorCode != "" {
		labels["error_code"] = ct.ErrorCode
	}

	return &normalizer.CloudEvent{
		TenantID:   tenantID,
		EntityID:   entityID,
		EventType:  eventType,
		ObservedAt: observedAt,
		Source:     "aws-cloudtrail",
		Provider:   "aws",
		Region:     ct.AWSRegion,
		Principal:  ct.UserIdentity.UserName,
		Action:     ct.EventName,
		Resource:   entityID,
		Result:     result,
		RawPayload: raw,
		Labels:     labels,
	}, nil
}

// normalizeEventName maps AWS API actions to Viola event types.
func normalizeEventName(name, source string) string {
	// High-risk actions get specific types
	highRisk := map[string]string{
		"AssumeRole":                      "cloud_assume_role",
		"ConsoleLogin":                    "cloud_console_login",
		"CreateUser":                      "cloud_user_created",
		"DeleteUser":                      "cloud_user_deleted",
		"AttachUserPolicy":                "cloud_policy_attached",
		"AttachRolePolicy":                "cloud_policy_attached",
		"CreateAccessKey":                 "cloud_access_key_created",
		"PutBucketPolicy":                 "cloud_bucket_policy_changed",
		"AuthorizeSecurityGroupIngress":   "cloud_sg_rule_added",
		"RevokeSecurityGroupIngress":      "cloud_sg_rule_removed",
		"CreateSecurityGroup":             "cloud_sg_created",
		"RunInstances":                    "cloud_instance_launched",
		"TerminateInstances":              "cloud_instance_terminated",
		"StopLogging":                     "cloud_logging_disabled",
		"DeleteTrail":                     "cloud_trail_deleted",
		"PutBucketAcl":                    "cloud_bucket_acl_changed",
		"CreateLoginProfile":              "cloud_login_profile_created",
		"UpdateLoginProfile":              "cloud_login_profile_updated",
		"DisableKey":                      "cloud_kms_key_disabled",
		"ScheduleKeyDeletion":             "cloud_kms_key_scheduled_deletion",
	}

	if eventType, ok := highRisk[name]; ok {
		return eventType
	}

	// Generic categorization
	switch {
	case contains(name, "Create", "Put", "Run", "Start", "Launch"):
		return "cloud_resource_created"
	case contains(name, "Delete", "Remove", "Terminate", "Stop"):
		return "cloud_resource_deleted"
	case contains(name, "Modify", "Update", "Change", "Set"):
		return "cloud_resource_modified"
	case contains(name, "Get", "Describe", "List", "Lookup"):
		return "cloud_api_read"
	default:
		return "cloud_api_call"
	}
}

func contains(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// IsRunning returns whether the poller is active.
func (p *CloudTrailPoller) IsRunning() bool { return p.running }

// EventCount returns total events published.
func (p *CloudTrailPoller) EventCount() int64 { return p.eventCount }

// ErrorCount returns total errors.
func (p *CloudTrailPoller) ErrorCount() int64 { return p.errorCount }

// LastPoll returns the time of the last poll.
func (p *CloudTrailPoller) LastPoll() time.Time { return p.lastPoll }
