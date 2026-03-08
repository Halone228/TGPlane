package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/worker/manager"
)

type WorkerHandler struct {
	mgr *manager.Manager
	ctx context.Context // application lifetime context for AddWorker subscribe loop
}

func NewWorkerHandler(mgr *manager.Manager, appCtx context.Context) *WorkerHandler {
	return &WorkerHandler{mgr: mgr, ctx: appCtx}
}

func (h *WorkerHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/workers")
	g.GET("", h.list)
	g.GET("/metrics", h.collectMetrics)
	g.POST("", h.add)
	g.DELETE("/:id", h.remove)
	g.POST("/:id/drain", h.drain)
}

// GET /api/v1/workers
func (h *WorkerHandler) list(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"workers": h.mgr.Workers()})
}

// GET /api/v1/workers/metrics
func (h *WorkerHandler) collectMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	c.JSON(http.StatusOK, h.mgr.CollectMetrics(ctx))
}

// POST /api/v1/workers
func (h *WorkerHandler) add(c *gin.Context) {
	var body struct {
		ID   string `json:"id"   binding:"required"`
		Addr string `json:"addr" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.mgr.AddWorker(h.ctx, manager.WorkerConfig{
		ID:   body.ID,
		Addr: body.Addr,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": body.ID, "addr": body.Addr})
}

// DELETE /api/v1/workers/:id
func (h *WorkerHandler) remove(c *gin.Context) {
	id := c.Param("id")
	if err := h.mgr.RemoveWorker(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /api/v1/workers/:id/drain
func (h *WorkerHandler) drain(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	result, err := h.mgr.DrainWorker(ctx, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
