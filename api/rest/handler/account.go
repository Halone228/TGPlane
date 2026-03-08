package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/account"
)

// accountAuthHelper is implemented by the worker manager to support auth flow.
type accountAuthHelper interface {
	SendAuthCode(ctx context.Context, workerID, sessionID, code string) error
	SendPassword(ctx context.Context, workerID, sessionID, password string) error
	GetAuthState(ctx context.Context, workerID, sessionID string) (string, error)
}

// AccountHandler handles HTTP requests for account management.
type AccountHandler struct {
	svc    *account.Service
	worker workerAssigner
	auth   accountAuthHelper
}

func NewAccountHandler(svc *account.Service, worker workerAssigner) *AccountHandler {
	var auth accountAuthHelper
	if a, ok := worker.(accountAuthHelper); ok {
		auth = a
	}
	return &AccountHandler{svc: svc, worker: worker, auth: auth}
}

// Register mounts account routes on the given router group.
func (h *AccountHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/accounts")
	g.POST("", h.create)
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.DELETE("/:id", h.delete)
	g.POST("/:id/auth-code", h.sendAuthCode)
	g.POST("/:id/password", h.sendPassword)
	g.GET("/:id/auth-state", h.getAuthState)
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
	if h.worker != nil {
		workerID, err := h.worker.AssignAccount(c.Request.Context(), a.SessionID, a.Phone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResp(err))
			return
		}
		a.WorkerID = &workerID
		a.Status = account.StatusAuthorizing
		_ = h.svc.UpdateStatus(c.Request.Context(), a.ID, account.StatusAuthorizing)
		_ = h.svc.UpdateWorkerID(c.Request.Context(), a.SessionID, workerID)
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

// POST /api/v1/accounts/:id/auth-code
func (h *AccountHandler) sendAuthCode(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, errorResp(fmt.Errorf("auth not available")))
		return
	}
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	var body struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	a, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResp(err))
		return
	}
	if a.WorkerID == nil {
		c.JSON(http.StatusBadRequest, errorResp(fmt.Errorf("account not assigned to a worker")))
		return
	}
	if err := h.auth.SendAuthCode(c.Request.Context(), *a.WorkerID, a.SessionID, body.Code); err != nil {
		c.JSON(http.StatusInternalServerError, errorResp(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "code_submitted"})
}

// POST /api/v1/accounts/:id/password
func (h *AccountHandler) sendPassword(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, errorResp(fmt.Errorf("auth not available")))
		return
	}
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	var body struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	a, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResp(err))
		return
	}
	if a.WorkerID == nil {
		c.JSON(http.StatusBadRequest, errorResp(fmt.Errorf("account not assigned to a worker")))
		return
	}
	if err := h.auth.SendPassword(c.Request.Context(), *a.WorkerID, a.SessionID, body.Password); err != nil {
		c.JSON(http.StatusInternalServerError, errorResp(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "password_submitted"})
}

// GET /api/v1/accounts/:id/auth-state
func (h *AccountHandler) getAuthState(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, errorResp(fmt.Errorf("auth not available")))
		return
	}
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp(err))
		return
	}
	a, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResp(err))
		return
	}
	if a.WorkerID == nil {
		c.JSON(http.StatusOK, gin.H{"session_id": a.SessionID, "state": "not_assigned"})
		return
	}
	state, err := h.auth.GetAuthState(c.Request.Context(), *a.WorkerID, a.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"session_id": a.SessionID, "state": state})
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
