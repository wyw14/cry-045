package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ReceiptStep struct {
	Name string
	Run  func(context.Context) error
}

type SubmissionReceipt struct {
	RequestID string
	Step      string
	Actor     string
	CreatedAt time.Time
}

type ReceiptLedger struct {
	mu       sync.RWMutex
	receipts []SubmissionReceipt
	clock    Clock
}

func NewReceiptLedger(clock Clock) *ReceiptLedger {
	if clock == nil {
		clock = RealClock{}
	}
	return &ReceiptLedger{clock: clock}
}

func (l *ReceiptLedger) Submit(ctx context.Context, requestID, actor string, steps []ReceiptStep) error {
	if err := validateReceiptSubmission(requestID, actor, steps); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	staged := make([]SubmissionReceipt, 0, len(steps))
	seen := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := strings.TrimSpace(step.Name)
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate receipt step %q", name)
		}
		seen[name] = struct{}{}
		if err := step.Run(ctx); err != nil {
			return fmt.Errorf("receipt step %s: %w", name, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		staged = append(staged, SubmissionReceipt{
			RequestID: requestID,
			Step:      name,
			Actor:     actor,
			CreatedAt: l.now(),
		})
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	l.receipts = append(l.receipts, staged...)
	return nil
}

func validateReceiptSubmission(requestID, actor string, steps []ReceiptStep) error {
	if strings.TrimSpace(requestID) == "" {
		return errors.New("request id is required")
	}
	if strings.TrimSpace(actor) == "" {
		return errors.New("actor is required")
	}
	if len(steps) == 0 {
		return errors.New("at least one receipt step is required")
	}
	for _, step := range steps {
		if strings.TrimSpace(step.Name) == "" || step.Run == nil {
			return errors.New("receipt step name and runner are required")
		}
	}
	return nil
}

func (l *ReceiptLedger) now() time.Time {
	if l.clock == nil {
		return time.Now().UTC()
	}
	return l.clock.Now().UTC()
}

func (l *ReceiptLedger) Snapshot(requestID string) []SubmissionReceipt {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]SubmissionReceipt, 0)
	for _, receipt := range l.receipts {
		if receipt.RequestID == requestID {
			out = append(out, receipt)
		}
	}
	return append([]SubmissionReceipt(nil), out...)
}
