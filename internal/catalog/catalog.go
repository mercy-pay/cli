// Package catalog holds the open countries, operators and event types, copied from
// docs.mercy-pay.com/snippets/catalog.json, for completion and suggestions.
package catalog

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog.json
var raw []byte

type file struct {
	Countries []struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Currency  string `json:"currency"`
		Providers []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"providers"`
	} `json:"countries"`
	EventTypes []string `json:"event_types"`
}

func load() file {
	var catalog file
	_ = json.Unmarshal(raw, &catalog)
	return catalog
}

// Providers are the operator codes, with their name and country for completion menus.
func Providers() []string {
	var codes []string
	for _, country := range load().Countries {
		for _, provider := range country.Providers {
			codes = append(codes, provider.Code+"\t"+provider.Name+", "+country.Name)
		}
	}
	return codes
}

// EventTypes are every webhook event type.
func EventTypes() []string {
	return load().EventTypes
}
