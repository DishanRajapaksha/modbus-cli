package modbusclient

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	TypeRaw     = "raw"
	TypeUint16  = "uint16"
	TypeInt16   = "int16"
	TypeUint32  = "uint32"
	TypeInt32   = "int32"
	TypeUint64  = "uint64"
	TypeInt64   = "int64"
	TypeFloat32 = "float32"
	TypeFloat64 = "float64"
)

func DecodeCoils(data []byte, quantity uint16) []bool {
	values := make([]bool, 0, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		values = append(values, data[byteIndex]&(1<<bitIndex) != 0)
	}
	return values
}

func PackCoils(values []bool) []byte {
	out := make([]byte, (len(values)+7)/8)
	for i, value := range values {
		if value {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out
}

func RegistersToBytes(values []uint16) []byte {
	out := make([]byte, len(values)*2)
	for i, value := range values {
		binary.BigEndian.PutUint16(out[i*2:], value)
	}
	return out
}

func BytesToRegisters(data []byte) ([]uint16, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("%w: register response has odd byte length", ErrValidation)
	}
	values := make([]uint16, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		values = append(values, binary.BigEndian.Uint16(data[i:i+2]))
	}
	return values, nil
}

func DecodeRegisters(data []byte, valueType, byteOrder, wordOrder string) ([]Value, error) {
	valueType = normaliseType(valueType)
	if valueType == TypeRaw {
		return []Value{{Address: 0, Value: strings.ToUpper(hex.EncodeToString(data)), Raw: strings.ToUpper(hex.EncodeToString(data))}}, nil
	}
	registers, err := BytesToRegisters(data)
	if err != nil {
		return nil, err
	}
	groupSize, err := registersPerValue(valueType)
	if err != nil {
		return nil, err
	}
	if len(registers)%groupSize != 0 {
		return nil, fmt.Errorf("%w: %s requires groups of %d register(s)", ErrValidation, valueType, groupSize)
	}
	values := make([]Value, 0, len(registers)/groupSize)
	for i := 0; i < len(registers); i += groupSize {
		group := registers[i : i+groupSize]
		raw := RegistersToBytes(group)
		ordered, err := orderBytes(raw, byteOrder, wordOrder)
		if err != nil {
			return nil, err
		}
		decoded, err := decodeOrderedBytes(ordered, valueType)
		if err != nil {
			return nil, err
		}
		values = append(values, Value{Address: uint16(i), Value: decoded, Raw: strings.ToUpper(hex.EncodeToString(raw))})
	}
	return values, nil
}

func EncodeRegisterValues(rawValues []string, valueType, byteOrder, wordOrder string) ([]uint16, []any, error) {
	valueType = normaliseType(valueType)
	if valueType == TypeRaw {
		compact := strings.Join(rawValues, "")
		compact = strings.TrimPrefix(strings.TrimPrefix(compact, "0x"), "0X")
		data, err := hex.DecodeString(compact)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid raw hex value: %v", ErrValidation, err)
		}
		registers, err := BytesToRegisters(data)
		if err != nil {
			return nil, nil, err
		}
		display := make([]any, 0, len(registers))
		for _, register := range registers {
			display = append(display, register)
		}
		return registers, display, nil
	}
	registers := []uint16{}
	display := []any{}
	for _, raw := range rawValues {
		data, parsed, err := encodeOne(raw, valueType)
		if err != nil {
			return nil, nil, err
		}
		ordered, err := reverseOrderBytes(data, byteOrder, wordOrder)
		if err != nil {
			return nil, nil, err
		}
		regs, err := BytesToRegisters(ordered)
		if err != nil {
			return nil, nil, err
		}
		registers = append(registers, regs...)
		display = append(display, parsed)
	}
	return registers, display, nil
}

func ParseCoilValues(value string) ([]bool, error) {
	parts := splitValues(value)
	values := make([]bool, 0, len(parts))
	for _, part := range parts {
		switch strings.ToLower(part) {
		case "1", "true", "on":
			values = append(values, true)
		case "0", "false", "off":
			values = append(values, false)
		default:
			return nil, fmt.Errorf("%w: invalid coil value %q", ErrValidation, part)
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: --value is required", ErrValidation)
	}
	return values, nil
}

func splitValues(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normaliseType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return TypeRaw
	}
	return value
}

func registersPerValue(valueType string) (int, error) {
	switch valueType {
	case TypeUint16, TypeInt16:
		return 1, nil
	case TypeUint32, TypeInt32, TypeFloat32:
		return 2, nil
	case TypeUint64, TypeInt64, TypeFloat64:
		return 4, nil
	default:
		return 0, fmt.Errorf("%w: unsupported register type %q", ErrValidation, valueType)
	}
}

func orderBytes(data []byte, byteOrder, wordOrder string) ([]byte, error) {
	out := append([]byte(nil), data...)
	switch strings.ToLower(byteOrder) {
	case "", "big":
	case "little":
		for i := 0; i < len(out); i += 2 {
			out[i], out[i+1] = out[i+1], out[i]
		}
	default:
		return nil, fmt.Errorf("%w: --byte-order must be big or little", ErrValidation)
	}
	switch strings.ToLower(wordOrder) {
	case "", "high-low":
	case "low-high":
		for i, j := 0, len(out)-2; i < j; i, j = i+2, j-2 {
			out[i], out[i+1], out[j], out[j+1] = out[j], out[j+1], out[i], out[i+1]
		}
	default:
		return nil, fmt.Errorf("%w: --word-order must be high-low or low-high", ErrValidation)
	}
	return out, nil
}

func reverseOrderBytes(data []byte, byteOrder, wordOrder string) ([]byte, error) {
	return orderBytes(data, byteOrder, wordOrder)
}

func decodeOrderedBytes(data []byte, valueType string) (any, error) {
	switch valueType {
	case TypeUint16:
		return binary.BigEndian.Uint16(data), nil
	case TypeInt16:
		return int16(binary.BigEndian.Uint16(data)), nil
	case TypeUint32:
		return binary.BigEndian.Uint32(data), nil
	case TypeInt32:
		return int32(binary.BigEndian.Uint32(data)), nil
	case TypeUint64:
		return binary.BigEndian.Uint64(data), nil
	case TypeInt64:
		return int64(binary.BigEndian.Uint64(data)), nil
	case TypeFloat32:
		return math.Float32frombits(binary.BigEndian.Uint32(data)), nil
	case TypeFloat64:
		return math.Float64frombits(binary.BigEndian.Uint64(data)), nil
	default:
		return nil, fmt.Errorf("%w: unsupported register type %q", ErrValidation, valueType)
	}
}

func encodeOne(raw string, valueType string) ([]byte, any, error) {
	switch valueType {
	case TypeUint16:
		v, err := strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid uint16 %q", ErrValidation, raw)
		}
		out := make([]byte, 2)
		binary.BigEndian.PutUint16(out, uint16(v))
		return out, uint16(v), nil
	case TypeInt16:
		v, err := strconv.ParseInt(raw, 10, 16)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid int16 %q", ErrValidation, raw)
		}
		out := make([]byte, 2)
		binary.BigEndian.PutUint16(out, uint16(int16(v)))
		return out, int16(v), nil
	case TypeUint32:
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid uint32 %q", ErrValidation, raw)
		}
		out := make([]byte, 4)
		binary.BigEndian.PutUint32(out, uint32(v))
		return out, uint32(v), nil
	case TypeInt32:
		v, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid int32 %q", ErrValidation, raw)
		}
		out := make([]byte, 4)
		binary.BigEndian.PutUint32(out, uint32(int32(v)))
		return out, int32(v), nil
	case TypeUint64:
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid uint64 %q", ErrValidation, raw)
		}
		out := make([]byte, 8)
		binary.BigEndian.PutUint64(out, v)
		return out, v, nil
	case TypeInt64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid int64 %q", ErrValidation, raw)
		}
		out := make([]byte, 8)
		binary.BigEndian.PutUint64(out, uint64(v))
		return out, v, nil
	case TypeFloat32:
		v, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid float32 %q", ErrValidation, raw)
		}
		out := make([]byte, 4)
		f := float32(v)
		binary.BigEndian.PutUint32(out, math.Float32bits(f))
		return out, f, nil
	case TypeFloat64:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid float64 %q", ErrValidation, raw)
		}
		out := make([]byte, 8)
		binary.BigEndian.PutUint64(out, math.Float64bits(v))
		return out, v, nil
	default:
		return nil, nil, fmt.Errorf("%w: unsupported register type %q", ErrValidation, valueType)
	}
}
