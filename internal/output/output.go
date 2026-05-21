package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

const (
	FormatTable = "table"
	FormatText  = "text"
	FormatJSON  = "json"
	FormatJSONL = "jsonl"
	FormatCSV   = "csv"
)

func NormaliseFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", FormatTable:
		return FormatTable
	case FormatText:
		return FormatText
	case FormatJSON:
		return FormatJSON
	case FormatJSONL:
		return FormatJSONL
	case FormatCSV:
		return FormatCSV
	default:
		return value
	}
}

func WriteJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func WriteJSONLine(w io.Writer, value any) error {
	return json.NewEncoder(w).Encode(value)
}

func WriteText(w io.Writer, value any) error {
	_, err := fmt.Fprintln(w, value)
	return err
}

func WriteTable(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func WriteCSV(w io.Writer, headers []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
