// Package params turns `-d` flags into a request body or a query string.
//
//	-d name=Ada                  a string
//	-d amount:=1000              raw JSON: a number, true, null, an array or an object
//	-d metadata[order]=4821      a nested object
//	-d events[]=payment.failed   an array item
package params

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Parse builds a JSON object from the flags.
func Parse(pairs []string) (map[string]any, error) {
	body := map[string]any{}

	for _, pair := range pairs {
		key, raw, isJSON, err := split(pair)
		if err != nil {
			return nil, err
		}

		var value any = raw
		if isJSON {
			if err := json.Unmarshal([]byte(raw), &value); err != nil {
				return nil, fmt.Errorf("%s : JSON invalide après :=", key)
			}
		}

		if err := set(body, path(key), value); err != nil {
			return nil, fmt.Errorf("%s : %w", key, err)
		}
	}

	return body, nil
}

// Query encodes the flags as a query string, keeping the bracket syntax the API reads.
func Query(pairs []string) (url.Values, error) {
	query := url.Values{}
	for _, pair := range pairs {
		key, raw, _, err := split(pair)
		if err != nil {
			return nil, err
		}
		query.Add(key, raw)
	}
	return query, nil
}

func split(pair string) (key, value string, isJSON bool, err error) {
	index := strings.Index(pair, "=")
	if index <= 0 {
		return "", "", false, fmt.Errorf("%q : attendu cle=valeur ou cle:=json", pair)
	}
	key, value = pair[:index], pair[index+1:]
	if strings.HasSuffix(key, ":") {
		return strings.TrimSuffix(key, ":"), value, true, nil
	}
	return key, value, false, nil
}

// path splits "metadata[order]" into ["metadata", "order"] and "events[]" into ["events", ""].
func path(key string) []string {
	head, rest, found := strings.Cut(key, "[")
	parts := []string{head}
	for found {
		var part string
		part, rest, _ = strings.Cut(rest, "]")
		parts = append(parts, part)
		_, rest, found = strings.Cut(rest, "[")
	}
	return parts
}

func set(node map[string]any, keys []string, value any) error {
	key := keys[0]

	if len(keys) == 1 {
		node[key] = value
		return nil
	}

	if keys[1] == "" {
		list, _ := node[key].([]any)
		node[key] = append(list, value)
		return nil
	}

	child, ok := node[key].(map[string]any)
	if !ok {
		if _, exists := node[key]; exists {
			return fmt.Errorf("déjà défini avec une autre forme")
		}
		child = map[string]any{}
		node[key] = child
	}
	return set(child, keys[1:], value)
}
