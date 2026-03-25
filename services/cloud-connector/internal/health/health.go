package health

import (
	"encoding/json"
	"net/http"

	"github.com/viola/cloud-connector/internal/providers/aws"
)

type Handler struct {
	poller *aws.CloudTrailPoller
}

func New(poller *aws.CloudTrailPoller) *Handler {
	return &Handler{poller: poller}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"running":     h.poller.IsRunning(),
		"events":      h.poller.EventCount(),
		"errors":      h.poller.ErrorCount(),
		"last_poll":   h.poller.LastPoll(),
	})
}
