package openai

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// stubIssuer plays auth.openai.com for both the documented ("official") and
// opencode flows. No test in this package touches the network.
type stubIssuer struct {
	t   *testing.T
	srv *httptest.Server
	key *rsa.PrivateKey

	mu sync.Mutex
	// knobs
	badState      bool
	omitScope     bool // drop chatgpt.tokens.use.direct from the token response
	wrongNonce    bool
	signWithOther *rsa.PrivateKey
	subject       string
	onRefresh     func(form map[string][]string) (status int, body string) // optional override
	// observations
	authQueries   []url.Values
	tokenForms    []url.Values
	challenge     string
	nonce         string
	issuedClient  string
	refreshCount  int
	exchangeCount int
	seq           int
}

func newStub(t *testing.T) *stubIssuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s := &stubIssuer{t: t, key: key, subject: "user-123", issuedClient: "oaiapp_issued_1"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"jwks_uri": s.srv.URL + "/jwks"})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
			"n": b64(key.N.Bytes()), "e": b64([]byte{1, 0, 1}),
		}}})
	})
	authorize := func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		q := r.URL.Query()
		s.authQueries = append(s.authQueries, q)
		s.challenge = q.Get("code_challenge")
		s.nonce = q.Get("nonce")
		redirect, _ := url.Parse(q.Get("redirect_uri"))
		out := redirect.Query()
		out.Set("code", "code-abc")
		out.Set("state", q.Get("state"))
		if s.badState {
			out.Set("state", "attacker-state")
		}
		if q.Get("client_id") == DynamicClientID {
			out.Set("client_id", s.issuedClient)
		}
		redirect.RawQuery = out.Encode()
		http.Redirect(w, r, redirect.String(), http.StatusFound)
	}
	mux.HandleFunc("/api/accounts/authorize", authorize)
	mux.HandleFunc("/oauth/authorize", authorize)
	token := func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", 400)
			return
		}
		form := r.PostForm
		s.tokenForms = append(s.tokenForms, form)
		w.Header().Set("Content-Type", "application/json")
		switch form.Get("grant_type") {
		case "authorization_code":
			s.exchangeCount++
			sum := sha256.Sum256([]byte(form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(sum[:]) != s.challenge || form.Get("code") != "code-abc" {
				w.WriteHeader(400)
				fmt.Fprint(w, `{"error":"invalid_grant"}`)
				return
			}
			clientID := form.Get("client_id")
			nonce := s.nonce
			if s.wrongNonce {
				nonce = "not-the-nonce"
			}
			scope := "chatgpt.tokens.use.direct email offline_access openid profile resource.invoke"
			if s.omitScope {
				scope = "email offline_access openid profile"
			}
			s.seq++
			resp := map[string]any{
				"access_token":  fmt.Sprintf("access-%d", s.seq),
				"refresh_token": fmt.Sprintf("refresh-%d", s.seq),
				"id_token":      s.idToken(clientID, nonce),
				"token_type":    "Bearer",
				"expires_in":    3600,
				"scope":         scope,
			}
			json.NewEncoder(w).Encode(resp)
		case "refresh_token":
			s.refreshCount++
			if s.onRefresh != nil {
				status, body := s.onRefresh(form)
				w.WriteHeader(status)
				fmt.Fprint(w, body)
				return
			}
			s.seq++
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  fmt.Sprintf("access-%d", s.seq),
				"refresh_token": fmt.Sprintf("refresh-%d", s.seq),
				"expires_in":    3600,
			})
		default:
			w.WriteHeader(400)
			fmt.Fprint(w, `{"error":"unsupported_grant_type"}`)
		}
	}
	mux.HandleFunc("/api/accounts/oauth/token", token)
	mux.HandleFunc("/oauth/token", token)
	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)
	return s
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (s *stubIssuer) idToken(clientID, nonce string) string {
	claims := map[string]any{
		"iss": s.srv.URL, "aud": clientID, "sub": s.subject, "email": "dev@example.com",
		"exp": time.Now().Add(time.Hour).Unix(), "nonce": nonce,
		"https://api.openai.com/auth": map[string]string{"chatgpt_account_id": "acct-42"},
	}
	hdr, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "k1", "typ": "JWT"})
	pl, _ := json.Marshal(claims)
	signed := b64(hdr) + "." + b64(pl)
	key := s.key
	if s.signWithOther != nil {
		key = s.signWithOther
	}
	sum := sha256.Sum256([]byte(signed))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		s.t.Fatal(err)
	}
	return signed + "." + b64(sig)
}

// browserFollowing is the stand-in "browser": it GETs the authorization URL and
// follows the redirect into fonso's loopback listener.
func browserFollowing(t *testing.T, seen *[]string) func(string) error {
	return func(u string) error {
		if seen != nil {
			*seen = append(*seen, u)
		}
		go func() {
			resp, err := http.Get(u)
			if err != nil {
				t.Logf("stub browser: %v", err)
				return
			}
			resp.Body.Close()
		}()
		return nil
	}
}

func testConfig(t *testing.T, s *stubIssuer, mode Mode, out *strings.Builder, browser func(string) error) Config {
	return Config{
		Mode:         mode,
		Issuer:       s.srv.URL,
		CallbackAddr: "127.0.0.1:0",
		StorePath:    filepath.Join(t.TempDir(), "cfg", "openai-auth.json"),
		OpenBrowser:  browser,
		Out:          out,
		LoginTimeout: 10 * time.Second,
	}
}

func rsaKey() (*rsa.PrivateKey, error) { return rsa.GenerateKey(rand.Reader, 2048) }
