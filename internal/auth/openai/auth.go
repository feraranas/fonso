// Package openai implements "Sign in with ChatGPT" OAuth for fonso: a PKCE
// loopback login, an on-disk credential record, and safe token refresh.
//
// Two modes exist:
//
//   - ModeOfficial (default) follows OpenAI's documented flow for open-source
//     clients (https://developers.openai.com/siwc/token-sharing-open-source):
//     dynamic client registration, resource=https://api.openai.com/v1, and
//     inference against the public Responses API.
//   - ModeOpenCode reuses the Codex/opencode client id and the chatgpt.com
//     backend. It is NOT a registered client for fonso; it exists only because
//     it is what opencode does, and it must be chosen explicitly.
//
// Tokens are never logged or printed anywhere in this package.
package openai

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Mode selects which OAuth client/endpoints are used.
type Mode string

const (
	ModeOfficial Mode = "official"
	ModeOpenCode Mode = "opencode"
)

const (
	DefaultIssuer = "https://auth.openai.com"

	// DynamicClientID is the first-time registration entrypoint in OpenAI's docs.
	DynamicClientID = "dynamic_agent_client"
	// OfficialResource is the audience for ChatGPT-plan access tokens.
	OfficialResource = "https://api.openai.com/v1"
	// RequiredScope must be granted for ChatGPT plan usage in official mode.
	RequiredScope = "chatgpt.tokens.use.direct"
	// AgentName is sent as agent_name_hint on first registration.
	AgentName = "fonso"

	// OpenCodeClientID is opencode's (Codex's) public client id.
	OpenCodeClientID = "app_EMoamEEZ73f0CkXaXp7hrann"

	OfficialInferenceURL = "https://api.openai.com/v1/responses"
	OpenCodeInferenceURL = "https://chatgpt.com/backend-api/codex/responses"

	DefaultCallbackAddr = "127.0.0.1:1455"
	callbackPath        = "/auth/callback"

	// refreshSkew refreshes this long before the access token expires.
	refreshSkew = 2 * time.Minute
)

// Config holds everything injectable so tests can point at stub servers.
type Config struct {
	Mode         Mode
	Issuer       string // default DefaultIssuer
	InferenceURL string // default depends on Mode
	CallbackAddr string // default DefaultCallbackAddr; tests use 127.0.0.1:0
	StorePath    string // default ~/.config/fonso/openai-auth.json
	HTTPClient   *http.Client
	// OpenBrowser opens the authorization URL. Default tries the OS opener.
	OpenBrowser  func(url string) error
	Out          io.Writer // where login instructions are printed; default os.Stdout
	Now          func() time.Time
	LoginTimeout time.Duration // default 5 minutes
}

func (c Config) withDefaults() Config {
	if c.Mode == "" {
		c.Mode = ModeOfficial
	}
	if c.Issuer == "" {
		c.Issuer = DefaultIssuer
	}
	c.Issuer = strings.TrimRight(c.Issuer, "/")
	if c.InferenceURL == "" {
		if c.Mode == ModeOpenCode {
			c.InferenceURL = OpenCodeInferenceURL
		} else {
			c.InferenceURL = OfficialInferenceURL
		}
	}
	if c.CallbackAddr == "" {
		c.CallbackAddr = DefaultCallbackAddr
	}
	if c.StorePath == "" {
		c.StorePath = DefaultStorePath()
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if c.OpenBrowser == nil {
		c.OpenBrowser = openSystemBrowser
	}
	if c.Out == nil {
		c.Out = os.Stdout
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.LoginTimeout == 0 {
		c.LoginTimeout = 5 * time.Minute
	}
	return c
}

func (c Config) authorizeURL() string {
	if c.Mode == ModeOpenCode {
		return c.Issuer + "/oauth/authorize"
	}
	return c.Issuer + "/api/accounts/authorize"
}

func (c Config) tokenURL() string {
	if c.Mode == ModeOpenCode {
		return c.Issuer + "/oauth/token"
	}
	return c.Issuer + "/api/accounts/oauth/token"
}

// InferenceEndpoint returns the Responses endpoint for this configuration.
func (c Config) InferenceEndpoint() string { return c.withDefaults().InferenceURL }

// DefaultStorePath is ~/.config/fonso/openai-auth.json (XDG_CONFIG_HOME wins).
func DefaultStorePath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "fonso", "openai-auth.json")
}

// Record is the credential file. Its String/GoString are redacted so a stray
// %v or %+v can never leak tokens.
type Record struct {
	Mode           Mode      `json:"mode"`
	Issuer         string    `json:"issuer"`
	Email          string    `json:"email,omitempty"`
	Subject        string    `json:"subject,omitempty"`
	ClientID       string    `json:"client_id"`
	ExtAgentHostID string    `json:"ext_agent_host_id,omitempty"`
	IDToken        string    `json:"id_token,omitempty"`
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	TokenType      string    `json:"token_type,omitempty"`
	ExpiresAt      time.Time `json:"expires_at"`
	Scopes         []string  `json:"scopes,omitempty"`
	AccountID      string    `json:"account_id,omitempty"`
	SavedAt        time.Time `json:"saved_at"`
}

func (r Record) String() string {
	return fmt.Sprintf("openai-auth{mode=%s client_id=%s email=%s expires_at=%s tokens=<redacted>}", r.Mode, r.ClientID, r.Email, r.ExpiresAt.Format(time.RFC3339))
}
func (r Record) GoString() string { return r.String() }

// LoadRecord reads the credential file.
func LoadRecord(path string) (Record, error) {
	var rec Record
	b, err := os.ReadFile(path)
	if err != nil {
		return rec, err
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		return rec, fmt.Errorf("parse %s: %w", path, err)
	}
	return rec, nil
}

// SaveRecord writes the credential file atomically with mode 0600: a
// 0600 temp file in the same directory is fsynced, then renamed over the target.
func SaveRecord(path string, rec Record) error {
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, b)
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".openai-auth-*.tmp") // created 0600
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// loadOrCreateHostID returns the stable ext_agent_host_id for this machine,
// generating a urn:uuid: value once and persisting it next to the auth file.
func loadOrCreateHostID(storePath string) (string, error) {
	path := filepath.Join(filepath.Dir(storePath), "openai-host-id")
	if b, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(b)); strings.HasPrefix(id, "urn:uuid:") {
			return id, nil
		}
	}
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // RFC 4122 variant
	id := fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
	if err := writeFileAtomic(path, []byte(id+"\n")); err != nil {
		return "", err
	}
	return id, nil
}
