package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// Column defines a table column for output formatting.
type Column struct {
	Header string
	Field  string                     // JSON field name
	Format func(v interface{}) string // Optional custom formatter
}

// Printer handles output formatting across table, json, and yaml modes.
type Printer struct {
	Format    string // "table", "json", "yaml"
	NoHeaders bool
	Quiet     bool
	Writer    io.Writer
}

func (p *Printer) writer() io.Writer {
	if p.Writer != nil {
		return p.Writer
	}
	return os.Stdout
}

// Print outputs data in the configured format.
// data is raw JSON bytes from the API response.
// columns define table output; ignored for json/yaml.
func (p *Printer) Print(data []byte, columns []Column) error {
	switch p.Format {
	case "json":
		return p.printJSON(data)
	case "yaml":
		return p.printYAML(data)
	default:
		return p.printTable(data, columns)
	}
}

// PrintSingle outputs a single object (not an array).
func (p *Printer) PrintSingle(data []byte, columns []Column) error {
	switch p.Format {
	case "json":
		return p.printJSON(data)
	case "yaml":
		return p.printYAML(data)
	default:
		return p.printSingleTable(data, columns)
	}
}

// PrintCreated outputs the result of a create operation.
// In quiet mode, only prints the value of idField.
func (p *Printer) PrintCreated(data []byte, idField string) error {
	if p.Quiet {
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
		if v, ok := obj[idField]; ok {
			fmt.Fprintln(p.writer(), v)
		}
		return nil
	}
	return p.printJSON(data)
}

// PrintDeleted outputs confirmation of a delete operation.
func (p *Printer) PrintDeleted(resource, id string) {
	if !p.Quiet {
		fmt.Fprintf(p.writer(), "%s %q deleted\n", resource, id)
	}
}

func (p *Printer) printJSON(data []byte) error {
	var buf json.RawMessage
	if err := json.Unmarshal(data, &buf); err != nil {
		// Not valid JSON, print raw
		fmt.Fprintln(p.writer(), string(data))
		return nil
	}
	pretty, err := json.MarshalIndent(buf, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(p.writer(), string(pretty))
	return nil
}

func (p *Printer) printYAML(data []byte) error {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Fprint(p.writer(), string(out))
	return nil
}

func (p *Printer) printTable(data []byte, columns []Column) error {
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		// Might be a single object — wrap it
		var single map[string]interface{}
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return fmt.Errorf("cannot parse response as table data: %w", err)
		}
		rows = []map[string]interface{}{single}
	}

	if p.Quiet && len(rows) > 0 {
		for _, row := range rows {
			if v, ok := row["id"]; ok {
				fmt.Fprintln(p.writer(), v)
			}
		}
		return nil
	}

	w := tabwriter.NewWriter(p.writer(), 0, 0, 2, ' ', 0)

	if !p.NoHeaders {
		headers := make([]string, len(columns))
		seps := make([]string, len(columns))
		for i, col := range columns {
			headers[i] = col.Header
			seps[i] = strings.Repeat("-", len(col.Header))
		}
		fmt.Fprintln(w, strings.Join(headers, "\t"))
		fmt.Fprintln(w, strings.Join(seps, "\t"))
	}

	for _, row := range rows {
		vals := make([]string, len(columns))
		for i, col := range columns {
			v := extractField(row, col.Field)
			if col.Format != nil {
				vals[i] = col.Format(v)
			} else {
				vals[i] = formatValue(v)
			}
		}
		fmt.Fprintln(w, strings.Join(vals, "\t"))
	}

	return w.Flush()
}

func (p *Printer) printSingleTable(data []byte, columns []Column) error {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("cannot parse response: %w", err)
	}

	if p.Quiet {
		if v, ok := obj["id"]; ok {
			fmt.Fprintln(p.writer(), v)
		}
		return nil
	}

	w := p.writer()
	for _, col := range columns {
		v := extractField(obj, col.Field)
		var s string
		if col.Format != nil {
			s = col.Format(v)
		} else {
			s = formatValue(v)
		}
		fmt.Fprintf(w, "%-20s %s\n", col.Header+":", s)
	}
	return nil
}

func extractField(obj map[string]interface{}, field string) interface{} {
	parts := strings.Split(field, ".")
	var current interface{} = obj
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}

func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}
