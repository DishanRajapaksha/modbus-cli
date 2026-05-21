package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath = "config.yaml"
	DefaultAddress    = "127.0.0.1:502"
)

var ErrConfig = errors.New("config error")

type Config struct {
	Connection ConnectionConfig `yaml:"connection"`
	RTU        RTUConfig        `yaml:"rtu"`
	Output     OutputConfig     `yaml:"output"`
}

type ConnectionConfig struct {
	Transport   string        `yaml:"transport"`
	Address     string        `yaml:"address"`
	UnitID      byte          `yaml:"unit_id"`
	Timeout     time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

type RTUConfig struct {
	BaudRate int    `yaml:"baud_rate"`
	DataBits int    `yaml:"data_bits"`
	StopBits int    `yaml:"stop_bits"`
	Parity   string `yaml:"parity"`
}

type OutputConfig struct {
	Format string `yaml:"format"`
}

type FileConfig struct {
	Config         `yaml:",inline"`
	DefaultProfile string            `yaml:"default_profile"`
	Profiles       map[string]Config `yaml:"profiles"`
}

type Overrides struct {
	Transport string
	Address   string
	UnitID    *uint
	Timeout   *time.Duration
	Format    string
	BaudRate  *int
	DataBits  *int
	StopBits  *int
	Parity    string
	Verbose   bool
	Debug     bool
}

func DefaultConfig() Config {
	return Config{
		Connection: ConnectionConfig{
			Transport:   "tcp",
			Address:     DefaultAddress,
			UnitID:      1,
			Timeout:     5 * time.Second,
			IdleTimeout: time.Minute,
		},
		RTU: RTUConfig{
			BaudRate: 19200,
			DataBits: 8,
			StopBits: 1,
			Parity:   "E",
		},
		Output: OutputConfig{
			Format: "table",
		},
	}
}

func LoadForProfile(path string, profile string, overrides Overrides) (Config, error) {
	cfg := DefaultConfig()
	if path != "" {
		file, err := LoadFile(path)
		if err != nil {
			return cfg, err
		}
		cfg = mergeConfig(cfg, file.Config)
		selected := profile
		if selected == "" {
			selected = file.DefaultProfile
		}
		if selected != "" {
			profileCfg, ok := file.Profiles[selected]
			if !ok {
				return cfg, fmt.Errorf("%w: profile %q not found", ErrConfig, selected)
			}
			cfg = mergeConfig(cfg, profileCfg)
		}
	}
	applyOverrides(&cfg, overrides)
	if err := Validate(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func LoadFile(path string) (FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && path == DefaultConfigPath {
			return FileConfig{Config: DefaultConfig()}, nil
		}
		return FileConfig{}, fmt.Errorf("%w: read config %q: %v", ErrConfig, path, err)
	}
	cfg := FileConfig{Config: DefaultConfig()}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("%w: parse config %q: %v", ErrConfig, path, err)
	}
	return cfg, nil
}

func Validate(cfg Config) error {
	switch strings.ToLower(cfg.Connection.Transport) {
	case "tcp", "rtu":
	default:
		return fmt.Errorf("%w: connection.transport must be tcp or rtu", ErrConfig)
	}
	if strings.TrimSpace(cfg.Connection.Address) == "" {
		return fmt.Errorf("%w: connection.address is required", ErrConfig)
	}
	if cfg.Connection.UnitID == 0 {
		return fmt.Errorf("%w: connection.unit_id must be between 1 and 247", ErrConfig)
	}
	if cfg.Connection.UnitID > 247 {
		return fmt.Errorf("%w: connection.unit_id must be between 1 and 247", ErrConfig)
	}
	if cfg.Connection.Timeout <= 0 {
		return fmt.Errorf("%w: connection.timeout must be greater than zero", ErrConfig)
	}
	if cfg.Connection.IdleTimeout < 0 {
		return fmt.Errorf("%w: connection.idle_timeout must be zero or greater", ErrConfig)
	}
	if cfg.RTU.BaudRate <= 0 {
		return fmt.Errorf("%w: rtu.baud_rate must be greater than zero", ErrConfig)
	}
	if cfg.RTU.DataBits < 5 || cfg.RTU.DataBits > 8 {
		return fmt.Errorf("%w: rtu.data_bits must be between 5 and 8", ErrConfig)
	}
	if cfg.RTU.StopBits != 1 && cfg.RTU.StopBits != 2 {
		return fmt.Errorf("%w: rtu.stop_bits must be 1 or 2", ErrConfig)
	}
	switch strings.ToUpper(cfg.RTU.Parity) {
	case "N", "E", "O":
	default:
		return fmt.Errorf("%w: rtu.parity must be N, E, or O", ErrConfig)
	}
	return nil
}

func StarterConfigYAML() ([]byte, error) {
	return yaml.Marshal(FileConfig{
		Config:         DefaultConfig(),
		DefaultProfile: "local",
		Profiles: map[string]Config{
			"local": {
				Connection: ConnectionConfig{Transport: "tcp", Address: DefaultAddress, UnitID: 1},
			},
			"serial": {
				Connection: ConnectionConfig{Transport: "rtu", Address: "/dev/ttyUSB0", UnitID: 1},
				RTU:        DefaultConfig().RTU,
			},
		},
	})
}

func mergeConfig(base Config, override Config) Config {
	if override.Connection.Transport != "" {
		base.Connection.Transport = override.Connection.Transport
	}
	if override.Connection.Address != "" {
		base.Connection.Address = override.Connection.Address
	}
	if override.Connection.UnitID != 0 {
		base.Connection.UnitID = override.Connection.UnitID
	}
	if override.Connection.Timeout != 0 {
		base.Connection.Timeout = override.Connection.Timeout
	}
	if override.Connection.IdleTimeout != 0 {
		base.Connection.IdleTimeout = override.Connection.IdleTimeout
	}
	if override.RTU.BaudRate != 0 {
		base.RTU.BaudRate = override.RTU.BaudRate
	}
	if override.RTU.DataBits != 0 {
		base.RTU.DataBits = override.RTU.DataBits
	}
	if override.RTU.StopBits != 0 {
		base.RTU.StopBits = override.RTU.StopBits
	}
	if override.RTU.Parity != "" {
		base.RTU.Parity = override.RTU.Parity
	}
	if override.Output.Format != "" {
		base.Output.Format = override.Output.Format
	}
	return base
}

func applyOverrides(cfg *Config, overrides Overrides) {
	if overrides.Transport != "" {
		cfg.Connection.Transport = overrides.Transport
	}
	if overrides.Address != "" {
		cfg.Connection.Address = overrides.Address
	}
	if overrides.UnitID != nil {
		cfg.Connection.UnitID = byte(*overrides.UnitID)
	}
	if overrides.Timeout != nil {
		cfg.Connection.Timeout = *overrides.Timeout
	}
	if overrides.Format != "" {
		cfg.Output.Format = overrides.Format
	}
	if overrides.BaudRate != nil {
		cfg.RTU.BaudRate = *overrides.BaudRate
	}
	if overrides.DataBits != nil {
		cfg.RTU.DataBits = *overrides.DataBits
	}
	if overrides.StopBits != nil {
		cfg.RTU.StopBits = *overrides.StopBits
	}
	if overrides.Parity != "" {
		cfg.RTU.Parity = overrides.Parity
	}
}
