package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// AuditChain protects the chronological review narrative. Every event carries
// the previous digest, making later edits detectable without a remote service.
type AuditChain struct {
	Events []AuditEvent
	Head   string
}

type AuditCheckpoint struct {
	RequestID string
	Revision  int
	Status    SelectionStatus
	Actor     string
	Digest    string
	CreatedAt time.Time
}

func (c *AuditChain) Append(event AuditEvent) error {
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.RequestID) == "" {
		return fmt.Errorf("audit identity is required")
	}
	if len(c.Events) > 0 && event.CreatedAt.Before(c.Events[len(c.Events)-1].CreatedAt) {
		return fmt.Errorf("audit event is out of order")
	}
	event.Hash = c.nextDigest(event)
	c.Events = append(c.Events, event)
	c.Head = event.Hash
	return nil
}

func (c AuditChain) nextDigest(event AuditEvent) string {
	payload, _ := json.Marshal(struct {
		Previous string
		Event    AuditEvent
	}{Previous: c.Head, Event: event})
	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:])
}

func (c AuditChain) Verify() error {
	previous := ""
	for index, event := range c.Events {
		payload, _ := json.Marshal(struct {
			Previous string
			Event    AuditEvent
		}{Previous: previous, Event: event})
		h := sha256.Sum256(payload)
		expected := hex.EncodeToString(h[:])
		if event.Hash != expected {
			return fmt.Errorf("audit hash mismatch at %d", index)
		}
		if index > 0 && event.CreatedAt.Before(c.Events[index-1].CreatedAt) {
			return fmt.Errorf("audit order mismatch at %d", index)
		}
		previous = event.Hash
	}
	if previous != c.Head {
		return fmt.Errorf("audit head mismatch")
	}
	return nil
}

func (c AuditChain) Checkpoints() []AuditCheckpoint {
	checkpoints := make([]AuditCheckpoint, 0, len(c.Events))
	for _, event := range c.Events {
		var after struct {
			ID       string
			Revision int
			Status   SelectionStatus
		}
		if json.Unmarshal([]byte(event.After), &after) != nil {
			continue
		}
		checkpoints = append(checkpoints, AuditCheckpoint{RequestID: event.RequestID, Revision: after.Revision, Status: after.Status, Actor: event.ActorID, Digest: event.Hash, CreatedAt: event.CreatedAt})
	}
	sort.SliceStable(checkpoints, func(i, j int) bool { return checkpoints[i].CreatedAt.Before(checkpoints[j].CreatedAt) })
	return checkpoints
}

func (c AuditChain) ChangesBetween(from, to time.Time) []AuditEvent {
	result := make([]AuditEvent, 0)
	for _, event := range c.Events {
		if !event.CreatedAt.Before(from) && event.CreatedAt.Before(to) {
			result = append(result, event)
		}
	}
	return result
}

func (c AuditChain) Actors() []string {
	set := map[string]bool{}
	for _, event := range c.Events {
		if event.ActorID != "" {
			set[event.ActorID] = true
		}
	}
	actors := make([]string, 0, len(set))
	for actor := range set {
		actors = append(actors, actor)
	}
	sort.Strings(actors)
	return actors
}

func (c AuditChain) Summary() map[string]int {
	result := map[string]int{}
	for _, event := range c.Events {
		result[event.Action]++
	}
	return result
}

func (c AuditChain) Clone() AuditChain {
	return AuditChain{Events: append([]AuditEvent(nil), c.Events...), Head: c.Head}
}

func BuildCheckpoint(event AuditEvent) (AuditCheckpoint, error) {
	var after struct {
		ID       string
		Revision int
		Status   SelectionStatus
	}
	if err := json.Unmarshal([]byte(event.After), &after); err != nil {
		return AuditCheckpoint{}, err
	}
	return AuditCheckpoint{RequestID: event.RequestID, Revision: after.Revision, Status: after.Status, Actor: event.ActorID, Digest: event.Hash, CreatedAt: event.CreatedAt}, nil
}
