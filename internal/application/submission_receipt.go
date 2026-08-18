package application

import (
	"context"
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
}

func NewReceiptLedger(Clock) *ReceiptLedger { return &ReceiptLedger{} }

func (l *ReceiptLedger) Submit(ctx context.Context, requestID, actor string, steps []ReceiptStep) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	pending := make([]SubmissionReceipt, 0, len(steps))
	for _, step := range steps {
		if err := step.Run(ctx); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		pending = append(pending, SubmissionReceipt{
			RequestID: requestID,
			Step:      step.Name,
			Actor:     actor,
			CreatedAt: time.Now().UTC(),
		})
	}
	l.mu.Lock()
	l.receipts = append(l.receipts, pending...)
	l.mu.Unlock()
	return nil
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
	return out
}
