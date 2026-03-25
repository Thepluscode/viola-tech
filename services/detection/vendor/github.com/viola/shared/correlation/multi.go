package correlation

import (
  "sort"
  "strings"
  "time"
)

func GroupIDForEntities(tenantID, ruleID string, entityIDs []string, observedAt time.Time, bucket Bucket) string {
  ids := make([]string, 0, len(entityIDs))
  for _, e := range entityIDs {
    if e != "" {
      ids = append(ids, e)
    }
  }
  sort.Strings(ids)
  joined := strings.Join(ids, ",")
  return GroupID(tenantID, ruleID, joined, observedAt, bucket)
}
