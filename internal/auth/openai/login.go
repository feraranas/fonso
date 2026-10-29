package openai

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func randomB64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

type callbackResult struct {
	code, clientID string
	err            error
}

// Login runs the browser PKCE flow and stores the resulting credentials.
// The authorization URL is printed (without any id_token_hint) and the system
// browser is opened with the full URL; tokens are never printed.
func Login(ctx context.Context, cfg Config) (Record, error) {
	cfg = cfg.withDefaults()

	var (
		hostID   string
		existing Record
		reauth   bool
		err      error
	)
	if cfg.Mode == ModeOfficial {
		if hostID, err = loadOrCreateHostID(cfg.StorePath); err != nil {
			return Record{}, fmt.Errorf("host id: %w", err)
		}
		if rec, lerr := LoadRecord(cfg.StorePath); lerr == nil && rec.Mode == ModeOfficial && rec.ClientID != "" && rec.Issuer == cfg.Issuer {
			existing, reauth = rec, true
		}
	}

	state, err := randomB64(32)
	if err != nil {
		return Record{}, err
	}
	nonce, err := randomB64(16)
	if err != nil {
		return Record{}, err
	}
	verifier, err := randomB64(64)
	if err != nil {
		return Record{}, err
	}

	ln, err := net.Listen("tcp", cfg.CallbackAddr)
	if err != nil {
		return Record{}, fmt.Errorf("cannot listen for the OAuth callback on %s (is another login running?): %w", cfg.CallbackAddr, err)
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d%s", ln.Addr().(*net.TCPAddr).Port, callbackPath)

	results := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		res := parseCallback(r.URL.Query(), state)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if res.err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<h1>fonso login failed</h1><p>%s</p>", html.EscapeString(res.err.Error()))
		} else {
			fmt.Fprint(w, "<h1>fonso is signed in</h1><p>You can close this tab.</p>")
		}
		select {
		case results <- res:
		default:
		}
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	clientID := OpenCodeClientID
	if cfg.Mode == ModeOfficial {
		clientID = DynamicClientID
		if reauth {
			clientID = existing.ClientID
		}
	}
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge":        {pkceChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}
	if cfg.Mode == ModeOfficial {
		q.Set("scope", "openid profile email offline_access resource.invoke "+RequiredScope)
		q.Set("resource", OfficialResource)
		q.Set("nonce", nonce)
		q.Set("ext_agent_host_id", hostID)
		if reauth {
			if existing.Email != "" {
				q.Set("login_hint", existing.Email)
			}
		} else {
			q.Set("agent_name_hint", AgentName)
		}
	} else {
		q.Set("scope", "openid profile email offline_access")
		q.Set("id_token_add_organizations", "true")
		q.Set("codex_cli_simplified_flow", "true")
		q.Set("originator", AgentName)
	}
	printable := cfg.authorizeURL() + "?" + q.Encode()
	if reauth && existing.IDToken != "" {
		q.Set("id_token_hint", existing.IDToken) // only ever sent to the browser, never printed
	}
	full := cfg.authorizeURL() + "?" + q.Encode()

	fmt.Fprintf(cfg.Out, "Opening your browser to sign in with ChatGPT.\nIf it does not open, visit:\n\n  %s\n\nWaiting for the callback on %s ...\n", printable, redirectURI)
	if err := cfg.OpenBrowser(full); err != nil {
		fmt.Fprintf(cfg.Out, "(could not open a browser automatically: %v)\n", err)
	}

	var res callbackResult
	select {
	case res = <-results:
	case <-time.After(cfg.LoginTimeout):
		return Record{}, errors.New("timed out waiting for the browser sign-in")
	case <-ctx.Done():
		return Record{}, ctx.Err()
	}
	if res.err != nil {
		return Record{}, res.err
	}

	// Client id: new registrations must return an issued id; reauth must not change it.
	issuedClientID := OpenCodeClientID
	if cfg.Mode == ModeOfficial {
		switch {
		case reauth && res.clientID != "" && res.clientID != existing.ClientID:
			return Record{}, errors.New("callback returned a different client_id than the saved registration; refusing to replace it")
		case reauth:
			issuedClientID = existing.ClientID
		case res.clientID == "" || res.clientID == DynamicClientID:
			return Record{}, errors.New("registration incomplete: callback did not include an issued client_id")
		default:
			issuedClientID = res.clientID
		}
	}

	tok, err := cfg.exchangeCode(ctx, issuedClientID, res.code, verifier, redirectURI)
	if err != nil {
		return Record{}, err
	}

	rec := Record{
		Mode:           cfg.Mode,
		Issuer:         cfg.Issuer,
		ClientID:       issuedClientID,
		ExtAgentHostID: hostID,
		IDToken:        tok.IDToken,
		AccessToken:    tok.AccessToken,
		RefreshToken:   tok.RefreshToken,
		TokenType:      tok.TokenType,
		ExpiresAt:      cfg.Now().Add(tok.lifetime()),
		Scopes:         splitScopes(tok.Scope),
		SavedAt:        cfg.Now().UTC(),
	}
	if rec.RefreshToken == "" {
		return Record{}, errors.New("token response had no refresh_token (offline_access not granted?)")
	}

	if cfg.Mode == ModeOfficial {
		cl, err := cfg.verifyIDToken(tok.IDToken, issuedClientID, nonce)
		if err != nil {
			return Record{}, fmt.Errorf("ID token validation failed: %w", err)
		}
		if !hasScope(rec.Scopes, RequiredScope) {
			return Record{}, fmt.Errorf("ChatGPT plan usage was not granted (missing scope %s); re-run login and approve it", RequiredScope)
		}
		if reauth && existing.Subject != "" && existing.Subject != cl.Subject {
			return Record{}, errors.New("signed in as a different ChatGPT account than the saved registration; refusing to replace credentials")
		}
		rec.Subject, rec.Email = cl.Subject, cl.Email
	} else {
		rec.AccountID = extractAccountID(tok.IDToken, tok.AccessToken)
		if c, ok := parseClaims(tok.IDToken); ok {
			rec.Email, rec.Subject = c.Email, c.Subject
		}
	}

	if err := SaveRecord(cfg.StorePath, rec); err != nil {
		return Record{}, fmt.Errorf("saving credentials: %w", err)
	}
	return rec, nil
}

func parseCallback(q url.Values, wantState string) callbackResult {
	if q.Get("state") != wantState {
		return callbackResult{err: errors.New("OAuth callback state mismatch")}
	}
	if e := q.Get("error"); e != "" {
		return callbackResult{err: fmt.Errorf("authorization failed: %s", e)}
	}
	code := q.Get("code")
	if code == "" {
		return callbackResult{err: errors.New("OAuth callback had no code")}
	}
	return callbackResult{code: code, clientID: q.Get("client_id")}
}

func hasScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

func (t tokenResponse) lifetime() time.Duration {
	if t.ExpiresIn <= 0 {
		return time.Hour
	}
	return time.Duration(t.ExpiresIn) * time.Second
}

// tokenError carries only the OAuth error code, never a response body.
type tokenError struct {
	Status int
	Code   string
}

func (e *tokenError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("token endpoint returned %d (%s)", e.Status, e.Code)
	}
	return fmt.Sprintf("token endpoint returned %d", e.Status)
}

func (c Config) postToken(ctx context.Context, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error json.RawMessage `json:"error"`
			Code  string          `json:"code"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		code := e.Code
		var s string
		if json.Unmarshal(e.Error, &s) == nil && s != "" {
			code = s
		} else {
			var o struct {
				Code string `json:"code"`
			}
			if json.Unmarshal(e.Error, &o) == nil && o.Code != "" {
				code = o.Code
			}
		}
		return tokenResponse{}, &tokenError{Status: resp.StatusCode, Code: code}
	}
	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return tokenResponse{}, fmt.Errorf("decode token response: %w", err)
	}
	if tok.AccessToken == "" {
		return tokenResponse{}, errors.New("token response had no access_token")
	}
	return tok, nil
}

func (c Config) exchangeCode(ctx context.Context, clientID, code, verifier, redirectURI string) (tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {redirectURI},
	}
	if c.Mode == ModeOfficial {
		form.Set("resource", OfficialResource)
	}
	return c.postToken(ctx, form)
}

func openSystemBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}

func splitScopes(s string) []string { return strings.Fields(s) }
