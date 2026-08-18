package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

type RevisionAnswer struct {
	RequestID string
	Revision  int
	IssueID   string
	Text      string
}

type RevisionAnswerBook struct {
	mu      sync.RWMutex
	answers map[string]RevisionAnswer
}

func NewRevisionAnswerBook() *RevisionAnswerBook {
	return &RevisionAnswerBook{answers: make(map[string]RevisionAnswer)}
}

func (b *RevisionAnswerBook) Apply(ctx context.Context, requestID string, revision int, answers []RevisionAnswer) error {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || revision < 1 {
		return errors.New("request id and positive revision are required")
	}
	if len(answers) == 0 {
		return errors.New("at least one issue answer is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	staged := make(map[string]RevisionAnswer, len(answers))
	for _, answer := range answers {
		if err := ctx.Err(); err != nil {
			return err
		}
		normalized, err := normalizeRevisionAnswer(requestID, revision, answer)
		if err != nil {
			return err
		}
		key := revisionAnswerKey(requestID, revision, normalized.IssueID)
		if _, duplicate := staged[key]; duplicate {
			return fmt.Errorf("duplicate issue answer %q", normalized.IssueID)
		}
		staged[key] = normalized
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	for key, answer := range staged {
		b.answers[key] = answer
	}
	return nil
}

func normalizeRevisionAnswer(requestID string, revision int, answer RevisionAnswer) (RevisionAnswer, error) {
	answer.RequestID = strings.TrimSpace(answer.RequestID)
	answer.IssueID = strings.TrimSpace(answer.IssueID)
	answer.Text = strings.TrimSpace(answer.Text)
	if answer.RequestID != requestID || answer.Revision != revision {
		return RevisionAnswer{}, errors.New("answer belongs to another request revision")
	}
	if answer.IssueID == "" {
		return RevisionAnswer{}, errors.New("issue id is required")
	}
	if answer.Text == "" {
		return RevisionAnswer{}, errors.New("answer text is required")
	}
	return answer, nil
}

func (b *RevisionAnswerBook) Get(requestID string, revision int, issueID string) (RevisionAnswer, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	answer, ok := b.answers[revisionAnswerKey(requestID, revision, issueID)]
	return answer, ok
}

func revisionAnswerKey(requestID string, revision int, issueID string) string {
	return fmt.Sprintf("%s\x00%d\x00%s", requestID, revision, issueID)
}

func (b *RevisionAnswerBook) Snapshot(requestID string, revision int) []RevisionAnswer {
	b.mu.RLock()
	defer b.mu.RUnlock()
	prefix := fmt.Sprintf("%s\x00%d\x00", requestID, revision)
	out := make([]RevisionAnswer, 0)
	for key, answer := range b.answers {
		if strings.HasPrefix(key, prefix) {
			out = append(out, answer)
		}
	}
	return out
}
