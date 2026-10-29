package openai

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
)

// Source hands out a valid access token, refreshing it from the credential
// file when it is about to expire.
//
// Concurrency model (no lock files): an in-process mutex serializes refreshes
// inside one fonso process. Across processes, every call re-reads the file
// first, so a token another process just refreshed is picked up instead of
// refreshed again; and if our refresh loses a race (the rotating refresh
// token was already used elsewhere), we re-read the file and adopt the winner's
// tokens. A refreshed set is written atomically (temp file + rename).
type Source struct {
	cfg Config
	mu  sync.Mutex
}

// NewSource builds a Source for cfg.
func NewSource(cfg Config) *Source { return &Source{cfg: cfg.withDefaults()} }

// ErrLoginRequired means the stored credentials are unusable and the user must log in again.
var ErrLoginRequired = errors.New("ChatGPT sign-in is missing or expired; run the agent with --login")

// Token returns the access token and ChatGPT account id (empty in official mode).
func (s *Source) Token(ctx context.Context) (access, accountID string, err error) {
	return s.get(ctx, "")
}

// Refresh forces a refresh after the server rejected stale (e.g. HTTP 401),
// unless another caller already replaced that token.
func (s *Source) Refresh(ctx context.Context, stale string) (access, accountID string, err error) {
	if stale == "" {
		return "", "", errors.New("Refresh needs the rejected access token")
	}
	return s.get(ctx, stale)
}

func (s *Source) get(ctx context.Context, stale string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, err := s.load()
	if err != nil {
		return "", "", err
	}
	fresh := s.cfg.Now().Add(refreshSkew).Before(rec.ExpiresAt)
	if stale != "" {
		fresh = rec.AccessToken != stale // someone already replaced the rejected token
	}
	if fresh {
		return rec.AccessToken, rec.AccountID, nil
	}

	next, err := s.refresh(ctx, rec)
	if err != nil {
		// Lost a race with another process? Adopt whatever is on disk now.
		if cur, lerr := s.load(); lerr == nil && cur.RefreshToken != rec.RefreshToken && s.cfg.Now().Before(cur.ExpiresAt) {
			return cur.AccessToken, cur.AccountID, nil
		}
		return "", "", err
	}
	if err := SaveRecord(s.cfg.StorePath, next); err != nil {
		return "", "", fmt.Errorf("saving refreshed credentials: %w", err)
	}
	return next.AccessToken, next.AccountID, nil
}

func (s *Source) load() (Record, error) {
	rec, err := LoadRecord(s.cfg.StorePath)
	if err != nil {
		return Record{}, fmt.Errorf("%w (%v)", ErrLoginRequired, err)
	}
	if rec.Mode != s.cfg.Mode {
		return Record{}, fmt.Errorf("%w (stored login is for mode %q, requested %q)", ErrLoginRequired, rec.Mode, s.cfg.Mode)
	}
	if rec.RefreshToken == "" || rec.AccessToken == "" {
		return Record{}, ErrLoginRequired
	}
	return rec, nil
}

func (s *Source) refresh(ctx context.Context, rec Record) (Record, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {rec.ClientID},
		"refresh_token": {rec.RefreshToken},
	}
	if s.cfg.Mode == ModeOfficial {
		form.Set("resource", OfficialResource)
	}
	tok, err := s.cfg.postToken(ctx, form)
	if err != nil {
		var te *tokenError
		if errors.As(err, &te) && isTerminalRefreshError(te) {
			return Record{}, fmt.Errorf("%w: %v", ErrLoginRequired, err)
		}
		return Record{}, fmt.Errorf("refreshing ChatGPT token: %w", err)
	}
	next := rec
	next.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		next.RefreshToken = tok.RefreshToken
	}
	if tok.IDToken != "" {
		next.IDToken = tok.IDToken
	}
	if tok.TokenType != "" {
		next.TokenType = tok.TokenType
	}
	if sc := tok.Scope; sc != "" {
		next.Scopes = splitScopes(sc)
	}
	next.ExpiresAt = s.cfg.Now().Add(tok.lifetime())
	next.SavedAt = s.cfg.Now().UTC()
	if s.cfg.Mode == ModeOpenCode {
		if id := extractAccountID(tok.IDToken, tok.AccessToken); id != "" {
			next.AccountID = id
		}
	}
	return next, nil
}

func isTerminalRefreshError(e *tokenError) bool {
	switch e.Code {
	case "invalid_grant", "invalid_refresh_token", "token_expired", "refresh_token_expired",
		"refresh_token_invalidated", "refresh_token_reused":
		return true
	}
	return false
}
