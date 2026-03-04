package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/webhook"
)

type WebhookHandler struct{ svc *webhook.Service }

func NewWebhookHandler(svc *webhook.Service) *WebhookHandler { return &WebhookHandler{svc: svc} }

func (h *WebhookHandler) Register(r gin.IRouter) {
	g := r.Group("/webhooks")
	g.POST("", h.create)
	g.GET("", h.list)
	g.DELETE("/:id", h.delete)
}

// POST /api/v1/webhooks
func (h *WebhookHandler) create(c *gin.Context) {
	var body struct {
		URL    string   `json:"url"    binding:"required,url"`
		Secret string   `json:"secret"`
		Events []string `json:"events"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Events == nil {
		body.Events = []string{}
	}
	wh, err := h.svc.Create(c.Request.Context(), body.URL, body.Secret, body.Events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, wh)
}

// GET /api/v1/webhooks
func (h *WebhookHandler) list(c *gin.Context) {
	hooks, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hooks)
}

// DELETE /api/v1/webhooks/:id
func (h *WebhookHandler) delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if err == webhook.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
