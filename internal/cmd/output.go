package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// The --output formats: json (indented, the default), compact (one line, for scripts and
// jq), table (lists as columns, an object as key/value lines).
const (
	outputJSON    = "json"
	outputCompact = "compact"
	outputTable   = "table"
)

// Columns shown first in a table, in this order, when the objects have them.
var preferredColumns = []string{"id", "status", "amount", "currency", "net_amount", "payment_method", "mobile_money_provider", "type", "name", "email", "url", "created"}

const maxColumns = 6

func render(out io.Writer, body []byte, format string) {
	switch format {
	case outputCompact:
		var compact bytes.Buffer
		if json.Compact(&compact, body) != nil {
			out.Write(body)
			fmt.Fprintln(out)
			return
		}
		fmt.Fprintln(out, readable(compact.String()))
	case outputTable:
		if !printTable(out, body) {
			printJSON(out, body)
		}
	default:
		printJSON(out, body)
	}
}

// printTable writes a list or an object as aligned text. False when the body is neither.
func printTable(out io.Writer, body []byte) bool {
	var decoded map[string]any
	if json.Unmarshal(body, &decoded) != nil {
		return false
	}

	if rows, ok := decoded["data"].([]any); ok {
		printRows(out, rows)
		if more, _ := decoded["has_more"].(bool); more {
			if cursor, _ := decoded["next_cursor"].(string); cursor != "" {
				fmt.Fprintf(out, "\nLa suite : -d starting_after=%s\n", cursor)
			}
		}
		return true
	}

	object := decoded
	if inner, ok := decoded["data"].(map[string]any); ok {
		object = inner
	}
	if _, isError := object["error"]; isError {
		return false
	}

	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return rank(keys[i]) < rank(keys[j]) || (rank(keys[i]) == rank(keys[j]) && keys[i] < keys[j])
	})
	for _, key := range keys {
		fmt.Fprintf(writer, "%s\t%s\n", key, cell(object[key]))
	}
	writer.Flush()
	return true
}

func printRows(out io.Writer, rows []any) {
	if len(rows) == 0 {
		fmt.Fprintln(out, "Aucun résultat.")
		return
	}
	first, _ := rows[0].(map[string]any)
	var columns []string
	for _, column := range preferredColumns {
		if _, ok := first[column]; ok && len(columns) < maxColumns {
			columns = append(columns, column)
		}
	}
	if len(columns) == 0 {
		for column, value := range first {
			if isScalar(value) && len(columns) < maxColumns {
				columns = append(columns, column)
			}
		}
		sort.Strings(columns)
	}

	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, strings.ToUpper(strings.Join(columns, "\t")))
	for _, row := range rows {
		object, _ := row.(map[string]any)
		values := make([]string, len(columns))
		for i, column := range columns {
			values[i] = cell(object[column])
		}
		fmt.Fprintln(writer, strings.Join(values, "\t"))
	}
	writer.Flush()
}

func rank(key string) int {
	for i, column := range preferredColumns {
		if column == key {
			return i
		}
	}
	return len(preferredColumns)
}

func isScalar(value any) bool {
	switch value.(type) {
	case string, float64, bool, nil:
		return true
	}
	return false
}

// cell is one value in a column: dates shortened, numbers without decimals, objects inline.
func cell(value any) string {
	switch typed := value.(type) {
	case nil:
		return "-"
	case string:
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed.Local().Format("2006-01-02 15:04")
		}
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%g", typed)
	case bool:
		if typed {
			return "oui"
		}
		return "non"
	default:
		encoded, _ := json.Marshal(typed)
		text := []rune(readable(string(encoded)))
		if len(text) > 60 {
			return string(text[:57]) + "..."
		}
		return string(text)
	}
}
