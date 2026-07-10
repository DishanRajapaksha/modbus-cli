from pathlib import Path

Path("internal/output/output.go").write_text(r'''package output

import (
    "encoding/csv"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "strings"
    "text/tabwriter"
)

var ErrOutput = errors.New("output error")

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
    if err := enc.Encode(value); err != nil {
        return fmt.Errorf("%w: %v", ErrOutput, err)
    }
    return nil
}

func WriteJSONLine(w io.Writer, value any) error {
    if err := json.NewEncoder(w).Encode(value); err != nil {
        return fmt.Errorf("%w: %v", ErrOutput, err)
    }
    return nil
}

func WriteText(w io.Writer, value any) error {
    if _, err := fmt.Fprintln(w, value); err != nil {
        return fmt.Errorf("%w: %v", ErrOutput, err)
    }
    return nil
}

func WriteTable(w io.Writer, headers []string, rows [][]string) error {
    tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
    if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
        return fmt.Errorf("%w: %v", ErrOutput, err)
    }
    for _, row := range rows {
        if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
            return fmt.Errorf("%w: %v", ErrOutput, err)
        }
    }
    if err := tw.Flush(); err != nil {
        return fmt.Errorf("%w: %v", ErrOutput, err)
    }
    return nil
}

func WriteCSV(w io.Writer, headers []string, rows [][]string) error {
    cw := csv.NewWriter(w)
    if len(headers) > 0 {
        if err := cw.Write(headers); err != nil {
            return fmt.Errorf("%w: %v", ErrOutput, err)
        }
    }
    for _, row := range rows {
        if err := cw.Write(row); err != nil {
            return fmt.Errorf("%w: %v", ErrOutput, err)
        }
    }
    cw.Flush()
    if err := cw.Error(); err != nil {
        return fmt.Errorf("%w: %v", ErrOutput, err)
    }
    return nil
}

func WriteCSVRows(w io.Writer, rows [][]string) error {
    return WriteCSV(w, nil, rows)
}
''')

app = Path("internal/cli/app.go")
text = app.read_text()
text = text.replace(
    '"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"',
    '"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"\n\t"github.com/DishanRajapaksha/modbus-cli/internal/output"',
    1,
)
old = '''\tcase errors.Is(err, context.DeadlineExceeded), strings.Contains(strings.ToLower(err.Error()), "timeout"):
\t\treturn exitTimeout
\tcase errors.Is(err, modbusclient.ErrConnection):'''
new = '''\tcase errors.Is(err, context.DeadlineExceeded), strings.Contains(strings.ToLower(err.Error()), "timeout"):
\t\treturn exitTimeout
\tcase errors.Is(err, output.ErrOutput):
\t\treturn exitOutputError
\tcase errors.Is(err, modbusclient.ErrConnection):'''
if text.count(old) != 1:
    raise SystemExit("exit-code mapping marker not found")
app.write_text(text.replace(old, new, 1))

render = Path("internal/cli/render.go")
text = render.read_text()
text = text.replace(
    'return output.WriteCSV(a.out, []string{"Point", "Kind", "Address", "Value", "Unit", "Raw"}, readRows(result))',
    'return output.WriteCSV(a.out, readHeaders(), readRows(result))',
    1,
)
text = text.replace(
    'return output.WriteTable(a.out, []string{"Point", "Kind", "Address", "Value", "Unit", "Raw"}, readRows(result))',
    'return output.WriteTable(a.out, readHeaders(), readRows(result))',
    1,
)
text = text.replace(
    'return output.WriteCSV(a.out, []string{"Point", "Kind", "Address", "Value", "Unit", "Raw"}, readRows(result))',
    'return output.WriteCSVRows(a.out, readRows(result))',
    1,
)
marker = 'func readRows(result modbusclient.ReadResult) [][]string {'
helper = '''func readHeaders() []string {
\treturn []string{"Point", "Kind", "Address", "Value", "Unit", "Raw"}
}

'''
if marker not in text:
    raise SystemExit("readRows marker not found")
text = text.replace(marker, helper + marker, 1)
text = text.replace('return err', 'return fmt.Errorf("%w: %v", output.ErrOutput, err)')
render.write_text(text)

commands = Path("internal/cli/commands.go")
text = commands.read_text()
marker = '''\tticker := time.NewTicker(*interval)
\tdefer ticker.Stop()
\tfor {'''
replacement = '''\tticker := time.NewTicker(*interval)
\tdefer ticker.Stop()
\tif output.NormaliseFormat(format) == output.FormatCSV {
\t\tif err := output.WriteCSV(a.out, readHeaders(), nil); err != nil {
\t\t\treturn err
\t\t}
\t}
\tfor {'''
if text.count(marker) != 2:
    raise SystemExit(f"watch loop marker count: {text.count(marker)}")
commands.write_text(text.replace(marker, replacement, 2))

Path("internal/cli/output_contracts_test.go").write_text(r'''package cli

import (
    "bytes"
    "errors"
    "strings"
    "testing"

    "github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
    "github.com/DishanRajapaksha/modbus-cli/internal/output"
)

func TestOutputErrorUsesSharedExitCode(t *testing.T) {
    if got := mapExitCode(output.ErrOutput); got != exitOutputError {
        t.Fatalf("output exit code = %d, want %d", got, exitOutputError)
    }
    if !errors.Is(output.ErrOutput, output.ErrOutput) {
        t.Fatal("output sentinel is not comparable")
    }
}

func TestCSVStreamWritesHeaderOnce(t *testing.T) {
    var out bytes.Buffer
    app := NewAppWithFactory(&out, &out, fakeFactory{})
    if err := output.WriteCSV(&out, readHeaders(), nil); err != nil {
        t.Fatal(err)
    }
    result := modbusclient.ReadResult{Kind: "holding-registers", Values: []modbusclient.Value{{Address: 1, Value: 42}}}
    if err := app.renderWatch(output.FormatCSV, result); err != nil {
        t.Fatal(err)
    }
    if err := app.renderWatch(output.FormatCSV, result); err != nil {
        t.Fatal(err)
    }
    if count := strings.Count(out.String(), "Point,Kind,Address,Value,Unit,Raw"); count != 1 {
        t.Fatalf("CSV header count = %d, output: %q", count, out.String())
    }
}
''')
