package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Every operation of the API reference is reachable: a resource command, or a dedicated one.
func TestEveryOperationOfTheReferenceHasACommand(t *testing.T) {
	content, err := os.ReadFile("../../testdata/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(content, &spec); err != nil {
		t.Fatal(err)
	}

	known := map[string]operation{}
	for _, op := range operations {
		known[op.OperationID] = op
	}

	for path, methods := range spec.Paths {
		for method, definition := range methods {
			if _, dedicated := dedicatedOperations[definition.OperationID]; dedicated {
				continue
			}
			op, ok := known[definition.OperationID]
			if !ok {
				t.Errorf("no command for %s (%s %s)", definition.OperationID, method, path)
				continue
			}
			if !strings.EqualFold(op.Method, method) || normalized("/api"+op.Path) != normalized(path) {
				t.Errorf("%s: command says %s %s, reference says %s %s", definition.OperationID, op.Method, op.Path, method, path)
			}
		}
	}
}

// "/api/v1/customers/{customer}/payment_methods/{payment_method}" → "/api/v1/customers/{}/payment_methods/{}".
func normalized(path string) string {
	var out strings.Builder
	inside := false
	for _, r := range path {
		switch {
		case r == '{':
			inside = true
			out.WriteString("{}")
		case r == '}':
			inside = false
		case !inside:
			out.WriteRune(r)
		}
	}
	return out.String()
}
