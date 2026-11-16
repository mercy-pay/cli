// Package signature builds and checks the MercyPay-Signature header, exactly as the
// Mercy Pay servers do: v1 is the HMAC-SHA256 of "{t}.{raw body}" keyed with the secret.
package signature

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Tolerance is how far a signature's timestamp may be from now and still be accepted.
const Tolerance = 5 * time.Minute

// Compute returns the hex HMAC-SHA256 of "{timestamp}.{payload}".
func Compute(secret string, payload []byte, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", timestamp)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Header returns the full header value: "t=<unix>,v1=<hex>".
func Header(secret string, payload []byte, timestamp int64) string {
	return fmt.Sprintf("t=%d,v1=%s", timestamp, Compute(secret, payload, timestamp))
}

// Verify checks a header against the payload, accepting any v1 value (several are sent for
// 24 hours after a secret rotation).
func Verify(header string, payload []byte, secret string, now time.Time) bool {
	var timestamp int64 = -1
	var candidates []string

	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				timestamp = parsed
			}
		case "v1":
			candidates = append(candidates, value)
		}
	}

	if timestamp < 0 {
		return false
	}
	if delta := now.Sub(time.Unix(timestamp, 0)); delta > Tolerance || delta < -Tolerance {
		return false
	}

	expected := Compute(secret, payload, timestamp)
	for _, candidate := range candidates {
		if hmac.Equal([]byte(candidate), []byte(expected)) {
			return true
		}
	}
	return false
}

// NewSecret returns a fresh "whsec_" secret of 32 random characters.
func NewSecret() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i, b := range raw {
		raw[i] = alphabet[int(b)%len(alphabet)]
	}
	return "whsec_" + string(raw), nil
}
