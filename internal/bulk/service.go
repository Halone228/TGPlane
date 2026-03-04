// Package bulk executes the same operation across many sessions concurrently.
package bulk

import (
	"context"
	"fmt"
	"sync"

	"github.com/tgplane/tgplane/internal/account"
	"github.com/tgplane/tgplane/internal/bot"
	"github.com/tgplane/tgplane/internal/worker/manager"
)

const defaultConcurrency = 10

// Service runs bulk operations using the underlying account/bot services and worker manager.
type Service struct {
	accountSvc  *account.Service
	botSvc      *bot.Service
	workerMgr   *manager.Manager
	concurrency int
}

func NewService(
	accountSvc *account.Service,
	botSvc *bot.Service,
	workerMgr *manager.Manager,
) *Service {
	return &Service{
		accountSvc:  accountSvc,
		botSvc:      botSvc,
		workerMgr:   workerMgr,
		concurrency: defaultConcurrency,
	}
}

// --- result types ---

// ItemResult holds the outcome for one item in a bulk request.
type ItemResult struct {
	Index int         `json:"index"`
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// BulkResult is the top-level response for every bulk endpoint.
type BulkResult struct {
	Total     int          `json:"total"`
	Succeeded int          `json:"succeeded"`
	Failed    int          `json:"failed"`
	Items     []ItemResult `json:"items"`
}

// --- request types ---

type AddBotRequest struct {
	Token string `json:"token" binding:"required"`
}

type AddAccountRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// --- operations ---

// AddBots creates bot records and assigns each to the least-loaded worker in parallel.
func (s *Service) AddBots(ctx context.Context, reqs []AddBotRequest) BulkResult {
	results := make([]ItemResult, len(reqs))
	s.parallel(len(reqs), func(i int) {
		b, err := s.botSvc.Add(ctx, bot.CreateRequest{Token: reqs[i].Token})
		if err != nil {
			results[i] = ItemResult{Index: i, Error: err.Error()}
			return
		}
		if s.workerMgr != nil {
			if _, err := s.workerMgr.AssignBot(ctx, b.SessionID, b.Token); err != nil {
				results[i] = ItemResult{Index: i, Error: err.Error()}
				return
			}
		}
		results[i] = ItemResult{Index: i, OK: true, Data: b}
	})
	return summarize(results)
}

// AddAccounts creates account records and assigns each to the least-loaded worker in parallel.
func (s *Service) AddAccounts(ctx context.Context, reqs []AddAccountRequest) BulkResult {
	results := make([]ItemResult, len(reqs))
	s.parallel(len(reqs), func(i int) {
		a, err := s.accountSvc.Add(ctx, account.CreateRequest{Phone: reqs[i].Phone})
		if err != nil {
			results[i] = ItemResult{Index: i, Error: err.Error()}
			return
		}
		if s.workerMgr != nil {
			if _, err := s.workerMgr.AssignAccount(ctx, a.SessionID, a.Phone); err != nil {
				results[i] = ItemResult{Index: i, Error: err.Error()}
				return
			}
		}
		results[i] = ItemResult{Index: i, OK: true, Data: a}
	})
	return summarize(results)
}

// RemoveSessions removes multiple sessions by session_id.
// It resolves the worker_id via the bot/account services.
func (s *Service) RemoveSessions(ctx context.Context, sessionIDs []string) BulkResult {
	results := make([]ItemResult, len(sessionIDs))
	s.parallel(len(sessionIDs), func(i int) {
		sid := sessionIDs[i]

		workerID, err := s.resolveWorkerID(ctx, sid)
		if err != nil {
			results[i] = ItemResult{Index: i, Error: err.Error()}
			return
		}
		if workerID != "" && s.workerMgr != nil {
			if err := s.workerMgr.RemoveSession(ctx, workerID, sid); err != nil {
				results[i] = ItemResult{Index: i, Error: err.Error()}
				return
			}
		}
		results[i] = ItemResult{Index: i, OK: true, Data: map[string]string{"session_id": sid}}
	})
	return summarize(results)
}

// --- helpers ---

// resolveWorkerID looks up the worker responsible for a session by checking bots then accounts.
func (s *Service) resolveWorkerID(ctx context.Context, sessionID string) (string, error) {
	if b, err := s.botSvc.GetBySessionID(ctx, sessionID); err == nil {
		if b.WorkerID != nil {
			return *b.WorkerID, nil
		}
		return "", nil
	}
	if a, err := s.accountSvc.GetBySessionID(ctx, sessionID); err == nil {
		if a.WorkerID != nil {
			return *a.WorkerID, nil
		}
		return "", nil
	}
	return "", fmt.Errorf("session %q not found", sessionID)
}

// parallel runs fn(i) for each index 0..n-1 with bounded concurrency.
func (s *Service) parallel(n int, fn func(i int)) {
	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fn(i)
		}(i)
	}
	wg.Wait()
}

func summarize(items []ItemResult) BulkResult {
	succeeded := 0
	for _, it := range items {
		if it.OK {
			succeeded++
		}
	}
	return BulkResult{
		Total:     len(items),
		Succeeded: succeeded,
		Failed:    len(items) - succeeded,
		Items:     items,
	}
}
