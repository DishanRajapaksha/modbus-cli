package modbusclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
)

var (
	ErrConnection = errors.New("connection error")
	ErrRequest    = errors.New("modbus request error")
	ErrValidation = errors.New("validation error")
)

type Factory interface {
	New(config.Config, Options) (Client, error)
}

type FactoryFunc func(config.Config, Options) (Client, error)

func (f FactoryFunc) New(cfg config.Config, opts Options) (Client, error) {
	return f(cfg, opts)
}

type Options struct {
	Verbose bool
	Debug   bool
}

type Client interface {
	Connect(context.Context) error
	Close() error
	ReadCoils(context.Context, uint16, uint16) ([]byte, error)
	ReadDiscreteInputs(context.Context, uint16, uint16) ([]byte, error)
	ReadHoldingRegisters(context.Context, uint16, uint16) ([]byte, error)
	ReadInputRegisters(context.Context, uint16, uint16) ([]byte, error)
	WriteSingleCoil(context.Context, uint16, bool) ([]byte, error)
	WriteMultipleCoils(context.Context, uint16, []bool) ([]byte, error)
	WriteSingleRegister(context.Context, uint16, uint16) ([]byte, error)
	WriteMultipleRegisters(context.Context, uint16, []uint16) ([]byte, error)
	ReadDeviceIdentification(context.Context, IdentificationLevel) (map[byte][]byte, error)
}

type IdentificationLevel string

const (
	IdentificationBasic    IdentificationLevel = "basic"
	IdentificationRegular  IdentificationLevel = "regular"
	IdentificationExtended IdentificationLevel = "extended"
)

func ParseIdentificationLevel(value string) (IdentificationLevel, error) {
	switch IdentificationLevel(value) {
	case "", IdentificationBasic:
		return IdentificationBasic, nil
	case IdentificationRegular:
		return IdentificationRegular, nil
	case IdentificationExtended:
		return IdentificationExtended, nil
	default:
		return "", fmt.Errorf("%w: --level must be basic, regular, or extended", ErrValidation)
	}
}

type ReadResult struct {
	Kind      string    `json:"kind"`
	Address   uint16    `json:"address"`
	Quantity  uint16    `json:"quantity"`
	Type      string    `json:"type,omitempty"`
	Raw       string    `json:"raw,omitempty"`
	Values    []Value   `json:"values"`
	Timestamp time.Time `json:"timestamp"`
}

type Value struct {
	Address uint16 `json:"address"`
	Value   any    `json:"value"`
	Raw     string `json:"raw,omitempty"`
}

type WriteResult struct {
	Kind      string    `json:"kind"`
	Address   uint16    `json:"address"`
	Quantity  uint16    `json:"quantity"`
	Type      string    `json:"type,omitempty"`
	Values    []any     `json:"values"`
	DryRun    bool      `json:"dry_run"`
	Sent      bool      `json:"sent"`
	Timestamp time.Time `json:"timestamp"`
}

type DiagnosticResult struct {
	Transport string `json:"transport"`
	Address   string `json:"address"`
	UnitID    byte   `json:"unit_id"`
	Result    string `json:"result"`
	Error     string `json:"error,omitempty"`
}

type IdentificationResult struct {
	Level   string            `json:"level"`
	Objects map[string]string `json:"objects"`
}
