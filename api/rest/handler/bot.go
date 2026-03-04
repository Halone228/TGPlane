package handler

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/bot"
)

type BotHandler struct {
	svc *bot.Service
}

func NewBotHandler(svc *bot.Service) *BotHandler {
	return &BotHandler{svc: svc}
}

func (h *BotHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/bots")
	g.POST("", h.create)
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.DELETE("/:id", h.delete)
}

// POST /api/v1/bots
func (h *BotHandler) create(c *gin.Context) {
	var req bot.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResp(err))
		return
	}
	b, err := h.svc.Add(c.Request.Context(), req)
	if err != nil {
		c.JSON(500, errorResp(err))
		return
	}
	c.JSON(201, b)
}

// GET /api/v1/bots
func (h *BotHandler) list(c *gin.Context) {
	f := bot.ListFilter{
		Limit:  queryInt(c, "limit", 50),
		Offset: queryInt(c, "offset", 0),
	}
	if s := c.Query("status"); s != "" {
		st := bot.Status(s)
		f.Status = &st
	}
	bots, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(500, errorResp(err))
		return
	}
	c.JSON(200, bots)
}

// GET /api/v1/bots/:id
func (h *BotHandler) get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, errorResp(err))
		return
	}
	b, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResp(err))
			return
		}
		c.JSON(500, errorResp(err))
		return
	}
	c.JSON(200, b)
}

// DELETE /api/v1/bots/:id
func (h *BotHandler) delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, errorResp(err))
		return
	}
	if err := h.svc.Remove(c.Request.Context(), id); err != nil {
		c.JSON(500, errorResp(err))
		return
	}
	c.Status(204)
}
