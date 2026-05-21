package cli

import (
	"bytes"
	"context"
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

func TestPreCommandAddressBecomesConnectAddressForRead(t *testing.T) {
	var out, err bytes.Buffer
	code := NewAppWithFactory(&out, &err, fakeFactory{}).Run([]string{"--address", "192.0.2.10:502", "read", "holding-registers", "--address", "0", "--quantity", "1", "--type", "uint16"})
	if code != exitSuccess {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "4660") {
		t.Fatalf("unexpected output: %s", out.String())
	}
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
