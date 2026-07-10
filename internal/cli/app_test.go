package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
)

func TestHelpExitsSuccess(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"help"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "modbus-cli") {
		t.Fatalf("unexpected help: %s", out.String())
	}
}

func TestCompletionsHelpExitsSuccess(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"completions", "--help"})
	if code != exitSuccess {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(err.String(), "Usage of completions:") {
		t.Fatalf("unexpected stderr: %s", err.String())
	}
}

func TestWatchRejectsJSONFormat(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"watch", "coils", "--format", "json"})
	if code != exitConfigError {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(err.String(), "use --format jsonl") {
		t.Fatalf("unexpected stderr: %s", err.String())
	}
}

func TestWriteDryRunDoesNotCreateClient(t *testing.T) {
	var out, err bytes.Buffer
	factory := fakeFactory{failOnNew: true}
	code := NewAppWithFactory(&out, &err, factory).Run([]string{"write", "register", "--address", "10", "--type", "uint16", "--value", "42"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "false") || !strings.Contains(out.String(), "true") {
		t.Fatalf("expected dry-run table output, got: %s", out.String())
	}
}

func TestReadUsesFactory(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"read", "holding-registers", "--address", "0", "--quantity", "1", "--type", "uint16"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "4660") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestPreCommandConnectAddressForRead(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"--connect-address", "192.0.2.10:502", "read", "holding-registers", "--address", "0", "--quantity", "1", "--type", "uint16"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "4660") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestPointsListsConfiguredPoints(t *testing.T) {
	path := writePointConfig(t)
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"points", "--config", path})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "active_power") || !strings.Contains(out.String(), "kW") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestReadPointUsesRawReadPath(t *testing.T) {
	path := writePointConfig(t)
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"read-point", "active_power", "--config", path})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "active_power") || !strings.Contains(out.String(), "466") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestWritePointDryRunDoesNotCreateClient(t *testing.T) {
	path := writePointConfig(t)
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{failOnNew: true}).Run([]string{"write-point", "breaker_closed", "--config", path, "--value", "on"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "breaker_closed") || !strings.Contains(out.String(), "true") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestWatchPointRejectsJSONFormat(t *testing.T) {
	path := writePointConfig(t)
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"watch-point", "active_power", "--config", path, "--format", "json"})
	if code != exitConfigError {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(err.String(), "use --format jsonl") {
		t.Fatalf("unexpected stderr: %s", err.String())
	}
}

func TestSunSpecHelpExitsSuccess(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"sunspec", "--help"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(err.String(), "Usage of sunspec:") {
		t.Fatalf("unexpected stderr: %s", err.String())
	}
}

func TestSunSpecScanUsesFactory(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeSunSpecFactory{}).Run([]string{"sunspec", "scan"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "Model") || !strings.Contains(out.String(), "1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func writePointConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`points:
  - name: active_power
    kind: holding-register
    address: 0
    quantity: 1
    type: uint16
    unit: kW
    scale: 0.1
  - name: breaker_closed
    kind: coil
    address: 0
    quantity: 1
    writable: true
`), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

type fakeFactory struct {
	failOnNew bool
}

func (f fakeFactory) New(config.Config, modbusclient.Options) (modbusclient.Client, error) {
	if f.failOnNew {
		return nil, modbusclient.ErrConnection
	}
	return fakeClient{}, nil
}

type fakeClient struct{}

func (fakeClient) Connect(context.Context) error { return nil }
func (fakeClient) Close() error                  { return nil }

func (fakeClient) ReadCoils(context.Context, uint16, uint16) ([]byte, error) {
	return []byte{0x01}, nil
}

func (fakeClient) ReadDiscreteInputs(context.Context, uint16, uint16) ([]byte, error) {
	return []byte{0x01}, nil
}

func (fakeClient) ReadHoldingRegisters(context.Context, uint16, uint16) ([]byte, error) {
	return []byte{0x12, 0x34}, nil
}

func (fakeClient) ReadInputRegisters(context.Context, uint16, uint16) ([]byte, error) {
	return []byte{0x12, 0x34}, nil
}

func (fakeClient) WriteSingleCoil(context.Context, uint16, bool) ([]byte, error) {
	return []byte{0x00, 0x00}, nil
}

func (fakeClient) WriteMultipleCoils(context.Context, uint16, []bool) ([]byte, error) {
	return []byte{0x00, 0x00}, nil
}

func (fakeClient) WriteSingleRegister(context.Context, uint16, uint16) ([]byte, error) {
	return []byte{0x00, 0x00}, nil
}

func (fakeClient) WriteMultipleRegisters(context.Context, uint16, []uint16) ([]byte, error) {
	return []byte{0x00, 0x00}, nil
}

func (fakeClient) ReadDeviceIdentification(context.Context, modbusclient.IdentificationLevel) (map[byte][]byte, error) {
	return map[byte][]byte{0: []byte("vendor")}, nil
}

type fakeSunSpecFactory struct{}

func (fakeSunSpecFactory) New(config.Config, modbusclient.Options) (modbusclient.Client, error) {
	return fakeSunSpecClient{registers: commonModelRegisters()}, nil
}

type fakeSunSpecClient struct {
	registers map[uint16]uint16
}

func (fakeSunSpecClient) Connect(context.Context) error { return nil }
func (fakeSunSpecClient) Close() error                  { return nil }
func (fakeSunSpecClient) ReadCoils(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (fakeSunSpecClient) ReadDiscreteInputs(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) ReadHoldingRegisters(_ context.Context, address uint16, quantity uint16) ([]byte, error) {
	out := make([]byte, quantity*2)
	for i := uint16(0); i < quantity; i++ {
		value := f.registers[address+i]
		out[i*2] = byte(value >> 8)
		out[i*2+1] = byte(value)
	}
	return out, nil
}
func (fakeSunSpecClient) ReadInputRegisters(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (fakeSunSpecClient) WriteSingleCoil(context.Context, uint16, bool) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (fakeSunSpecClient) WriteMultipleCoils(context.Context, uint16, []bool) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (fakeSunSpecClient) WriteSingleRegister(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (fakeSunSpecClient) WriteMultipleRegisters(context.Context, uint16, []uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (fakeSunSpecClient) ReadDeviceIdentification(context.Context, modbusclient.IdentificationLevel) (map[byte][]byte, error) {
	return nil, modbusclient.ErrRequest
}

func commonModelRegisters() map[uint16]uint16 {
	return map[uint16]uint16{
		40000: 0x5375,
		40001: 0x6e53,
		40002: 1,
		40003: 66,
		40070: 0xffff,
	}
}
