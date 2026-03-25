package kafka

import (
  "errors"
  "time"
)

const (
  HdrTenantID  = "x-tenant-id"
  HdrRequestID = "x-request-id"
  HdrSource    = "x-source"
  HdrSchema    = "x-schema"
  HdrEmittedAt = "x-emitted-at"
)

var RequiredHeaders = []string{HdrTenantID, HdrRequestID, HdrSource, HdrSchema, HdrEmittedAt}

func ValidateHeaders(h map[string]string) error {
  for _, k := range RequiredHeaders {
    if h[k] == "" {
      return errors.New("missing required header: " + k)
    }
  }
  if _, err := time.Parse(time.RFC3339, h[HdrEmittedAt]); err != nil {
    return errors.New("invalid x-emitted-at (RFC3339 required)")
  }
  return nil
}
