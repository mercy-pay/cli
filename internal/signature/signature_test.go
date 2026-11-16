package signature

import (
	"strings"
	"testing"
	"time"
)

// Same vector as hash_hmac('sha256', "1791022800.{body}", 'whsec_test') on the server.
const (
	vectorSecret    = "whsec_test"
	vectorPayload   = `{"id":"evt_test","object":"event"}`
	vectorTimestamp = int64(1791022800)
	vectorHex       = "52dafa4217ab4708f69eaeecb1f77e14f6405da36a477620c75328c30567e13b"
)

func TestComputeMatchesTheServer(t *testing.T) {
	if got := Compute(vectorSecret, []byte(vectorPayload), vectorTimestamp); got != vectorHex {
		t.Fatalf("got %s, want %s", got, vectorHex)
	}
}

func TestVerify(t *testing.T) {
	at := time.Unix(vectorTimestamp, 0)
	header := Header(vectorSecret, []byte(vectorPayload), vectorTimestamp)

	if !Verify(header, []byte(vectorPayload), vectorSecret, at.Add(time.Minute)) {
		t.Fatal("a fresh, correct signature must pass")
	}
	if Verify(header, []byte(vectorPayload+" "), vectorSecret, at) {
		t.Fatal("a changed body must fail")
	}
	if Verify(header, []byte(vectorPayload), vectorSecret, at.Add(6*time.Minute)) {
		t.Fatal("an old signature must fail")
	}
	rotated := "t=1791022800,v1=deadbeef,v1=" + vectorHex
	if !Verify(rotated, []byte(vectorPayload), vectorSecret, at) {
		t.Fatal("any matching v1 must pass")
	}
}

func TestNewSecret(t *testing.T) {
	secret, err := NewSecret()
	if err != nil || !strings.HasPrefix(secret, "whsec_") || len(secret) != 38 {
		t.Fatalf("unexpected secret %q (%v)", secret, err)
	}
}
