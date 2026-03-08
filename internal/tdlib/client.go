package tdlib

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/zelenin/go-tdlib/client"
	"go.uber.org/zap"
)

// SessionConfig holds per-session TDLib configuration.
type SessionConfig struct {
	SessionID string
	// Phone number for user accounts. Empty for bots.
	PhoneNumber string
	// Bot token. Empty for user accounts.
	BotToken string
	// APIID and APIHash from https://my.telegram.org
	APIID   int32
	APIHash string
	// DataDir is the root directory; per-session data lives at DataDir/SessionID/
	DataDir  string
	LogLevel int32
	UseTestDC bool
}

// AuthState represents the current TDLib authorization state.
type AuthState string

const (
	AuthStateWaitPhone    AuthState = "waiting_phone"
	AuthStateWaitCode     AuthState = "waiting_code"
	AuthStateWaitPassword AuthState = "waiting_password"
	AuthStateReady        AuthState = "ready"
	AuthStateError        AuthState = "error"
	AuthStateClosed       AuthState = "closed"
)

// Client wraps a single TDLib instance for one account or bot.
type Client struct {
	id    string
	tdlib *client.Client
	log   *zap.Logger

	// For user accounts: channels for submitting code/password during auth.
	mu        sync.RWMutex
	authState AuthState
	authErr   error
	codeCh    chan string // nil for bots
	passwordCh chan string // nil for bots
}

// New creates and authorizes a TDLib client. For bots it blocks until auth is complete.
// For user accounts it starts auth asynchronously - caller must monitor AuthState()
// and call SendCode()/SendPassword() as needed.
func New(cfg SessionConfig, log *zap.Logger) (*Client, error) {
	l := log.With(zap.String("session_id", cfg.SessionID))

	if cfg.BotToken != "" {
		return newBot(cfg, l)
	}
	return newAccount(cfg, l)
}

func newBot(cfg SessionConfig, log *zap.Logger) (*Client, error) {
	filesDir := filepath.Join(cfg.DataDir, cfg.SessionID)
	params := buildParams(cfg, filesDir)
	auth := client.BotAuthorizer(params, cfg.BotToken)

	tdlibClient, err := client.NewClient(auth,
		client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{NewVerbosityLevel: cfg.LogLevel}),
	)
	if err != nil {
		return nil, fmt.Errorf("create bot client for session %s: %w", cfg.SessionID, err)
	}
	return &Client{
		id:        cfg.SessionID,
		tdlib:     tdlibClient,
		log:       log,
		authState: AuthStateReady,
	}, nil
}

func newAccount(cfg SessionConfig, log *zap.Logger) (*Client, error) {
	filesDir := filepath.Join(cfg.DataDir, cfg.SessionID)
	params := buildParams(cfg, filesDir)
	auth := client.ClientAuthorizer(params)

	c := &Client{
		id:         cfg.SessionID,
		log:        log,
		authState:  AuthStateWaitPhone,
		codeCh:     auth.Code,
		passwordCh: auth.Password,
	}

	// Send phone number to the authorizer channel.
	auth.PhoneNumber <- cfg.PhoneNumber

	// Run NewClient (which calls Authorize) in a goroutine.
	// NewClient blocks until auth is complete or fails.
	go func() {
		tdlibClient, err := client.NewClient(auth,
			client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{NewVerbosityLevel: cfg.LogLevel}),
		)
		if err != nil {
			c.mu.Lock()
			c.authState = AuthStateError
			c.authErr = err
			c.mu.Unlock()
			log.Error("tdlib authorization failed", zap.Error(err))
			return
		}
		c.mu.Lock()
		c.tdlib = tdlibClient
		c.authState = AuthStateReady
		c.mu.Unlock()
		log.Info("tdlib authorization complete")
	}()

	// Monitor the auth State channel to track what TDLib is waiting for.
	go func() {
		for state := range auth.State {
			switch state.AuthorizationStateType() {
			case client.TypeAuthorizationStateWaitCode:
				c.mu.Lock()
				c.authState = AuthStateWaitCode
				c.mu.Unlock()
				log.Info("TDLib waiting for auth code")
			case client.TypeAuthorizationStateWaitPassword:
				c.mu.Lock()
				c.authState = AuthStateWaitPassword
				c.mu.Unlock()
				log.Info("TDLib waiting for 2FA password")
			case client.TypeAuthorizationStateReady:
				c.mu.Lock()
				c.authState = AuthStateReady
				c.mu.Unlock()
			}
		}
	}()

	return c, nil
}

// ID returns the session identifier.
func (c *Client) ID() string { return c.id }

// AuthState returns the current authorization state.
func (c *Client) AuthState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return string(c.authState)
}

// AuthError returns the authorization error, if any.
func (c *Client) AuthError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authErr
}

// SendCode submits the authentication code. Only valid when auth state is "waiting_code".
func (c *Client) SendCode(code string) error {
	c.mu.RLock()
	state := c.authState
	ch := c.codeCh
	c.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("not a user account session")
	}
	if state != AuthStateWaitCode {
		return fmt.Errorf("session not waiting for code (state: %s)", state)
	}

	select {
	case ch <- code:
		return nil
	default:
		return fmt.Errorf("code channel is full or closed")
	}
}

// SendPassword submits the 2FA password. Only valid when auth state is "waiting_password".
func (c *Client) SendPassword(password string) error {
	c.mu.RLock()
	state := c.authState
	ch := c.passwordCh
	c.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("not a user account session")
	}
	if state != AuthStateWaitPassword {
		return fmt.Errorf("session not waiting for password (state: %s)", state)
	}

	select {
	case ch <- password:
		return nil
	default:
		return fmt.Errorf("password channel is full or closed")
	}
}

// Send dispatches a TDLib function and returns the raw response.
func (c *Client) Send(req client.Type) (client.Type, error) {
	c.mu.RLock()
	cl := c.tdlib
	c.mu.RUnlock()
	if cl == nil {
		return nil, fmt.Errorf("client not yet authorized")
	}
	return cl.GetMe()
}

// GetMe returns the current account info.
func (c *Client) GetMe() (*client.User, error) {
	c.mu.RLock()
	cl := c.tdlib
	c.mu.RUnlock()
	if cl == nil {
		return nil, fmt.Errorf("client not yet authorized")
	}
	return cl.GetMe()
}

// Close stops the TDLib client and releases resources.
func (c *Client) Close() {
	c.mu.RLock()
	cl := c.tdlib
	c.mu.RUnlock()
	if cl != nil {
		if _, err := cl.Close(); err != nil {
			c.log.Warn("error closing tdlib client", zap.Error(err))
		}
	}
}

// Listener returns a channel that receives incoming updates.
func (c *Client) Listener() *client.Listener {
	c.mu.RLock()
	cl := c.tdlib
	c.mu.RUnlock()
	if cl == nil {
		return nil
	}
	return cl.GetListener()
}

// RunEventLoop reads updates from TDLib and forwards them to the provided handler.
// For user accounts that are still authorizing, it waits until the client is ready.
// Blocks until ctx is cancelled or the client is closed.
func (c *Client) RunEventLoop(ctx context.Context, handler func(update interface{})) {
	// Wait for client to be ready (for async account auth).
	for {
		c.mu.RLock()
		cl := c.tdlib
		state := c.authState
		c.mu.RUnlock()

		if cl != nil {
			break
		}
		if state == AuthStateError || state == AuthStateClosed {
			c.log.Error("event loop: auth failed, not starting")
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			// poll until client is ready
		}
	}

	listener := c.Listener()
	if listener == nil {
		c.log.Error("event loop: could not get listener")
		return
	}
	defer listener.Close()

	c.log.Info("event loop started, listener registered")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("event loop: context cancelled")
			return
		case update, ok := <-listener.Updates:
			if !ok {
				c.log.Info("tdlib listener closed")
				return
			}
			c.log.Debug("event loop: received update",
				zap.String("type", fmt.Sprintf("%T", update)),
			)
			handler(update)
		}
	}
}

func buildParams(cfg SessionConfig, filesDir string) *client.SetTdlibParametersRequest {
	return &client.SetTdlibParametersRequest{
		UseTestDc:           cfg.UseTestDC,
		DatabaseDirectory:   filesDir,
		FilesDirectory:      filesDir,
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseMessageDatabase:  true,
		UseSecretChats:      false,
		ApiId:               cfg.APIID,
		ApiHash:             cfg.APIHash,
		SystemLanguageCode:  "en",
		DeviceModel:         "TGPlane",
		ApplicationVersion:  "0.1.0",
	}
}
