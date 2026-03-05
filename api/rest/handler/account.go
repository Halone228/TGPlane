package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/account"
)

// AccountHandler handles HTTP requests for account management.
type AccountHandler struct {
	svc *account.Service
}

func NewAccountHandler(svc *account.Service) *AccountHandler {
	return &AccountHandler{svc: svc}
}

// Register mounts account routes on the given router group.
func (h *AccountHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/accounts")
	g.POST("", h.create)
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.DELETE("/:id", h.delete)
}

// POST /api/v1/accounts
func (h *AccountHandler) create(c *gin.Context) {
	var req account.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	a, err := h.svc.Add(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp(err))
		return
	}
	c.JSON(http.StatusCreated, a)
}

// GET /api/v1/accounts
func (h *AccountHandler) list(c *gin.Context) {
	f := account.ListFilter{
		Limit:  queryInt(c, "limit", 50),
		Offset: queryInt(c, "offset", 0),
	}
	if s := c.Query("status"); s != "" {
		st := account.Status(s)
		f.Status = &st
	}
	accounts, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp(err))
		return
	}
	if accounts == nil {
		accounts = []*account.Account{}
	}
	c.JSON(http.StatusOK, accounts)
}

// GET /api/v1/accounts/:id
func (h *AccountHandler) get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	a, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorResp(err))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp(err))
		return
	}
	c.JSON(http.StatusOK, a)
}

// DELETE /api/v1/accounts/:id
func (h *AccountHandler) delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	if err := h.svc.Remove(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResp(err))
		return
	}
	c.Status(http.StatusNoContent)
}

// --- helpers ---

func parseID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

func queryInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func errorResp(err error) gin.H {
	return gin.H{"error": err.Error()}
}
