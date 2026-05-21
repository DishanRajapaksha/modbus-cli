package cli

import (
	"flag"
	"fmt"
	"time"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/output"
)

type commonOptions struct {
	configPath string
	profile    string
	format     string
	transport  string
	address    string
	unitID     uint
	timeout    time.Duration
	baudRate   int
	dataBits   int
	stopBits   int
	parity     string
	verbose    bool
	debug      bool
}

func addCommonFlags(fs *flag.FlagSet, opts *commonOptions, defaultFormat string, formatHelp string, includeFormat bool, includeConnectionAddress bool) {
	defaults := config.DefaultConfig()
	opts.configPath = config.DefaultConfigPath
	opts.format = defaultFormat
	opts.transport = defaults.Connection.Transport
	opts.address = defaults.Connection.Address
	opts.unitID = uint(defaults.Connection.UnitID)
	opts.timeout = defaults.Connection.Timeout
	opts.baudRate = defaults.RTU.BaudRate
	opts.dataBits = defaults.RTU.DataBits
	opts.stopBits = defaults.RTU.StopBits
	opts.parity = defaults.RTU.Parity

	fs.StringVar(&opts.configPath, "config", config.DefaultConfigPath, "YAML config file")
	fs.StringVar(&opts.profile, "profile", "", "config profile name")
	fs.StringVar(&opts.transport, "transport", opts.transport, "Modbus transport: tcp or rtu")
	if includeConnectionAddress {
		fs.StringVar(&opts.address, "address", opts.address, "TCP host:port or serial device path")
	} else {
		fs.StringVar(&opts.address, "connect-address", opts.address, "TCP host:port or serial device path")
	}
	fs.UintVar(&opts.unitID, "unit-id", opts.unitID, "Modbus unit/slave id")
	fs.DurationVar(&opts.timeout, "timeout", opts.timeout, "request timeout")
	if includeFormat {
		fs.StringVar(&opts.format, "format", defaultFormat, formatHelp)
	}
	fs.IntVar(&opts.baudRate, "baud-rate", opts.baudRate, "RTU baud rate")
	fs.IntVar(&opts.dataBits, "data-bits", opts.dataBits, "RTU data bits")
	fs.IntVar(&opts.stopBits, "stop-bits", opts.stopBits, "RTU stop bits")
	fs.StringVar(&opts.parity, "parity", opts.parity, "RTU parity: N, E, or O")
	fs.BoolVar(&opts.verbose, "verbose", false, "print high-level connection decisions")
	fs.BoolVar(&opts.debug, "debug", false, "enable lower-level Modbus client debug logging")
}

func (opts commonOptions) loadConfig(fs *flag.FlagSet, defaultFormat string) (config.Config, string, error) {
	visited := visitedFlags(fs)
	overrides := config.Overrides{}
	if visited["transport"] {
		overrides.Transport = opts.transport
	}
	if visited["address"] || visited["connect-address"] {
		overrides.Address = opts.address
	}
	if visited["unit-id"] {
		if opts.unitID > 247 {
			return config.Config{}, "", fmt.Errorf("%w: --unit-id must be between 1 and 247", config.ErrConfig)
		}
		overrides.UnitID = &opts.unitID
	}
	if visited["timeout"] {
		overrides.Timeout = &opts.timeout
	}
	if visited["format"] {
		overrides.Format = opts.format
	}
	if visited["baud-rate"] {
		overrides.BaudRate = &opts.baudRate
	}
	if visited["data-bits"] {
		overrides.DataBits = &opts.dataBits
	}
	if visited["stop-bits"] {
		overrides.StopBits = &opts.stopBits
	}
	if visited["parity"] {
		overrides.Parity = opts.parity
	}
	cfg, err := config.LoadForProfile(opts.configPath, opts.profile, overrides)
	if err != nil {
		return cfg, "", err
	}
	format := cfg.Output.Format
	if format == "" {
		format = defaultFormat
	}
	if visited["format"] {
		format = opts.format
	}
	if format == "" {
		format = output.FormatTable
	}
	return cfg, format, nil
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func validateSnapshotFormat(format string) error {
	switch output.NormaliseFormat(format) {
	case output.FormatTable, output.FormatText, output.FormatJSON, output.FormatJSONL, output.FormatCSV:
		return nil
	default:
		return fmt.Errorf("%w: invalid output format %q", config.ErrConfig, format)
	}
}

func validateStreamFormat(format string) error {
	switch output.NormaliseFormat(format) {
	case output.FormatJSON:
		return fmt.Errorf("%w: stream commands use line-delimited output; use --format jsonl instead of --format json", config.ErrConfig)
	case output.FormatText, output.FormatJSONL, output.FormatCSV:
		return nil
	default:
		return fmt.Errorf("%w: invalid stream output format %q", config.ErrConfig, format)
	}
}
