from pathlib import Path

app = Path("internal/cli/app.go")
text = app.read_text()
text = text.replace('import (\n\t"errors"', 'import (\n\t"context"\n\t"errors"', 1)
text = text.replace(
    '\texitRequestError  = 4\n\texitOutputError   = 9',
    '\texitRequestError   = 4\n\texitWriteRejected = 7\n\texitTimeout       = 8\n\texitOutputError   = 9',
    1,
)
text = text.replace(
    '\tcase errors.Is(err, modbusclient.ErrConnection):\n\t\treturn exitConnection',
    '\tcase errors.Is(err, context.DeadlineExceeded), strings.Contains(strings.ToLower(err.Error()), "timeout"):\n\t\treturn exitTimeout\n\tcase errors.Is(err, modbusclient.ErrConnection):\n\t\treturn exitConnection',
    1,
)
text = text.replace('"--transport", "--address", "--connect-address",', '"--transport", "--connect-address",', 1)
text = text.replace('  modbus-cli test-connection --transport tcp --address 127.0.0.1:502', '  modbus-cli test-connection --transport tcp --connect-address 127.0.0.1:502', 1)
text = text.replace(
    '  --address     TCP host:port or serial device path on diagnostics; coil/register address on read/write/watch\n  --connect-address\n                TCP host:port or serial device path on read/write/watch',
    '  --connect-address\n                TCP host:port or serial device path for all commands\n  --address     Coil or register address on read/write/watch commands',
    1,
)
text = text.replace('  --format      table, text, json, jsonl, or csv', '  --format      snapshots: table, text, json, csv; streams: text, jsonl, csv', 1)
app.write_text(text)

flags = Path("internal/cli/flags.go")
text = flags.read_text()
old = '''\tif includeConnectionAddress {\n\t\tfs.StringVar(&opts.address, "address", opts.address, "TCP host:port or serial device path")\n\t} else {\n\t\tfs.StringVar(&opts.address, "connect-address", opts.address, "TCP host:port or serial device path")\n\t}'''
new = '\tfs.StringVar(&opts.address, "connect-address", opts.address, "TCP host:port or serial device path")'
if old not in text:
    raise SystemExit("connection address flag block not found")
text = text.replace(old, new, 1)
text = text.replace('if visited["address"] || visited["connect-address"] {', 'if visited["connect-address"] {', 1)
text = text.replace('case output.FormatTable, output.FormatText, output.FormatJSON, output.FormatJSONL, output.FormatCSV:', 'case output.FormatTable, output.FormatText, output.FormatJSON, output.FormatCSV:', 1)
flags.write_text(text)

commands = Path("internal/cli/commands.go")
text = commands.read_text().replace('output format: table, text, json, jsonl, or csv', 'output format: table, text, json, or csv')
commands.write_text(text)

readme = Path("README.md")
text = readme.read_text()
text = text.replace('test-connection --transport tcp --address 127.0.0.1:502', 'test-connection --transport tcp --connect-address 127.0.0.1:502')
text = text.replace('modbus-cli --address 192.0.2.10:502 read coils', 'modbus-cli --connect-address 192.0.2.10:502 read coils')
text = text.replace('table\ntext\njson\njsonl\ncsv', 'table\ntext\njson\ncsv', 1)
text = text.replace('4  Modbus request error\n9  output or formatting error', '4  protocol or request error\n7  write or control rejected (reserved)\n8  operation timeout\n9  output or formatting error', 1)
readme.write_text(text)

Path("internal/cli/contracts_test.go").write_text(r'''package cli

import (
    "context"
    "reflect"
    "testing"
)

func TestSnapshotFormatsExcludeJSONL(t *testing.T) {
    if err := validateSnapshotFormat("jsonl"); err == nil {
        t.Fatal("snapshot commands must reject jsonl")
    }
    for _, format := range []string{"table", "text", "json", "csv"} {
        if err := validateSnapshotFormat(format); err != nil {
            t.Fatalf("snapshot format %q rejected: %v", format, err)
        }
    }
}

func TestStreamFormatsUseLineDelimitedOutput(t *testing.T) {
    for _, format := range []string{"text", "jsonl", "csv"} {
        if err := validateStreamFormat(format); err != nil {
            t.Fatalf("stream format %q rejected: %v", format, err)
        }
    }
    for _, format := range []string{"table", "json"} {
        if err := validateStreamFormat(format); err == nil {
            t.Fatalf("stream format %q must be rejected", format)
        }
    }
}

func TestConnectionAddressHasUnambiguousGlobalFlag(t *testing.T) {
    got, err := normaliseGlobalFlags([]string{"--connect-address", "192.0.2.10:502", "read", "coils"})
    if err != nil {
        t.Fatal(err)
    }
    want := []string{"read", "coils", "--connect-address", "192.0.2.10:502"}
    if !reflect.DeepEqual(got, want) {
        t.Fatalf("normalised arguments = %#v, want %#v", got, want)
    }
    if _, err := normaliseGlobalFlags([]string{"--address", "192.0.2.10:502", "read", "coils"}); err == nil {
        t.Fatal("ambiguous global --address must be rejected")
    }
}

func TestTimeoutExitCodeIsSharedContractValue(t *testing.T) {
    if got := mapExitCode(context.DeadlineExceeded); got != exitTimeout {
        t.Fatalf("timeout exit code = %d, want %d", got, exitTimeout)
    }
    if exitTimeout != 8 || exitWriteRejected != 7 || exitOutputError != 9 {
        t.Fatalf("shared exit-code contract changed: rejected=%d timeout=%d output=%d", exitWriteRejected, exitTimeout, exitOutputError)
    }
}
''')
