package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/session"
)

type SessionHandler struct {
	pool *session.Pool
}

func NewSessionHandler(pool *session.Pool) *SessionHandler {
	return &SessionHandler{pool: pool}
}

func (h *SessionHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/sessions")
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.DELETE("/:id", h.stop)
}

// GET /api/v1/sessions
func (h *SessionHandler) list(c *gin.Context) {
	c.JSON(http.StatusOK, h.pool.List())
}

// GET /api/v1/sessions/:id
func (h *SessionHandler) get(c *gin.Context) {
	id := c.Param("id")
	s, ok := h.pool.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, errorResp(errNotFound(id)))
		return
	}
	c.JSON(http.StatusOK, s)
}

// DELETE /api/v1/sessions/:id  — stops (but does not delete from DB) the session
func (h *SessionHandler) stop(c *gin.Context) {
	id := c.Param("id")
	if err := h.pool.Remove(id); err != nil {
		c.JSON(http.StatusNotFound, errorResp(err))
		return
	}
	c.Status(http.StatusNoContent)
}

func errNotFound(id string) error {
	return fmt.Errorf("session %q not found", id)
}
