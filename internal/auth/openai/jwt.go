package openai

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// claims decoded from a JWT payload (no signature check; see verifyIDToken).
type claims struct {
	Issuer           string           `json:"iss"`
	Subject          string           `json:"sub"`
	Audience         audience         `json:"aud"`
	Expiry           float64          `json:"exp"`
	Nonce            string           `json:"nonce"`
	Email            string           `json:"email"`
	ChatGPTAccountID string           `json:"chatgpt_account_id"`
	Organizations    []organization   `json:"organizations"`
	OpenAIAuth       *openAIAuthClaim `json:"https://api.openai.com/auth"`
}

type organization struct {
	ID string `json:"id"`
}

type openAIAuthClaim struct {
	ChatGPTAccountID string `json:"chatgpt_account_id"`
}

// audience accepts either a string or an array of strings.
type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*a = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*a = many
	return nil
}

func (a audience) has(v string) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

func splitJWT(token string) (header, payload, sig []byte, signed string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, nil, nil, "", errors.New("malformed JWT")
	}
	dec := base64.RawURLEncoding
	if header, err = dec.DecodeString(parts[0]); err != nil {
		return nil, nil, nil, "", err
	}
	if payload, err = dec.DecodeString(parts[1]); err != nil {
		return nil, nil, nil, "", err
	}
	if sig, err = dec.DecodeString(parts[2]); err != nil {
		return nil, nil, nil, "", err
	}
	return header, payload, sig, parts[0] + "." + parts[1], nil
}

// parseClaims decodes a JWT payload WITHOUT verifying it. Only use for
// non-security data such as the ChatGPT account id.
func parseClaims(token string) (claims, bool) {
	var c claims
	_, payload, _, _, err := splitJWT(token)
	if err != nil || json.Unmarshal(payload, &c) != nil {
		return claims{}, false
	}
	return c, true
}

// accountID mirrors opencode's extraction order: root chatgpt_account_id,
// nested https://api.openai.com/auth claim, then first organization id.
func (c claims) accountID() string {
	switch {
	case c.ChatGPTAccountID != "":
		return c.ChatGPTAccountID
	case c.OpenAIAuth != nil && c.OpenAIAuth.ChatGPTAccountID != "":
		return c.OpenAIAuth.ChatGPTAccountID
	case len(c.Organizations) > 0:
		return c.Organizations[0].ID
	}
	return ""
}

func extractAccountID(idToken, accessToken string) string {
	for _, t := range []string{idToken, accessToken} {
		if t == "" {
			continue
		}
		if c, ok := parseClaims(t); ok {
			if id := c.accountID(); id != "" {
				return id
			}
		}
	}
	return ""
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (c Config) fetchJSON(url string, out any) error {
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// verifyIDToken validates an ID token per OpenAI's docs: signature against the
// issuer's JWKS (RS256 or ES256), issuer, audience == issued client id,
// expiry, and the nonce for this attempt.
func (c Config) verifyIDToken(token, clientID, nonce string) (claims, error) {
	hdrRaw, payload, sig, signed, err := splitJWT(token)
	if err != nil {
		return claims{}, err
	}
	var hdr struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(hdrRaw, &hdr); err != nil {
		return claims{}, err
	}

	var disco struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := c.fetchJSON(c.Issuer+"/.well-known/openid-configuration", &disco); err != nil {
		return claims{}, fmt.Errorf("openid configuration: %w", err)
	}
	if disco.JWKSURI == "" {
		return claims{}, errors.New("openid configuration has no jwks_uri")
	}
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := c.fetchJSON(disco.JWKSURI, &set); err != nil {
		return claims{}, fmt.Errorf("jwks: %w", err)
	}
	var key *jwk
	for i := range set.Keys {
		if set.Keys[i].Kid == hdr.Kid {
			key = &set.Keys[i]
			break
		}
	}
	if key == nil {
		return claims{}, fmt.Errorf("no JWKS key for kid %q", hdr.Kid)
	}

	digest := sha256.Sum256([]byte(signed))
	switch hdr.Alg {
	case "RS256":
		n, e := b64Big(key.N), b64Big(key.E)
		if n == nil || e == nil || key.Kty != "RSA" {
			return claims{}, errors.New("bad RSA key")
		}
		pub := &rsa.PublicKey{N: n, E: int(e.Int64())}
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
			return claims{}, errors.New("ID token signature invalid")
		}
	case "ES256":
		x, y := b64Big(key.X), b64Big(key.Y)
		if x == nil || y == nil || key.Kty != "EC" || key.Crv != "P-256" || len(sig) != 64 {
			return claims{}, errors.New("bad EC key or signature")
		}
		pub := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
		if !ecdsa.Verify(pub, digest[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
			return claims{}, errors.New("ID token signature invalid")
		}
	default:
		return claims{}, fmt.Errorf("unsupported ID token alg %q", hdr.Alg)
	}

	var cl claims
	if err := json.Unmarshal(payload, &cl); err != nil {
		return claims{}, err
	}
	if strings.TrimRight(cl.Issuer, "/") != c.Issuer {
		return claims{}, errors.New("ID token issuer mismatch")
	}
	if !cl.Audience.has(clientID) {
		return claims{}, errors.New("ID token audience mismatch")
	}
	if time.Unix(int64(cl.Expiry), 0).Before(c.Now()) {
		return claims{}, errors.New("ID token expired")
	}
	if cl.Nonce != nonce {
		return claims{}, errors.New("ID token nonce mismatch")
	}
	if cl.Subject == "" {
		return claims{}, errors.New("ID token has no subject")
	}
	return cl, nil
}

func b64Big(s string) *big.Int {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(b) == 0 {
		return nil
	}
	return new(big.Int).SetBytes(b)
}
