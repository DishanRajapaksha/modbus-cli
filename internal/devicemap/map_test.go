package devicemap

import (
	"testing"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
)

func TestApplyReadScaleOffsetAndUnit(t *testing.T) {
	scale := 0.1
	offset := 5.0
	point := config.PointConfig{Name: "power", Unit: "kW", Scale: &scale, Offset: &offset}
	result := modbusclient.ReadResult{Values: []modbusclient.Value{{Value: uint16(100)}}}
	got := ApplyRead(point, result)
	if got.Point != "power" || got.Unit != "kW" {
		t.Fatalf("metadata not applied: %+v", got)
	}
	if got.Values[0].Value != float64(15) {
		t.Fatalf("value=%v", got.Values[0].Value)
	}
	if got.Values[0].Unit != "kW" {
		t.Fatalf("unit=%q", got.Values[0].Unit)
	}
}

func TestPrepareWriteValueReversesScaleOffset(t *testing.T) {
	scale := 0.1
	offset := 5.0
	point := config.PointConfig{Name: "power", Kind: "holding-register", Type: modbusclient.TypeUint16, Scale: &scale, Offset: &offset}
	got, err := PrepareWriteValue(point, "15")
	if err != nil {
		t.Fatal(err)
	}
	if got != "100" {
		t.Fatalf("got %q", got)
	}
}

func TestPrepareWriteValueLeavesRawHexAlone(t *testing.T) {
	point := config.PointConfig{Name: "raw", Kind: "holding-register", Type: modbusclient.TypeRaw}
	got, err := PrepareWriteValue(point, "3f800000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "3f800000" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteKindRejectsReadOnlyPoint(t *testing.T) {
	_, err := WriteKind(config.PointConfig{Name: "x", Kind: "input-register", Quantity: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}
