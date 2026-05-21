package devicemap

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
)

func Find(points []config.PointConfig, name string) (config.PointConfig, error) {
	for _, point := range points {
		if point.Name == name {
			return withDefaults(point), nil
		}
	}
	return config.PointConfig{}, fmt.Errorf("%w: point %q not found", modbusclient.ErrValidation, name)
}

func List(points []config.PointConfig) []config.PointConfig {
	out := make([]config.PointConfig, 0, len(points))
	for _, point := range points {
		out = append(out, withDefaults(point))
	}
	return out
}

func ReadKind(point config.PointConfig) (string, error) {
	switch strings.ToLower(point.Kind) {
	case "coil":
		return "coils", nil
	case "discrete-input":
		return "discrete-inputs", nil
	case "holding-register":
		return "holding-registers", nil
	case "input-register":
		return "input-registers", nil
	default:
		return "", fmt.Errorf("%w: unsupported point kind %q", modbusclient.ErrValidation, point.Kind)
	}
}

func WriteKind(point config.PointConfig) (string, error) {
	if !point.Writable {
		return "", fmt.Errorf("%w: point %q is not writable", modbusclient.ErrValidation, point.Name)
	}
	switch strings.ToLower(point.Kind) {
	case "coil":
		if point.Quantity == 1 {
			return "coil", nil
		}
		return "coils", nil
	case "holding-register":
		if point.Quantity == 1 {
			return "register", nil
		}
		return "registers", nil
	default:
		return "", fmt.Errorf("%w: point %q kind %q is read-only", modbusclient.ErrValidation, point.Name, point.Kind)
	}
}

func ApplyRead(point config.PointConfig, result modbusclient.ReadResult) modbusclient.ReadResult {
	point = withDefaults(point)
	result.Point = point.Name
	result.Unit = point.Unit
	for i := range result.Values {
		result.Values[i].Point = point.Name
		result.Values[i].Unit = point.Unit
		if value, ok := numeric(result.Values[i].Value); ok {
			result.Values[i].Value = value*scale(point) + offset(point)
		}
	}
	return result
}

func PrepareWriteValue(point config.PointConfig, value string) (string, error) {
	point = withDefaults(point)
	if strings.EqualFold(point.Kind, "coil") {
		return value, nil
	}
	if strings.EqualFold(point.Type, modbusclient.TypeRaw) {
		return value, nil
	}
	parts := splitValues(value)
	if len(parts) == 0 {
		return "", fmt.Errorf("%w: --value is required", modbusclient.ErrValidation)
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return "", fmt.Errorf("%w: invalid value %q for point %q", modbusclient.ErrValidation, part, point.Name)
		}
		raw := (parsed - offset(point)) / scale(point)
		out = append(out, strconv.FormatFloat(raw, 'f', -1, 64))
	}
	return strings.Join(out, ","), nil
}

func withDefaults(point config.PointConfig) config.PointConfig {
	if point.Quantity == 0 {
		point.Quantity = 1
	}
	if point.Type == "" {
		if strings.EqualFold(point.Kind, "coil") || strings.EqualFold(point.Kind, "discrete-input") {
			point.Type = "bool"
		} else {
			point.Type = modbusclient.TypeRaw
		}
	}
	if point.ByteOrder == "" {
		point.ByteOrder = "big"
	}
	if point.WordOrder == "" {
		point.WordOrder = "high-low"
	}
	return point
}

func scale(point config.PointConfig) float64 {
	if point.Scale == nil {
		return 1
	}
	return *point.Scale
}

func offset(point config.PointConfig) float64 {
	if point.Offset == nil {
		return 0
	}
	return *point.Offset
}

func numeric(value any) (float64, bool) {
	switch v := value.(type) {
	case uint16:
		return float64(v), true
	case int16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
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
