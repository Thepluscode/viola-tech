package kafka

import (
  "context"
  "time"
)

type Backoff struct {
  Base time.Duration
  Max  time.Duration
}

func (b Backoff) Sleep(ctx context.Context, attempt int) error {
  if attempt < 0 {
    attempt = 0
  }
  d := b.Base * time.Duration(1<<attempt)
  if d > b.Max {
    d = b.Max
  }
  t := time.NewTimer(d)
  defer t.Stop()

  select {
  case <-ctx.Done():
    return ctx.Err()
  case <-t.C:
    return nil
  }
}
