package cmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mercy-pay/cli/internal/api"
	"github.com/mercy-pay/cli/internal/signature"
)

func TestListenForwardsNewEventsSignedOldestFirst(t *testing.T) {
	var mu sync.Mutex
	var received []string

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !signature.Verify(r.Header.Get("MercyPay-Signature"), body, "whsec_local", time.Now()) {
			t.Errorf("bad signature for %s", body)
		}
		mu.Lock()
		received = append(received, r.Header.Get("MercyPay-Event-Id"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer local.Close()

	calls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch {
		case r.URL.Query().Get("ending_before") == "":
			w.Write([]byte(`{"has_more":false,"data":[{"id":"evt_old","object":"event","type":"payment.succeeded","data":{}}]}`))
		case r.URL.Query().Get("ending_before") == "evt_old":
			w.Write([]byte(`{"has_more":false,"data":[
				{"id":"evt_2","object":"event","type":"payment.failed","data":{}},
				{"id":"evt_1","object":"event","type":"payment.succeeded","data":{}}]}`))
		default:
			w.Write([]byte(`{"has_more":false,"data":[]}`))
		}
	}))
	defer remote.Close()

	var out bytes.Buffer
	client := api.New(remote.URL, "mp_sandbox_x", "test")
	l := &listener{
		client: client, target: local.URL, secret: "whsec_local", out: &out,
		http: local.Client(), poll: time.Millisecond, nowUnix: func() int64 { return time.Now().Unix() },
	}

	ctx := context.Background()
	if err := l.start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := l.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if err := l.tick(ctx); err != nil {
		t.Fatal(err)
	}

	if strings.Join(received, ",") != "evt_1,evt_2" {
		t.Fatalf("received %v, output:\n%s", received, out.String())
	}
	if l.cursor != "evt_2" || !strings.Contains(out.String(), "[200]") {
		t.Fatalf("cursor %s, output:\n%s", l.cursor, out.String())
	}
}

func TestListenFiltersTypes(t *testing.T) {
	var out bytes.Buffer
	l := &listener{types: []string{"payout.paid"}, out: &out}
	l.handle(context.Background(), event{ID: "evt_1", Type: "payment.succeeded"})
	l.handle(context.Background(), event{ID: "evt_2", Type: "payout.paid"})
	if strings.Contains(out.String(), "evt_1") || !strings.Contains(out.String(), "evt_2") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestNormalizeTarget(t *testing.T) {
	cases := map[string]string{
		"localhost:3000/hook":       "http://localhost:3000/hook",
		":8000/webhooks":            "http://localhost:8000/webhooks",
		"https://app.test/webhooks": "https://app.test/webhooks",
	}
	for in, want := range cases {
		if got, err := normalizeTarget(in); err != nil || got != want {
			t.Errorf("%s: got %s (%v), want %s", in, got, err, want)
		}
	}
	if _, err := normalizeTarget("ftp://x"); err == nil {
		t.Error("ftp must be refused")
	}
}
