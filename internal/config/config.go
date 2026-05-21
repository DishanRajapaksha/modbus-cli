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
	Points     []PointConfig    `yaml:"points,omitempty"`
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

type PointConfig struct {
	Name      string   `yaml:"name"`
	Kind      string   `yaml:"kind"`
	Address   uint16   `yaml:"address"`
	Quantity  uint16   `yaml:"quantity,omitempty"`
	Type      string   `yaml:"type,omitempty"`
	ByteOrder string   `yaml:"byte_order,omitempty"`
	WordOrder string   `yaml:"word_order,omitempty"`
	Unit      string   `yaml:"unit,omitempty"`
	Scale     *float64 `yaml:"scale,omitempty"`
	Offset    *float64 `yaml:"offset,omitempty"`
	Writable  bool     `yaml:"writable,omitempty"`
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
	if err := validateDeclaredPointLists(cfg); err != nil {
		return FileConfig{}, err
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
	if err := validatePoints(cfg.Points); err != nil {
		return err
	}
	return nil
}

func StarterConfigYAML() ([]byte, error) {
	cfg := DefaultConfig()
	cfg.Points = []PointConfig{
		{Name: "breaker_closed", Kind: "coil", Address: 0, Quantity: 1, Writable: true},
		{Name: "active_power", Kind: "holding-register", Address: 0, Quantity: 2, Type: "float32", Unit: "kW"},
	}
	return yaml.Marshal(FileConfig{
		Config:         cfg,
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
	if len(override.Points) > 0 {
		base.Points = mergePoints(base.Points, override.Points)
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

func mergePoints(base []PointConfig, override []PointConfig) []PointConfig {
	out := append([]PointConfig(nil), base...)
	index := map[string]int{}
	for i, point := range out {
		index[point.Name] = i
	}
	for _, point := range override {
		if i, ok := index[point.Name]; ok && point.Name != "" {
			out[i] = point
			continue
		}
		index[point.Name] = len(out)
		out = append(out, point)
	}
	return out
}

func validatePoints(points []PointConfig) error {
	seen := map[string]struct{}{}
	for _, point := range points {
		name := strings.TrimSpace(point.Name)
		if name == "" {
			return fmt.Errorf("%w: point name is required", ErrConfig)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("%w: duplicate point name %q", ErrConfig, name)
		}
		seen[name] = struct{}{}
		switch strings.ToLower(point.Kind) {
		case "coil", "discrete-input", "holding-register", "input-register":
		default:
			return fmt.Errorf("%w: point %q kind must be coil, discrete-input, holding-register, or input-register", ErrConfig, name)
		}
		if point.Quantity == 0 {
			return fmt.Errorf("%w: point %q quantity must be greater than zero", ErrConfig, name)
		}
		switch strings.ToLower(point.Kind) {
		case "coil", "discrete-input":
			if point.Quantity > 2000 {
				return fmt.Errorf("%w: point %q quantity must be 2000 or less", ErrConfig, name)
			}
		case "holding-register", "input-register":
			if point.Quantity > 125 {
				return fmt.Errorf("%w: point %q quantity must be 125 or less", ErrConfig, name)
			}
		}
		switch strings.ToLower(point.Type) {
		case "", "raw", "uint16", "int16", "uint32", "int32", "uint64", "int64", "float32", "float64":
		default:
			return fmt.Errorf("%w: point %q has unsupported type %q", ErrConfig, name, point.Type)
		}
		switch strings.ToLower(point.ByteOrder) {
		case "", "big", "little":
		default:
			return fmt.Errorf("%w: point %q byte_order must be big or little", ErrConfig, name)
		}
		switch strings.ToLower(point.WordOrder) {
		case "", "high-low", "low-high":
		default:
			return fmt.Errorf("%w: point %q word_order must be high-low or low-high", ErrConfig, name)
		}
		if point.Scale != nil && *point.Scale == 0 {
			return fmt.Errorf("%w: point %q scale must not be zero", ErrConfig, name)
		}
	}
	return nil
}

func validateDeclaredPointLists(file FileConfig) error {
	if err := validatePoints(file.Points); err != nil {
		return err
	}
	for name, profile := range file.Profiles {
		if err := validatePoints(profile.Points); err != nil {
			return fmt.Errorf("%w: profile %q: %v", ErrConfig, name, err)
		}
	}
	return nil
}
