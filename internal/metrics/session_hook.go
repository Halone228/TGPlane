package metrics

import "github.com/tgplane/tgplane/internal/session"

// SessionHook implements session.Hook and records pool events into Prometheus.
type SessionHook struct{}

func NewSessionHook() *SessionHook { return &SessionHook{} }

func (h *SessionHook) OnAdded(sessType session.Type) {
	t := string(sessType)
	SessionsTotal.WithLabelValues(t).Inc()
	SessionsActive.WithLabelValues(t, string(session.StatusAuthorizing)).Inc()
}

func (h *SessionHook) OnRemoved(sessType session.Type, finalStatus session.Status) {
	t := string(sessType)
	SessionsActive.WithLabelValues(t, string(finalStatus)).Dec()
}

func (h *SessionHook) OnStatusChanged(sessType session.Type, old, new session.Status) {
	t := string(sessType)
	SessionsActive.WithLabelValues(t, string(old)).Dec()
	SessionsActive.WithLabelValues(t, string(new)).Inc()
}

func (h *SessionHook) OnError(sessType session.Type) {
	SessionErrors.WithLabelValues(string(sessType)).Inc()
}
