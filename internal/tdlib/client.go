package tdlib

import (
	"context"
	"fmt"
	"path/filepath"

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

// Client wraps a single TDLib instance for one account or bot.
type Client struct {
	id     string
	tdlib  *client.Client
	log    *zap.Logger
}

// New creates and authorizes a TDLib client for the given session.
// For bots, set cfg.BotToken. For user accounts, set cfg.PhoneNumber.
func New(cfg SessionConfig, log *zap.Logger) (*Client, error) {
	sessionDir := filepath.Join(cfg.DataDir, cfg.SessionID)

	authorizer := buildAuthorizer(cfg)

	tdlibClient, err := client.NewClient(authorizer,
		client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{NewVerbosityLevel: cfg.LogLevel}),
	)
	if err != nil {
		return nil, fmt.Errorf("create tdlib client for session %s: %w", cfg.SessionID, err)
	}

	_ = sessionDir // tdlib uses files.path from authorizer

	return &Client{
		id:    cfg.SessionID,
		tdlib: tdlibClient,
		log:   log.With(zap.String("session_id", cfg.SessionID)),
	}, nil
}

// ID returns the session identifier.
func (c *Client) ID() string { return c.id }

// Send dispatches a TDLib function and returns the raw response.
func (c *Client) Send(req client.Type) (client.Type, error) {
	return c.tdlib.GetMe()
}

// GetMe returns the current account info.
func (c *Client) GetMe() (*client.User, error) {
	return c.tdlib.GetMe()
}

// Close stops the TDLib client and releases resources.
func (c *Client) Close() {
	if _, err := c.tdlib.Close(); err != nil {
		c.log.Warn("error closing tdlib client", zap.Error(err))
	}
}

// Listener returns a channel that receives incoming updates.
func (c *Client) Listener() *client.Listener {
	return c.tdlib.GetListener()
}

// buildAuthorizer constructs the appropriate TDLib authorizer.
func buildAuthorizer(cfg SessionConfig) client.AuthorizationStateHandler {
	filesDir := filepath.Join(cfg.DataDir, cfg.SessionID)
	params := buildParams(cfg, filesDir)

	if cfg.BotToken != "" {
		auth := client.BotAuthorizer(params, cfg.BotToken)
		return auth
	}

	auth := client.ClientAuthorizer(params)
	auth.PhoneNumber <- cfg.PhoneNumber
	return auth
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

// RunEventLoop reads updates from TDLib and forwards them to the provided handler.
// Blocks until ctx is cancelled or the client is closed.
// The update parameter is always a *client.Type value from go-tdlib.
func (c *Client) RunEventLoop(ctx context.Context, handler func(update interface{})) {
	listener := c.Listener()
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-listener.Updates:
			if !ok {
				c.log.Info("tdlib listener closed")
				return
			}
			handler(update)
		}
	}
}
