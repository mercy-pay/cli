package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrintJSONIsReadableAndStillValid(t *testing.T) {
	var out bytes.Buffer
	printJSON(&out, []byte(`{"url":"\/api\/v1\/payments","message":"Le champ doit \u00eatre valide \ud83d\ude00","quote":"\u0022"}`))

	want := "{\n  \"url\": \"/api/v1/payments\",\n  \"message\": \"Le champ doit être valide 😀\",\n  \"quote\": \"\\u0022\"\n}\n"
	if out.String() != want {
		t.Fatalf("got:\n%s", out.String())
	}
	if !json.Valid(out.Bytes()) {
		t.Fatal("output must stay valid JSON")
	}
}
