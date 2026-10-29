//go:build security

package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// These integration tests need a running management API. Set:
//
//	LINKSPIRE_API_URL=http://localhost:8081 go test -tags security ./test/security/...
func apiBase(t *testing.T) string {
	t.Helper()
	base := os.Getenv("LINKSPIRE_API_URL")
	if base == "" {
		t.Skip("set LINKSPIRE_API_URL to run API integration tests")
	}
	return base
}

var httpc = &http.Client{Timeout: 10 * time.Second}

func register(t *testing.T, base string) (token string, email string) {
	t.Helper()
	email = fmt.Sprintf("sec+%d@example.com", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]string{"email": email, "password": "sec-test-pass-1234", "home_region": "us"})
	resp, err := httpc.Post(base+"/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("register status %d: %s", resp.StatusCode, b)
	}
	var out struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Token == "" {
		t.Fatal("register returned no token")
	}
	return out.Token, email
}

func createLink(t *testing.T, base, token string) (domainID int64, code string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"destination": "https://example.com/" + fmt.Sprint(time.Now().UnixNano())})
	req, _ := http.NewRequest(http.MethodPost, base+"/v1/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpc.Do(req)
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create link status %d: %s", resp.StatusCode, b)
	}
	var out struct {
		DomainID int64  `json:"domain_id"`
		Code     string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.DomainID, out.Code
}

func getLink(t *testing.T, base, token string, domainID int64, code string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/links/%d/%s", base, domainID, code), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpc.Do(req)
	if err != nil {
		t.Fatalf("get link: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

// TestTenantIsolation: a link created in workspace A must not be readable by a
// user in workspace B, and must require authentication at all.
func TestTenantIsolation(t *testing.T) {
	base := apiBase(t)
	tokenA, _ := register(t, base)
	tokenB, _ := register(t, base)

	domainID, code := createLink(t, base, tokenA)

	if got := getLink(t, base, tokenA, domainID, code); got != 200 {
		t.Fatalf("owner A should read own link, got %d", got)
	}
	if got := getLink(t, base, tokenB, domainID, code); got != 404 {
		t.Errorf("tenant isolation breach: user B read A's link, status %d (want 404)", got)
	}
	if got := getLink(t, base, "", domainID, code); got != 401 {
		t.Errorf("unauthenticated read returned %d (want 401)", got)
	}
}

// TestRejectsMaliciousDestination: the fast-path scanner must reject dangerous
// schemes at creation (422), never persisting a row.
func TestRejectsMaliciousDestination(t *testing.T) {
	base := apiBase(t)
	token, _ := register(t, base)
	for _, dest := range []string{"javascript:alert(1)", "data:text/html,<script>", "http://169.254.169.254/latest/meta-data/"} {
		body, _ := json.Marshal(map[string]string{"destination": dest})
		req, _ := http.NewRequest(http.MethodPost, base+"/v1/links", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := httpc.Do(req)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		status := resp.StatusCode
		resp.Body.Close()
		if status < 400 {
			t.Errorf("destination %q accepted with status %d (want >=400)", dest, status)
		}
	}
}
