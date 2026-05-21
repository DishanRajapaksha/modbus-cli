package cli

import (
	"fmt"
	"sort"

	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
	"github.com/DishanRajapaksha/modbus-cli/internal/output"
)

func (a *App) renderDiagnostic(format string, result modbusclient.DiagnosticResult) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return output.WriteJSON(a.out, result)
	case output.FormatJSONL:
		return output.WriteJSONLine(a.out, result)
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Transport", "Address", "UnitID", "Result", "Error"}, [][]string{{result.Transport, result.Address, fmt.Sprint(result.UnitID), result.Result, result.Error}})
	case output.FormatText:
		if result.Error != "" {
			return output.WriteText(a.out, fmt.Sprintf("%s: %s", result.Result, result.Error))
		}
		return output.WriteText(a.out, result.Result)
	default:
		return output.WriteTable(a.out, []string{"Transport", "Address", "UnitID", "Result", "Error"}, [][]string{{result.Transport, result.Address, fmt.Sprint(result.UnitID), result.Result, result.Error}})
	}
}

func (a *App) renderRead(format string, result modbusclient.ReadResult) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return output.WriteJSON(a.out, result)
	case output.FormatJSONL:
		for _, value := range result.Values {
			if err := output.WriteJSONLine(a.out, value); err != nil {
				return err
			}
		}
		return nil
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Kind", "Address", "Value", "Raw"}, readRows(result))
	case output.FormatText:
		for _, value := range result.Values {
			if _, err := fmt.Fprintf(a.out, "%d %v\n", value.Address, value.Value); err != nil {
				return err
			}
		}
		return nil
	default:
		return output.WriteTable(a.out, []string{"Kind", "Address", "Value", "Raw"}, readRows(result))
	}
}

func readRows(result modbusclient.ReadResult) [][]string {
	rows := make([][]string, 0, len(result.Values))
	for _, value := range result.Values {
		rows = append(rows, []string{result.Kind, fmt.Sprint(value.Address), fmt.Sprint(value.Value), value.Raw})
	}
	return rows
}

func (a *App) renderWrite(format string, result modbusclient.WriteResult) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return output.WriteJSON(a.out, result)
	case output.FormatJSONL:
		return output.WriteJSONLine(a.out, result)
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Kind", "Address", "Quantity", "Values", "DryRun", "Sent"}, [][]string{writeRow(result)})
	case output.FormatText:
		return output.WriteText(a.out, fmt.Sprintf("%s address=%d quantity=%d values=%v dry_run=%t sent=%t", result.Kind, result.Address, result.Quantity, result.Values, result.DryRun, result.Sent))
	default:
		return output.WriteTable(a.out, []string{"Kind", "Address", "Quantity", "Values", "Dry run", "Sent"}, [][]string{writeRow(result)})
	}
}

func writeRow(result modbusclient.WriteResult) []string {
	return []string{result.Kind, fmt.Sprint(result.Address), fmt.Sprint(result.Quantity), fmt.Sprint(result.Values), fmt.Sprint(result.DryRun), fmt.Sprint(result.Sent)}
}

func (a *App) renderIdentification(format string, result modbusclient.IdentificationResult) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return output.WriteJSON(a.out, result)
	case output.FormatJSONL:
		for key, value := range result.Objects {
			if err := output.WriteJSONLine(a.out, map[string]string{"object": key, "value": value}); err != nil {
				return err
			}
		}
		return nil
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Object", "Value"}, identificationRows(result))
	case output.FormatText:
		for _, row := range identificationRows(result) {
			if _, err := fmt.Fprintf(a.out, "%s: %s\n", row[0], row[1]); err != nil {
				return err
			}
		}
		return nil
	default:
		return output.WriteTable(a.out, []string{"Object", "Value"}, identificationRows(result))
	}
}

func identificationRows(result modbusclient.IdentificationResult) [][]string {
	keys := make([]string, 0, len(result.Objects))
	for key := range result.Objects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, result.Objects[key]})
	}
	return rows
}

func (a *App) renderWatch(format string, result modbusclient.ReadResult) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSONL:
		return output.WriteJSONLine(a.out, result)
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Kind", "Address", "Value", "Raw"}, readRows(result))
	default:
		for _, value := range result.Values {
			if _, err := fmt.Fprintf(a.out, "%s %d %v\n", result.Timestamp.Format("2006-01-02T15:04:05Z07:00"), value.Address, value.Value); err != nil {
				return err
			}
		}
		return nil
	}
}
