package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/bulk"
)

type BulkHandler struct{ svc *bulk.Service }

func NewBulkHandler(svc *bulk.Service) *BulkHandler { return &BulkHandler{svc: svc} }

func (h *BulkHandler) Register(r gin.IRouter) {
	g := r.Group("/bulk")
	g.POST("/bots", h.addBots)
	g.POST("/accounts", h.addAccounts)
	g.DELETE("/sessions", h.removeSessions)
}

// POST /api/v1/bulk/bots
func (h *BulkHandler) addBots(c *gin.Context) {
	var body struct {
		Items []bulk.AddBotRequest `json:"items" binding:"required,min=1,max=500"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := h.svc.AddBots(c.Request.Context(), body.Items)
	c.JSON(http.StatusMultiStatus, result)
}

// POST /api/v1/bulk/accounts
func (h *BulkHandler) addAccounts(c *gin.Context) {
	var body struct {
		Items []bulk.AddAccountRequest `json:"items" binding:"required,min=1,max=500"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := h.svc.AddAccounts(c.Request.Context(), body.Items)
	c.JSON(http.StatusMultiStatus, result)
}

// DELETE /api/v1/bulk/sessions
func (h *BulkHandler) removeSessions(c *gin.Context) {
	var body struct {
		SessionIDs []string `json:"session_ids" binding:"required,min=1,max=1000"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := h.svc.RemoveSessions(c.Request.Context(), body.SessionIDs)
	c.JSON(http.StatusMultiStatus, result)
}
