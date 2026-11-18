package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeAPI answers like the API from a handler and remembers each request.
type fakeAPI struct {
	mu       sync.Mutex
	requests []*http.Request
	bodies   []string
}

func (f *fakeAPI) server(t *testing.T, handle func(r *http.Request) (int, string)) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, r)
		f.bodies = append(f.bodies, string(body))
		f.mu.Unlock()
		status, answer := handle(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)
	return server
}

// run executes mercypay with these arguments in a fresh config directory.
func run(t *testing.T, baseURL string, key string, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("MERCYPAY_CONFIG_DIR", t.TempDir())
	t.Setenv("MERCYPAY_API_KEY", key)
	t.Setenv("MERCYPAY_API_BASE", baseURL)
	flags = globals{}

	root := NewRoot()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return out.String(), errOut.String(), err
}

func TestOutputFormats(t *testing.T) {
	api := &fakeAPI{}
	server := api.server(t, func(*http.Request) (int, string) {
		return 200, `{"object":"list","has_more":true,"next_cursor":"pay_2","data":[{"id":"pay_1","status":"succeeded","amount":1000,"currency":"XAF","created":"2026-11-18T09:00:00+00:00","payer":{"name":"Awa"}}]}`
	})

	table, _, err := run(t, server.URL, "mp_sandbox_x", "payments", "list", "-o", "table")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(table, "ID") || !strings.Contains(table, "pay_1") || !strings.Contains(table, "succeeded") || !strings.Contains(table, "starting_after=pay_2") {
		t.Fatalf("table:\n%s", table)
	}

	compact, _, _ := run(t, server.URL, "mp_sandbox_x", "payments", "list", "-o", "compact")
	if strings.Count(strings.TrimSpace(compact), "\n") != 0 || !json.Valid([]byte(compact)) {
		t.Fatalf("compact must be one valid JSON line:\n%s", compact)
	}
}

func TestATestPaymentPicksTheNumberAndWaitsForTheResult(t *testing.T) {
	waitFirstDelay = time.Millisecond
	defer func() { waitFirstDelay = 2 * time.Second }()
	api := &fakeAPI{}
	reads := 0
	server := api.server(t, func(r *http.Request) (int, string) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/test_helpers/mobile_money_numbers"):
			return 200, `{"object":"list","data":[
				{"country":"CMR","provider":"ORANGE_CMR","phone":"237693456789","applies_to":"payment","outcome":"succeeded","failure_code":null},
				{"country":"CMR","provider":"ORANGE_CMR","phone":"237693456049","applies_to":"payment","outcome":"failed","failure_code":"insufficient_balance"}]}`
		case r.Method == http.MethodPost:
			return 201, `{"data":{"id":"pay_1","object":"payment","status":"pending"}}`
		default:
			reads++
			return 200, `{"data":{"id":"pay_1","object":"payment","status":"failed","failure_code":"insufficient_balance"}}`
		}
	})

	out, stderr, err := run(t, server.URL, "mp_sandbox_x", "payments", "create", "--test", "insufficient_balance", "--provider", "orange_cmr", "--wait", "-o", "compact")
	if err != nil {
		t.Fatal(err, stderr)
	}

	var sent map[string]any
	_ = json.Unmarshal([]byte(api.bodies[1]), &sent)
	if sent["phone"] != "237693456049" || sent["provider"] != "ORANGE_CMR" || sent["country"] != "CMR" || sent["amount"] != float64(1000) {
		t.Fatalf("payment sent: %v", sent)
	}
	if !strings.Contains(out, `"status":"failed"`) || reads != 1 || !strings.Contains(stderr, "Statut : pending") {
		t.Fatalf("out=%s stderr=%s reads=%d", out, stderr, reads)
	}

	if _, _, err := run(t, server.URL, "mp_live_x", "payments", "create", "--test", "succeeded", "--provider", "ORANGE_CMR"); err == nil || !strings.Contains(err.Error(), "sandbox") {
		t.Fatalf("a live key must be refused, got %v", err)
	}
}

func TestInitWritesARunnableProjectWithTheSandboxKeyOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "boutique")

	if _, _, err := run(t, "http://unused", "mp_sandbox_abc", "init", "node", dir); err != nil {
		t.Fatal(err)
	}
	env, _ := os.ReadFile(filepath.Join(dir, ".env"))
	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(env), "MERCY_PAY_API_KEY=mp_sandbox_abc") || !strings.Contains(string(readme), "# boutique") {
		t.Fatalf("env=%s readme=%s", env, readme)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(".gitignore missing")
	}

	live := filepath.Join(t.TempDir(), "live")
	_, _, _ = run(t, "http://unused", "mp_live_secret", "init", "next", live)
	envLocal, _ := os.ReadFile(filepath.Join(live, ".env.local"))
	if strings.Contains(string(envLocal), "mp_live_secret") {
		t.Fatal("a live key must never be written in a sample project")
	}

	if _, _, err := run(t, "http://unused", "mp_sandbox_abc", "init", "node", dir); err == nil {
		t.Fatal("a non-empty folder must be refused without --force")
	}

	for _, stack := range stackNames() {
		if _, _, err := run(t, "http://unused", "mp_sandbox_abc", "init", stack, filepath.Join(t.TempDir(), stack)); err != nil {
			t.Fatalf("%s: %v", stack, err)
		}
	}
}

func TestOpenKnowsPagesAndResources(t *testing.T) {
	if target, _ := openTarget("payments"); target != "https://docs.mercy-pay.com/api/payments/list" {
		t.Fatal(target)
	}
	if target, _ := openTarget("dashboard"); target != "https://mercy-pay.com/dashboard" {
		t.Fatal(target)
	}
	if _, err := openTarget("nulle-part"); err == nil {
		t.Fatal("an unknown page must be refused")
	}
}

func TestConfigUseChangesTheDefaultProfile(t *testing.T) {
	_, _, _ = run(t, "http://unused", "", "config", "list")
	dir := os.Getenv("MERCYPAY_CONFIG_DIR")
	_ = os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"profiles":{"default":{"merchant_name":"A","api_key":"mp_sandbox_a"},"autre":{"merchant_name":"B","api_key":"mp_live_b"}}}`), 0o600)

	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"config", "use", "autre"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	root = NewRoot()
	out.Reset()
	root.SetOut(&out)
	root.SetArgs([]string{"config", "list"})
	_ = root.Execute()
	if !strings.Contains(out.String(), "* autre\tB\tlive") || !strings.Contains(out.String(), "  default\tA\tsandbox") {
		t.Fatalf("list:\n%s", out.String())
	}
	if flags.profile != "autre" {
		t.Fatalf("profile in use: %s", flags.profile)
	}
}
