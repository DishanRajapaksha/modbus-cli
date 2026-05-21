package modbusclient

import "testing"

func TestDecodeCoils(t *testing.T) {
	got := DecodeCoils([]byte{0b00000101}, 4)
	want := []bool{true, false, true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d got %v want %v", i, got[i], want[i])
		}
	}
}

func TestDecodeRegistersFloat32LowHigh(t *testing.T) {
	values, err := DecodeRegisters([]byte{0x00, 0x00, 0x3f, 0x80}, TypeFloat32, "big", "low-high")
	if err != nil {
		t.Fatal(err)
	}
	if values[0].Value != float32(1) {
		t.Fatalf("got %v", values[0].Value)
	}
}

func TestEncodeRegisterValues(t *testing.T) {
	registers, display, err := EncodeRegisterValues([]string{"42"}, TypeUint16, "big", "high-low")
	if err != nil {
		t.Fatal(err)
	}
	if len(registers) != 1 || registers[0] != 42 {
		t.Fatalf("registers=%v", registers)
	}
	if display[0] != uint16(42) {
		t.Fatalf("display=%v", display)
	}
}

func TestParseCoilValues(t *testing.T) {
	values, err := ParseCoilValues("on,off,true,0")
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, true, false}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("index %d got %v want %v", i, values[i], want[i])
		}
	}
}
