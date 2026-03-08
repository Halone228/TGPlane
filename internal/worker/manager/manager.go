// Package manager runs on the main node and manages connections to all worker nodes.
package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	wc "github.com/tgplane/tgplane/internal/worker/client"
	"go.uber.org/zap"
)

// WorkerStore is an optional persistence layer for worker registrations.
type WorkerStore interface {
	Upsert(ctx context.Context, id, addr string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]WorkerEntry, error)
}

// WorkerEntry represents a persisted worker registration.
type WorkerEntry struct {
	ID   string
	Addr string
}

// UpdateHandler is called for every update routed from any worker.
type UpdateHandler func(workerID string, update *pb.TelegramUpdate)

// WorkerConfig holds static configuration for a single worker node.
type WorkerConfig struct {
	ID   string
	Addr string
}

// Manager maintains gRPC clients to all configured workers,
// keeps subscription streams alive, and routes updates downstream.
type Manager struct {
	mu      sync.RWMutex
	workers map[string]*wc.Client

	onUpdate      UpdateHandler
	onWorkerReady func(ctx context.Context, workerID string)
	log           *zap.Logger
	store         WorkerStore
}

func New(onUpdate UpdateHandler, log *zap.Logger, store WorkerStore) *Manager {
	return &Manager{
		workers:  make(map[string]*wc.Client),
		onUpdate: onUpdate,
		log:      log,
		store:    store,
	}
}

// AddWorker dials a new worker and starts streaming updates from it.
// ctx controls the lifetime of the subscribe loop.
func (m *Manager) AddWorker(ctx context.Context, cfg WorkerConfig) error {
	m.mu.Lock()
	if _, exists := m.workers[cfg.ID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("worker %q already registered", cfg.ID)
	}
	m.mu.Unlock()

	c, err := wc.New(cfg.ID, cfg.Addr, m.log)
	if err != nil {
		return err
	}

	// Health check before registering.
	hCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = c.Health(hCtx)
	cancel()
	if err != nil {
		c.Close()
		return fmt.Errorf("worker %s health check failed: %w", cfg.ID, err)
	}

	m.mu.Lock()
	m.workers[cfg.ID] = c
	m.mu.Unlock()

	if m.store != nil {
		if err := m.store.Upsert(ctx, cfg.ID, cfg.Addr); err != nil {
			m.log.Warn("persist worker registration", zap.Error(err))
		}
	}

	m.log.Info("worker registered", zap.String("id", cfg.ID), zap.String("addr", cfg.Addr))

	// Start the subscribe loop in the background with reconnect.
	go m.subscribeLoop(ctx, c, m.onUpdate)

	if m.onWorkerReady != nil {
		go m.onWorkerReady(ctx, cfg.ID)
	}

	return nil
}

// RemoveWorker closes the connection to a worker immediately.
func (m *Manager) RemoveWorker(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.workers[id]
	if !ok {
		return fmt.Errorf("worker %q not found", id)
	}
	c.Close()
	delete(m.workers, id)

	if m.store != nil {
		if err := m.store.Delete(context.Background(), id); err != nil {
			m.log.Warn("remove worker from store", zap.Error(err))
		}
	}

	m.log.Info("worker removed", zap.String("id", id))
	return nil
}

// RestoreWorkers re-registers workers previously saved in the store.
func (m *Manager) RestoreWorkers(ctx context.Context) int {
	if m.store == nil {
		return 0
	}
	entries, err := m.store.List(ctx)
	if err != nil {
		m.log.Error("load saved workers", zap.Error(err))
		return 0
	}
	restored := 0
	for _, e := range entries {
		if err := m.AddWorker(ctx, WorkerConfig{ID: e.ID, Addr: e.Addr}); err != nil {
			m.log.Warn("restore worker failed",
				zap.String("id", e.ID),
				zap.String("addr", e.Addr),
				zap.Error(err),
			)
			continue
		}
		restored++
	}
	m.log.Info("workers restored from database", zap.Int("count", restored))
	return restored
}

// DrainResult holds the outcome of a drain operation.
type DrainResult struct {
	Migrated int            `json:"migrated"`
	Failed   int            `json:"failed"`
	Errors   []SessionError `json:"errors,omitempty"`
}

// SessionError describes a per-session migration failure.
type SessionError struct {
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
}

// DrainWorker reassigns all sessions from the target worker to other workers,
// then removes it. The returned DrainResult contains per-session error details.
// The error return is reserved for fatal errors (worker not found, can't list sessions).
func (m *Manager) DrainWorker(ctx context.Context, id string) (*DrainResult, error) {
	m.mu.RLock()
	target, ok := m.workers[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("worker %q not found", id)
	}

	sessions, err := target.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sessions on %s: %w", id, err)
	}

	log := m.log.With(zap.String("drained_worker", id), zap.Int("sessions", len(sessions)))
	log.Info("draining worker")

	result := &DrainResult{}

	for _, sess := range sessions {
		// Pick a different worker for each session.
		dest, pickErr := m.leastLoadedExcluding(ctx, id)
		if pickErr != nil {
			log.Warn("no target worker for session, skipping",
				zap.String("session_id", sess.SessionId), zap.Error(pickErr))
			result.Failed++
			result.Errors = append(result.Errors, SessionError{
				SessionID: sess.SessionId,
				Error:     pickErr.Error(),
			})
			continue
		}

		var assignErr error
		switch sess.Type {
		case "bot":
			_, assignErr = dest.AddBot(ctx, sess.SessionId, "")
		default:
			_, assignErr = dest.AddAccount(ctx, sess.SessionId, "")
		}
		if assignErr != nil {
			log.Warn("reassign failed", zap.String("session_id", sess.SessionId), zap.Error(assignErr))
			result.Failed++
			result.Errors = append(result.Errors, SessionError{
				SessionID: sess.SessionId,
				Error:     assignErr.Error(),
			})
			continue
		}

		// Remove from source.
		if rmErr := target.RemoveSession(ctx, sess.SessionId); rmErr != nil {
			log.Warn("remove from source failed", zap.String("session_id", sess.SessionId), zap.Error(rmErr))
		}
		result.Migrated++
	}

	log.Info("drain complete", zap.Int("migrated", result.Migrated), zap.Int("failed", result.Failed))
	return result, m.RemoveWorker(id)
}

// leastLoadedExcluding picks the least-loaded worker, skipping excludeID.
func (m *Manager) leastLoadedExcluding(ctx context.Context, excludeID string) (*wc.Client, error) {
	m.mu.RLock()
	clients := make([]*wc.Client, 0, len(m.workers))
	for id, c := range m.workers {
		if id != excludeID {
			clients = append(clients, c)
		}
	}
	m.mu.RUnlock()

	if len(clients) == 0 {
		return nil, fmt.Errorf("no other workers available")
	}

	type scored struct {
		c     *wc.Client
		count int
	}
	scores := make([]scored, 0, len(clients))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, c := range clients {
		wg.Add(1)
		go func(c *wc.Client) {
			defer wg.Done()
			sessions, err := c.ListSessions(ctx)
			count := 0
			if err == nil {
				count = len(sessions)
			}
			mu.Lock()
			scores = append(scores, scored{c, count})
			mu.Unlock()
		}(c)
	}
	wg.Wait()

	best := scores[0]
	for _, s := range scores[1:] {
		if s.count < best.count {
			best = s
		}
	}
	return best.c, nil
}

// AssignAccount asks the least-loaded worker to start an account session.
func (m *Manager) AssignAccount(ctx context.Context, sessionID, phone string) (workerID string, err error) {
	c, err := m.leastLoaded(ctx)
	if err != nil {
		return "", err
	}
	if _, err := c.AddAccount(ctx, sessionID, phone); err != nil {
		return "", fmt.Errorf("add account on worker %s: %w", c.ID(), err)
	}
	return c.ID(), nil
}

// AssignBot asks the least-loaded worker to start a bot session.
func (m *Manager) AssignBot(ctx context.Context, sessionID, token string) (workerID string, err error) {
	c, err := m.leastLoaded(ctx)
	if err != nil {
		return "", err
	}
	if _, err := c.AddBot(ctx, sessionID, token); err != nil {
		return "", fmt.Errorf("add bot on worker %s: %w", c.ID(), err)
	}
	return c.ID(), nil
}

// SetOnWorkerReady registers a callback invoked (in a goroutine) each time a
// worker is successfully added via AddWorker.
func (m *Manager) SetOnWorkerReady(fn func(ctx context.Context, workerID string)) {
	m.onWorkerReady = fn
}

// AssignBotToWorker assigns a bot session to a specific worker (by ID).
func (m *Manager) AssignBotToWorker(ctx context.Context, workerID, sessionID, token string) (string, error) {
	m.mu.RLock()
	c, ok := m.workers[workerID]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("worker %q not found", workerID)
	}
	if _, err := c.AddBot(ctx, sessionID, token); err != nil {
		return "", fmt.Errorf("add bot on worker %s: %w", workerID, err)
	}
	return workerID, nil
}

// AssignAccountToWorker assigns an account session to a specific worker (by ID).
func (m *Manager) AssignAccountToWorker(ctx context.Context, workerID, sessionID, phone string) (string, error) {
	m.mu.RLock()
	c, ok := m.workers[workerID]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("worker %q not found", workerID)
	}
	if _, err := c.AddAccount(ctx, sessionID, phone); err != nil {
		return "", fmt.Errorf("add account on worker %s: %w", workerID, err)
	}
	return workerID, nil
}

// RemoveSession removes a session from the specified worker.
func (m *Manager) RemoveSession(ctx context.Context, workerID, sessionID string) error {
	m.mu.RLock()
	c, ok := m.workers[workerID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("worker %q not found", workerID)
	}
	return c.RemoveSession(ctx, sessionID)
}

// SendAuthCode routes an auth code to the worker running the given session.
func (m *Manager) SendAuthCode(ctx context.Context, workerID, sessionID, code string) error {
	m.mu.RLock()
	c, ok := m.workers[workerID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("worker %q not found", workerID)
	}
	_, err := c.SendAuthCode(ctx, sessionID, code)
	return err
}

// SendPassword routes a 2FA password to the worker running the given session.
func (m *Manager) SendPassword(ctx context.Context, workerID, sessionID, password string) error {
	m.mu.RLock()
	c, ok := m.workers[workerID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("worker %q not found", workerID)
	}
	_, err := c.SendPassword(ctx, sessionID, password)
	return err
}

// GetAuthState gets the auth state from the worker running the given session.
func (m *Manager) GetAuthState(ctx context.Context, workerID, sessionID string) (string, error) {
	m.mu.RLock()
	c, ok := m.workers[workerID]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("worker %q not found", workerID)
	}
	state, err := c.GetAuthState(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return state.State, nil
}

// Workers returns a snapshot list of currently registered worker IDs.
func (m *Manager) Workers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.workers))
	for id := range m.workers {
		ids = append(ids, id)
	}
	return ids
}

// CollectMetrics fetches metrics from all workers concurrently.
func (m *Manager) CollectMetrics(ctx context.Context) []*pb.WorkerMetrics {
	m.mu.RLock()
	clients := make([]*wc.Client, 0, len(m.workers))
	for _, c := range m.workers {
		clients = append(clients, c)
	}
	m.mu.RUnlock()

	type result struct {
		m   *pb.WorkerMetrics
		err error
	}
	ch := make(chan result, len(clients))

	for _, c := range clients {
		go func(c *wc.Client) {
			m, err := c.GetMetrics(ctx)
			ch <- result{m, err}
		}(c)
	}

	metrics := make([]*pb.WorkerMetrics, 0, len(clients))
	for range clients {
		r := <-ch
		if r.err == nil {
			metrics = append(metrics, r.m)
		}
	}
	return metrics
}

// subscribeLoop subscribes to updates from a worker and reconnects on error.
func (m *Manager) subscribeLoop(ctx context.Context, c *wc.Client, handler UpdateHandler) {
	log := m.log.With(zap.String("worker_id", c.ID()))
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}
		err := c.Subscribe(ctx, nil, wc.UpdateHandler(handler))
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Warn("subscribe stream ended, reconnecting",
				zap.Error(err),
				zap.Duration("backoff", backoff),
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}
}

// leastLoaded picks the worker with the fewest sessions.
func (m *Manager) leastLoaded(ctx context.Context) (*wc.Client, error) {
	m.mu.RLock()
	clients := make([]*wc.Client, 0, len(m.workers))
	for _, c := range m.workers {
		clients = append(clients, c)
	}
	m.mu.RUnlock()

	if len(clients) == 0 {
		return nil, fmt.Errorf("no workers available")
	}

	type scored struct {
		c     *wc.Client
		count int
	}
	scores := make([]scored, 0, len(clients))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, c := range clients {
		wg.Add(1)
		go func(c *wc.Client) {
			defer wg.Done()
			sessions, err := c.ListSessions(ctx)
			count := 0
			if err == nil {
				count = len(sessions)
			}
			mu.Lock()
			scores = append(scores, scored{c, count})
			mu.Unlock()
		}(c)
	}
	wg.Wait()

	best := scores[0]
	for _, s := range scores[1:] {
		if s.count < best.count {
			best = s
		}
	}
	return best.c, nil
}
