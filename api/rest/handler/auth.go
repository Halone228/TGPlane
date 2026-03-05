package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/auth"
)

type AuthHandler struct{ svc *auth.Service }

func NewAuthHandler(svc *auth.Service) *AuthHandler { return &AuthHandler{svc: svc} }

func (h *AuthHandler) Register(r gin.IRouter) {
	g := r.Group("/auth/keys")
	g.POST("", h.create)
	g.GET("", h.list)
	g.DELETE("/:id", h.delete)
}

// POST /api/v1/auth/keys
func (h *AuthHandler) create(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	k, rawKey, err := h.svc.Create(c.Request.Context(), body.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         k.ID,
		"name":       k.Name,
		"key_prefix": k.KeyPrefix,
		"key":        rawKey, // shown once
		"created_at": k.CreatedAt,
	})
}

// GET /api/v1/auth/keys
func (h *AuthHandler) list(c *gin.Context) {
	keys, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if keys == nil {
		keys = []*auth.APIKey{}
	}
	c.JSON(http.StatusOK, keys)
}

// DELETE /api/v1/auth/keys/:id
func (h *AuthHandler) delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if err == auth.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
