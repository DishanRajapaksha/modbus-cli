package modbusclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	gridx "github.com/grid-x/modbus"
)

type GridXFactory struct{}

func (GridXFactory) New(cfg config.Config, opts Options) (Client, error) {
	var handler gridx.ClientHandler
	switch strings.ToLower(cfg.Connection.Transport) {
	case "tcp":
		h := gridx.NewTCPClientHandler(cfg.Connection.Address)
		h.SlaveID = cfg.Connection.UnitID
		h.Timeout = cfg.Connection.Timeout
		h.IdleTimeout = cfg.Connection.IdleTimeout
		if opts.Debug {
			h.Logger = log.New(os.Stderr, "modbus-debug: ", 0)
		}
		handler = h
	case "rtu":
		h := gridx.NewRTUClientHandler(cfg.Connection.Address)
		h.SlaveID = cfg.Connection.UnitID
		h.Timeout = cfg.Connection.Timeout
		h.IdleTimeout = cfg.Connection.IdleTimeout
		h.BaudRate = cfg.RTU.BaudRate
		h.DataBits = cfg.RTU.DataBits
		h.StopBits = cfg.RTU.StopBits
		h.Parity = strings.ToUpper(cfg.RTU.Parity)
		if opts.Debug {
			h.Logger = log.New(os.Stderr, "modbus-debug: ", 0)
		}
		handler = h
	default:
		return nil, fmt.Errorf("%w: unsupported transport %q", ErrValidation, cfg.Connection.Transport)
	}
	return &gridXClient{handler: handler, client: gridx.NewClient(handler)}, nil
}

type gridXClient struct {
	handler gridx.ClientHandler
	client  gridx.Client
}

func (c *gridXClient) Connect(ctx context.Context) error {
	if err := c.handler.Connect(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrConnection, err)
	}
	return nil
}

func (c *gridXClient) Close() error {
	return c.handler.Close()
}

func (c *gridXClient) ReadCoils(ctx context.Context, address, quantity uint16) ([]byte, error) {
	return wrapRequest(c.client.ReadCoils(ctx, address, quantity))
}

func (c *gridXClient) ReadDiscreteInputs(ctx context.Context, address, quantity uint16) ([]byte, error) {
	return wrapRequest(c.client.ReadDiscreteInputs(ctx, address, quantity))
}

func (c *gridXClient) ReadHoldingRegisters(ctx context.Context, address, quantity uint16) ([]byte, error) {
	return wrapRequest(c.client.ReadHoldingRegisters(ctx, address, quantity))
}

func (c *gridXClient) ReadInputRegisters(ctx context.Context, address, quantity uint16) ([]byte, error) {
	return wrapRequest(c.client.ReadInputRegisters(ctx, address, quantity))
}

func (c *gridXClient) WriteSingleCoil(ctx context.Context, address uint16, value bool) ([]byte, error) {
	raw := uint16(0x0000)
	if value {
		raw = 0xff00
	}
	return wrapRequest(c.client.WriteSingleCoil(ctx, address, raw))
}

func (c *gridXClient) WriteMultipleCoils(ctx context.Context, address uint16, values []bool) ([]byte, error) {
	packed := PackCoils(values)
	return wrapRequest(c.client.WriteMultipleCoils(ctx, address, uint16(len(values)), packed))
}

func (c *gridXClient) WriteSingleRegister(ctx context.Context, address, value uint16) ([]byte, error) {
	return wrapRequest(c.client.WriteSingleRegister(ctx, address, value))
}

func (c *gridXClient) WriteMultipleRegisters(ctx context.Context, address uint16, values []uint16) ([]byte, error) {
	return wrapRequest(c.client.WriteMultipleRegisters(ctx, address, uint16(len(values)), RegistersToBytes(values)))
}

func (c *gridXClient) ReadDeviceIdentification(ctx context.Context, level IdentificationLevel) (map[byte][]byte, error) {
	var code gridx.ReadDeviceIDCode
	switch level {
	case IdentificationRegular:
		code = gridx.ReadDeviceIDCodeRegular
	case IdentificationExtended:
		code = gridx.ReadDeviceIDCodeExtended
	default:
		code = gridx.ReadDeviceIDCodeBasic
	}
	results, err := c.client.ReadDeviceIdentification(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequest, err)
	}
	return results, nil
}

func wrapRequest(results []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequest, err)
	}
	return results, nil
}
