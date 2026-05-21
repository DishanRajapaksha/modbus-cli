package sunspec

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
)

func TestScanFindsSunSpecModels(t *testing.T) {
	client := fakeSunSpecClient{registers: commonModelRegisters()}
	result, err := NewScanner(client).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Devices) != 1 {
		t.Fatalf("devices=%+v", result.Devices)
	}
	if len(result.Devices[0].Models) != 1 {
		t.Fatalf("models=%+v", result.Devices[0].Models)
	}
	if result.Devices[0].Models[0].ID != 1 {
		t.Fatalf("model=%+v", result.Devices[0].Models[0])
	}
}

func commonModelRegisters() map[uint16]uint16 {
	registers := map[uint16]uint16{}
	write32(registers, 40000, 0x53756e53)
	registers[40002] = 1
	registers[40003] = 66
	registers[40070] = 0xffff
	registers[40071] = 0
	return registers
}

func write32(registers map[uint16]uint16, address uint16, value uint32) {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], value)
	registers[address] = binary.BigEndian.Uint16(data[:2])
	registers[address+1] = binary.BigEndian.Uint16(data[2:])
}

type fakeSunSpecClient struct {
	registers map[uint16]uint16
}

func (f fakeSunSpecClient) Connect(context.Context) error { return nil }
func (f fakeSunSpecClient) Close() error                  { return nil }
func (f fakeSunSpecClient) ReadCoils(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) ReadDiscreteInputs(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) ReadHoldingRegisters(_ context.Context, address uint16, quantity uint16) ([]byte, error) {
	out := make([]byte, quantity*2)
	for i := uint16(0); i < quantity; i++ {
		binary.BigEndian.PutUint16(out[i*2:], f.registers[address+i])
	}
	return out, nil
}
func (f fakeSunSpecClient) ReadInputRegisters(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) WriteSingleCoil(context.Context, uint16, bool) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) WriteMultipleCoils(context.Context, uint16, []bool) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) WriteSingleRegister(context.Context, uint16, uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) WriteMultipleRegisters(context.Context, uint16, []uint16) ([]byte, error) {
	return nil, modbusclient.ErrRequest
}
func (f fakeSunSpecClient) ReadDeviceIdentification(context.Context, modbusclient.IdentificationLevel) (map[byte][]byte, error) {
	return nil, modbusclient.ErrRequest
}
