package cli

import (
	"fmt"
	"sort"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
	"github.com/DishanRajapaksha/modbus-cli/internal/output"
	"github.com/DishanRajapaksha/modbus-cli/internal/sunspec"
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
		return output.WriteCSV(a.out, []string{"Point", "Kind", "Address", "Value", "Unit", "Raw"}, readRows(result))
	case output.FormatText:
		for _, value := range result.Values {
			if _, err := fmt.Fprintf(a.out, "%d %v\n", value.Address, value.Value); err != nil {
				return err
			}
		}
		return nil
	default:
		return output.WriteTable(a.out, []string{"Point", "Kind", "Address", "Value", "Unit", "Raw"}, readRows(result))
	}
}

func readRows(result modbusclient.ReadResult) [][]string {
	rows := make([][]string, 0, len(result.Values))
	for _, value := range result.Values {
		rows = append(rows, []string{firstNonEmpty(value.Point, result.Point), result.Kind, fmt.Sprint(value.Address), fmt.Sprint(value.Value), firstNonEmpty(value.Unit, result.Unit), value.Raw})
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
		return output.WriteCSV(a.out, []string{"Point", "Kind", "Address", "Quantity", "Values", "Unit", "DryRun", "Sent"}, [][]string{writeRow(result)})
	case output.FormatText:
		return output.WriteText(a.out, fmt.Sprintf("%s address=%d quantity=%d values=%v dry_run=%t sent=%t", result.Kind, result.Address, result.Quantity, result.Values, result.DryRun, result.Sent))
	default:
		return output.WriteTable(a.out, []string{"Point", "Kind", "Address", "Quantity", "Values", "Unit", "Dry run", "Sent"}, [][]string{writeRow(result)})
	}
}

func writeRow(result modbusclient.WriteResult) []string {
	return []string{result.Point, result.Kind, fmt.Sprint(result.Address), fmt.Sprint(result.Quantity), fmt.Sprint(result.Values), result.Unit, fmt.Sprint(result.DryRun), fmt.Sprint(result.Sent)}
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

func (a *App) renderPoints(format string, points []config.PointConfig) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return output.WriteJSON(a.out, points)
	case output.FormatJSONL:
		for _, point := range points {
			if err := output.WriteJSONLine(a.out, point); err != nil {
				return err
			}
		}
		return nil
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Name", "Kind", "Address", "Quantity", "Type", "Unit", "Writable"}, pointRows(points))
	case output.FormatText:
		for _, point := range points {
			if _, err := fmt.Fprintf(a.out, "%s %s address=%d quantity=%d type=%s unit=%s writable=%t\n", point.Name, point.Kind, point.Address, point.Quantity, point.Type, point.Unit, point.Writable); err != nil {
				return err
			}
		}
		return nil
	default:
		return output.WriteTable(a.out, []string{"Name", "Kind", "Address", "Quantity", "Type", "Unit", "Writable"}, pointRows(points))
	}
}

func (a *App) renderSunSpecScan(format string, result sunspec.ScanResult) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return output.WriteJSON(a.out, result)
	case output.FormatJSONL:
		for _, row := range sunspecRows(result) {
			if err := output.WriteJSONLine(a.out, map[string]string{
				"device":  row[0],
				"model":   row[1],
				"block":   row[2],
				"address": row[3],
				"length":  row[4],
				"points":  row[5],
			}); err != nil {
				return err
			}
		}
		return nil
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Device", "Model", "Block", "Address", "Length", "Points"}, sunspecRows(result))
	case output.FormatText:
		for _, row := range sunspecRows(result) {
			if _, err := fmt.Fprintf(a.out, "device=%s model=%s block=%s address=%s length=%s points=%s\n", row[0], row[1], row[2], row[3], row[4], row[5]); err != nil {
				return err
			}
		}
		return nil
	default:
		return output.WriteTable(a.out, []string{"Device", "Model", "Block", "Address", "Length", "Points"}, sunspecRows(result))
	}
}

func (a *App) renderSunSpecRead(format string, result sunspec.ReadResult) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return output.WriteJSON(a.out, result)
	case output.FormatJSONL:
		return output.WriteJSONLine(a.out, result)
	case output.FormatCSV:
		return output.WriteCSV(a.out, []string{"Model", "Point", "Type", "Value"}, [][]string{sunspecReadRow(result)})
	case output.FormatText:
		return output.WriteText(a.out, fmt.Sprintf("model=%d point=%s type=%s value=%v", result.Model, result.Point, result.Type, result.Value))
	default:
		return output.WriteTable(a.out, []string{"Model", "Point", "Type", "Value"}, [][]string{sunspecReadRow(result)})
	}
}

func sunspecRows(result sunspec.ScanResult) [][]string {
	rows := [][]string{}
	for _, device := range result.Devices {
		for _, model := range device.Models {
			for _, block := range model.Blocks {
				rows = append(rows, []string{
					fmt.Sprint(device.Index),
					fmt.Sprint(model.ID),
					fmt.Sprint(block.Index),
					fmt.Sprint(block.Address),
					fmt.Sprint(block.Length),
					fmt.Sprint(len(block.Points)),
				})
			}
		}
	}
	return rows
}

func sunspecReadRow(result sunspec.ReadResult) []string {
	return []string{fmt.Sprint(result.Model), result.Point, result.Type, fmt.Sprint(result.Value)}
}

func pointRows(points []config.PointConfig) [][]string {
	rows := make([][]string, 0, len(points))
	for _, point := range points {
		rows = append(rows, []string{point.Name, point.Kind, fmt.Sprint(point.Address), fmt.Sprint(point.Quantity), point.Type, point.Unit, fmt.Sprint(point.Writable)})
	}
	return rows
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
		return output.WriteCSV(a.out, []string{"Point", "Kind", "Address", "Value", "Unit", "Raw"}, readRows(result))
	default:
		for _, value := range result.Values {
			if _, err := fmt.Fprintf(a.out, "%s %d %v\n", result.Timestamp.Format("2006-01-02T15:04:05Z07:00"), value.Address, value.Value); err != nil {
				return err
			}
		}
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
