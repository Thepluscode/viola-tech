package correlation

import (
  "crypto/sha256"
  "fmt"
  "io"
  "time"

  "github.com/oklog/ulid/v2"
)

type Bucket = time.Duration

const (
  Bucket5m  Bucket = 5 * time.Minute
  Bucket15m Bucket = 15 * time.Minute
  Bucket1h  Bucket = time.Hour
)

func GroupID(tenantID, ruleID, entityID string, observedAt time.Time, bucket Bucket) string {
  if bucket <= 0 {
    bucket = Bucket15m
  }
  bucketStart := observedAt.UTC().Truncate(time.Duration(bucket))

  seed := fmt.Sprintf("%s|%s|%s|%s", tenantID, ruleID, entityID, bucketStart.Format(time.RFC3339))
  sum := sha256.Sum256([]byte(seed))
  entropy := fixedEntropy(sum[:16])
  u := ulid.MustNew(ulid.Timestamp(bucketStart), entropy)
  return u.String()
}

type fixedEntropy []byte

// Read fills p with bytes cycled from f.
// Returns io.ErrUnexpectedEOF if f is empty to prevent a divide-by-zero panic
// and to surface a meaningful error to the caller (M1 fix).
func (f fixedEntropy) Read(p []byte) (int, error) {
  if len(f) == 0 {
    return 0, io.ErrUnexpectedEOF
  }
  for i := range p {
    p[i] = f[i%len(f)]
  }
  return len(p), nil
}

var _ io.Reader = fixedEntropy(nil)
