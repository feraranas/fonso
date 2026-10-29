package openai

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func seedRecord(t *testing.T, cfg Config, mode Mode, expiresIn time.Duration) Record {
	t.Helper()
	rec := Record{
		Mode: mode, Issuer: cfg.Issuer, ClientID: "oaiapp_issued_1",
		AccessToken: "old-access", RefreshToken: "old-refresh", AccountID: "acct-42",
		ExpiresAt: time.Now().Add(expiresIn),
	}
	if err := SaveRecord(cfg.StorePath, rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestExpiredTokenIsRefreshedBeforeUse(t *testing.T) {
	s := newStub(t)
	cfg := testConfig(t, s, ModeOfficial, &strings.Builder{}, nil)
	seedRecord(t, cfg, ModeOfficial, -time.Minute)
	src := NewSource(cfg)

	access, _, err := src.Token(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if access != "access-1" || s.refreshCount != 1 {
		t.Fatalf("access=%q refreshes=%d", access, s.refreshCount)
	}
	f := s.tokenForms[0]
	if f.Get("grant_type") != "refresh_token" || f.Get("refresh_token") != "old-refresh" || f.Get("client_id") != "oaiapp_issued_1" || f.Get("resource") != OfficialResource || f.Get("scope") != "" {
		t.Errorf("refresh form = %v", f)
	}
	// Rotated tokens persisted together, still 0600, other fields preserved.
	rec, _ := LoadRecord(cfg.StorePath)
	if rec.AccessToken != "access-1" || rec.RefreshToken != "refresh-1" || rec.ClientID != "oaiapp_issued_1" || rec.AccountID != "acct-42" {
		t.Errorf("record after refresh = %v", rec)
	}
	assertMode(t, cfg.StorePath, 0o600)

	// A valid token is reused without another network call.
	access, _, err = src.Token(t.Context())
	if err != nil || access != "access-1" || s.refreshCount != 1 {
		t.Fatalf("second call: %q %v refreshes=%d", access, err, s.refreshCount)
	}
}

func TestTokenInsideSkewWindowRefreshes(t *testing.T) {
	s := newStub(t)
	cfg := testConfig(t, s, ModeOpenCode, &strings.Builder{}, nil)
	seedRecord(t, cfg, ModeOpenCode, 30*time.Second) // inside the 2 minute skew
	access, account, err := NewSource(cfg).Token(t.Context())
	if err != nil || access != "access-1" || s.refreshCount != 1 {
		t.Fatalf("%q %v %d", access, err, s.refreshCount)
	}
	if account != "acct-42" {
		t.Errorf("account id should survive refresh, got %q", account)
	}
	if f := s.tokenForms[0]; f.Get("resource") != "" {
		t.Errorf("opencode refresh must not send resource: %v", f)
	}
}

func TestTerminalRefreshErrorRequiresLogin(t *testing.T) {
	s := newStub(t)
	s.onRefresh = func(map[string][]string) (int, string) {
		return 400, `{"error":"invalid_grant","error_description":"secret-detail"}`
	}
	cfg := testConfig(t, s, ModeOfficial, &strings.Builder{}, nil)
	seedRecord(t, cfg, ModeOfficial, -time.Hour)
	_, _, err := NewSource(cfg).Token(t.Context())
	if !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "secret-detail") || strings.Contains(err.Error(), "old-refresh") {
		t.Errorf("error leaked response body or token: %v", err)
	}
	rec, _ := LoadRecord(cfg.StorePath)
	if rec.RefreshToken != "old-refresh" {
		t.Errorf("credentials must be left alone on failure")
	}
}

// Another fonso process refreshes first: our rotating refresh token is then
// "reused". We must adopt the winner's tokens from disk, not fail.
func TestRefreshRaceAdoptsWinnersTokens(t *testing.T) {
	s := newStub(t)
	cfg := testConfig(t, s, ModeOfficial, &strings.Builder{}, nil)
	seedRecord(t, cfg, ModeOfficial, -time.Hour)
	s.onRefresh = func(map[string][]string) (int, string) {
		_ = SaveRecord(cfg.StorePath, Record{
			Mode: ModeOfficial, Issuer: cfg.Issuer, ClientID: "oaiapp_issued_1",
			AccessToken: "winner-access", RefreshToken: "winner-refresh", ExpiresAt: time.Now().Add(time.Hour),
		})
		return 400, `{"error":"refresh_token_reused"}`
	}
	access, _, err := NewSource(cfg).Token(t.Context())
	if err != nil || access != "winner-access" {
		t.Fatalf("access=%q err=%v", access, err)
	}
}

func TestRefreshWithStaleTokenSkipsNetworkWhenAlreadyReplaced(t *testing.T) {
	s := newStub(t)
	cfg := testConfig(t, s, ModeOfficial, &strings.Builder{}, nil)
	seedRecord(t, cfg, ModeOfficial, time.Hour)
	src := NewSource(cfg)

	access, _, err := src.Refresh(t.Context(), "some-older-rejected-token")
	if err != nil || access != "old-access" || s.refreshCount != 0 {
		t.Fatalf("%q %v %d", access, err, s.refreshCount)
	}
	access, _, err = src.Refresh(t.Context(), "old-access") // the one the server rejected
	if err != nil || access != "access-1" || s.refreshCount != 1 {
		t.Fatalf("%q %v %d", access, err, s.refreshCount)
	}
}

func TestModeMismatchAndMissingFileNeedLogin(t *testing.T) {
	s := newStub(t)
	cfg := testConfig(t, s, ModeOfficial, &strings.Builder{}, nil)
	if _, _, err := NewSource(cfg).Token(t.Context()); !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("missing file: %v", err)
	}
	seedRecord(t, cfg, ModeOpenCode, time.Hour)
	if _, _, err := NewSource(cfg).Token(t.Context()); !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("mode mismatch: %v", err)
	}
	if _, err := os.Stat(cfg.StorePath); err != nil {
		t.Fatal(err)
	}
}
