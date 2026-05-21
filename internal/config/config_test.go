package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadForProfileMergesDefaultsAndOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`connection:
  transport: tcp
  address: 10.0.0.1:502
  unit_id: 3
default_profile: serial
profiles:
  serial:
    connection:
      transport: rtu
      address: /dev/ttyUSB0
    rtu:
      baud_rate: 9600
`), 0o600); err != nil {
		t.Fatal(err)
	}
	timeout := 2 * time.Second
	unitID := uint(7)
	cfg, err := LoadForProfile(path, "", Overrides{UnitID: &unitID, Timeout: &timeout})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Connection.Transport != "rtu" || cfg.Connection.Address != "/dev/ttyUSB0" {
		t.Fatalf("unexpected connection: %+v", cfg.Connection)
	}
	if cfg.Connection.UnitID != 7 || cfg.Connection.Timeout != timeout {
		t.Fatalf("overrides not applied: %+v", cfg.Connection)
	}
	if cfg.RTU.BaudRate != 9600 || cfg.RTU.DataBits != 8 {
		t.Fatalf("unexpected rtu config: %+v", cfg.RTU)
	}
}

func TestValidateRejectsInvalidTransport(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Connection.Transport = "udp"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid transport error")
	}
}

func TestStarterConfigYAMLLoads(t *testing.T) {
	data, err := StarterConfigYAML()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadForProfile(path, "local", Overrides{}); err != nil {
		t.Fatal(err)
	}
}
