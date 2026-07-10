package cli

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
