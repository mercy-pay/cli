package params

import (
	"encoding/json"
	"testing"
)

func TestParse(t *testing.T) {
	body, err := Parse([]string{
		"name=Ada",
		"phone=237670000012",
		"amount:=1000",
		"metadata[order]=4821",
		"events[]=payment.succeeded",
		"events[]=payment.failed",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := json.Marshal(body)
	want := `{"amount":1000,"events":["payment.succeeded","payment.failed"],"metadata":{"order":"4821"},"name":"Ada","phone":"237670000012"}`
	if string(got) != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	for _, bad := range [][]string{{"noequals"}, {"amount:=abc"}, {"a=1", "a[b]=2"}} {
		if _, err := Parse(bad); err == nil {
			t.Fatalf("%v should fail", bad)
		}
	}
}

func TestQuery(t *testing.T) {
	query, err := Query([]string{"limit=5", "created[gte]=2026-11-01"})
	if err != nil {
		t.Fatal(err)
	}
	if query.Encode() != "created%5Bgte%5D=2026-11-01&limit=5" {
		t.Fatalf("got %s", query.Encode())
	}
}
