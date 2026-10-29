package openai

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %o, want %o", path, got, want)
	}
}

func TestOfficialLoginEndToEnd(t *testing.T) {
	s := newStub(t)
	var out strings.Builder
	var opened []string
	cfg := testConfig(t, s, ModeOfficial, &out, nil)
	cfg.OpenBrowser = browserFollowing(t, &opened)

	rec, err := Login(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Authorization request carries the documented parameters.
	if len(s.authQueries) != 1 {
		t.Fatalf("authorize calls = %d", len(s.authQueries))
	}
	q := s.authQueries[0]
	for k, want := range map[string]string{
		"client_id":             DynamicClientID,
		"agent_name_hint":       "fonso",
		"resource":              OfficialResource,
		"response_type":         "code",
		"code_challenge_method": "S256",
	} {
		if q.Get(k) != want {
			t.Errorf("authorize %s = %q, want %q", k, q.Get(k), want)
		}
	}
	for _, sc := range []string{"openid", "offline_access", "resource.invoke", RequiredScope} {
		if !strings.Contains(" "+q.Get("scope")+" ", " "+sc+" ") {
			t.Errorf("scope %q missing from %q", sc, q.Get("scope"))
		}
	}
	if !strings.HasPrefix(q.Get("ext_agent_host_id"), "urn:uuid:") || q.Get("nonce") == "" || q.Get("state") == "" {
		t.Errorf("host id / nonce / state missing: %v", q)
	}
	if !strings.HasPrefix(q.Get("redirect_uri"), "http://127.0.0.1:") || !strings.HasSuffix(q.Get("redirect_uri"), "/auth/callback") {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}

	// PKCE: the stub verified sha256(verifier)==challenge during the exchange (else invalid_grant).
	if s.exchangeCount != 1 || len(s.tokenForms) != 1 {
		t.Fatalf("exchange count = %d", s.exchangeCount)
	}
	f := s.tokenForms[0]
	if f.Get("client_id") != "oaiapp_issued_1" || f.Get("resource") != OfficialResource || f.Get("redirect_uri") != q.Get("redirect_uri") {
		t.Errorf("token form = %v", f)
	}

	// Stored record.
	if rec.ClientID != "oaiapp_issued_1" || rec.AccessToken != "access-1" || rec.RefreshToken != "refresh-1" || rec.Email != "dev@example.com" || rec.Subject != "user-123" {
		t.Errorf("record = %#v", rec)
	}
	assertMode(t, cfg.StorePath, 0o600)
	assertMode(t, filepath.Dir(cfg.StorePath), 0o700)
	onDisk, err := LoadRecord(cfg.StorePath)
	if err != nil || onDisk.AccessToken != "access-1" || onDisk.ExpiresAt.IsZero() {
		t.Fatalf("on disk = %v, %v", onDisk, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(cfg.StorePath))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}

	// Nothing secret printed.
	for _, secret := range []string{"access-1", "refresh-1", onDisk.IDToken, "code-abc"} {
		if strings.Contains(out.String(), secret) {
			t.Errorf("login output leaked %q", secret[:6])
		}
	}
	if s := fmt.Sprintf("%v %+v %#v", rec, rec, rec); strings.Contains(s, "access-1") || strings.Contains(s, "refresh-1") {
		t.Errorf("Record formatting leaked a token: %s", s)
	}

	// Second login = reauthorization: saved client_id, same host id, id_token_hint
	// to the browser only, no agent_name_hint.
	hostID := q.Get("ext_agent_host_id")
	out.Reset()
	opened = nil
	if _, err := Login(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	q2 := s.authQueries[1]
	if q2.Get("client_id") != "oaiapp_issued_1" || q2.Get("agent_name_hint") != "" || q2.Get("ext_agent_host_id") != hostID {
		t.Errorf("reauth query = %v", q2)
	}
	if q2.Get("id_token_hint") == "" || q2.Get("login_hint") != "dev@example.com" {
		t.Errorf("reauth hints missing: %v", q2)
	}
	if strings.Contains(out.String(), "id_token_hint") || strings.Contains(out.String(), onDisk.IDToken) {
		t.Errorf("printed URL leaked id_token_hint")
	}
	if len(opened) != 1 || !strings.Contains(opened[0], "id_token_hint=") {
		t.Errorf("browser URL should carry id_token_hint")
	}
	assertMode(t, filepath.Join(filepath.Dir(cfg.StorePath), "openai-host-id"), 0o600)
}

func TestLoginRejections(t *testing.T) {
	cases := []struct {
		name string
		tune func(*stubIssuer)
		want string
	}{
		{"state mismatch", func(s *stubIssuer) { s.badState = true }, "state mismatch"},
		{"plan scope not granted", func(s *stubIssuer) { s.omitScope = true }, "ChatGPT plan usage was not granted"},
		{"nonce mismatch", func(s *stubIssuer) { s.wrongNonce = true }, "nonce mismatch"},
		{"bad id token signature", func(s *stubIssuer) {
			k, _ := rsaKey()
			s.signWithOther = k
		}, "signature invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub(t)
			tc.tune(s)
			var out strings.Builder
			cfg := testConfig(t, s, ModeOfficial, &out, nil)
			cfg.OpenBrowser = browserFollowing(t, nil)
			_, err := Login(t.Context(), cfg)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
			if _, statErr := os.Stat(cfg.StorePath); !os.IsNotExist(statErr) {
				t.Errorf("credentials must not be written on failure")
			}
			if tc.name == "state mismatch" && s.exchangeCount != 0 {
				t.Errorf("code must not be exchanged after a state mismatch")
			}
		})
	}
}

func TestOpenCodeLoginEndToEnd(t *testing.T) {
	s := newStub(t)
	var out strings.Builder
	cfg := testConfig(t, s, ModeOpenCode, &out, nil)
	cfg.OpenBrowser = browserFollowing(t, nil)

	rec, err := Login(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	q := s.authQueries[0]
	if q.Get("client_id") != OpenCodeClientID || q.Get("originator") != "fonso" || q.Get("resource") != "" || q.Get("codex_cli_simplified_flow") != "true" {
		t.Errorf("opencode authorize query = %v", q)
	}
	if strings.Contains(q.Get("scope"), RequiredScope) {
		t.Errorf("opencode mode must not request the SIWC-only scope")
	}
	if f := s.tokenForms[0]; f.Get("client_id") != OpenCodeClientID || f.Get("resource") != "" {
		t.Errorf("token form = %v", f)
	}
	if rec.AccountID != "acct-42" || rec.Mode != ModeOpenCode {
		t.Errorf("record = %#v", rec)
	}
	assertMode(t, cfg.StorePath, 0o600)
}

func TestSaveRecordIsAtomicAnd0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d", "openai-auth.json")
	if err := SaveRecord(path, Record{AccessToken: "a", RefreshToken: "r"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveRecord(path, Record{AccessToken: "b", RefreshToken: "r2"}); err != nil {
		t.Fatal(err)
	}
	assertMode(t, path, 0o600)
	got, _ := LoadRecord(path)
	if got.AccessToken != "b" {
		t.Errorf("overwrite lost: %v", got)
	}
}

func TestAccountIDExtractionOrder(t *testing.T) {
	mk := func(payload string) string {
		return b64([]byte(`{"alg":"none"}`)) + "." + b64([]byte(payload)) + "."
	}
	cases := map[string]string{
		mk(`{"chatgpt_account_id":"root","https://api.openai.com/auth":{"chatgpt_account_id":"nested"}}`): "root",
		mk(`{"https://api.openai.com/auth":{"chatgpt_account_id":"nested"}}`):                             "nested",
		mk(`{"organizations":[{"id":"org-1"},{"id":"org-2"}]}`):                                           "org-1",
		mk(`{"email":"x"}`): "",
	}
	for tok, want := range cases {
		if got := extractAccountID(tok, ""); got != want {
			t.Errorf("extractAccountID = %q, want %q", got, want)
		}
	}
	if got := extractAccountID(mk(`{}`), mk(`{"chatgpt_account_id":"from-access"}`)); got != "from-access" {
		t.Errorf("fallback to access token = %q", got)
	}
}
