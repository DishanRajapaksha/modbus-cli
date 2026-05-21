package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
)

const (
	appName           = "modbus-cli"
	exitSuccess       = 0
	exitGeneralError  = 1
	exitConfigError   = 2
	exitConnection    = 3
	exitRequestError  = 4
	exitOutputError   = 9
	defaultStreamHelp = "output format: text, jsonl, or csv"
)

type App struct {
	out     io.Writer
	err     io.Writer
	factory modbusclient.Factory
}

func NewApp(out io.Writer, err io.Writer) *App {
	return &App{out: out, err: err, factory: modbusclient.GridXFactory{}}
}

func NewAppWithFactory(out io.Writer, err io.Writer, factory modbusclient.Factory) *App {
	return &App{out: out, err: err, factory: factory}
}

func Main() {
	code := NewApp(os.Stdout, os.Stderr).Run(os.Args[1:])
	if code != 0 {
		os.Exit(code)
	}
}

func (a *App) Run(args []string) int {
	normalised, err := normaliseGlobalFlags(args)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return exitConfigError
	}
	args = normalised
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		a.printUsage()
		return exitSuccess
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(a.out, "%s development\n", appName)
		return exitSuccess
	}

	switch args[0] {
	case "init-config":
		err = a.initConfig(args[1:])
	case "validate-config":
		err = a.validateConfig(args[1:])
	case "test-connection", "status":
		err = a.testConnection(args[1:])
	case "read":
		err = a.read(args[1:])
	case "write":
		err = a.write(args[1:])
	case "points":
		err = a.points(args[1:])
	case "read-point":
		err = a.readPoint(args[1:])
	case "write-point":
		err = a.writePoint(args[1:])
	case "watch-point":
		err = a.watchPoint(args[1:])
	case "identify":
		err = a.identify(args[1:])
	case "watch":
		err = a.watch(args[1:])
	case "completions":
		err = a.completions(args[1:])
	default:
		a.printUsage()
		fmt.Fprintf(a.err, "unknown command %q\n", args[0])
		return exitGeneralError
	}

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitSuccess
		}
		fmt.Fprintln(a.err, err)
		return mapExitCode(err)
	}
	return exitSuccess
}

func mapExitCode(err error) int {
	switch {
	case err == nil:
		return exitSuccess
	case errors.Is(err, flag.ErrHelp):
		return exitSuccess
	case errors.Is(err, config.ErrConfig):
		return exitConfigError
	case errors.Is(err, modbusclient.ErrValidation):
		return exitConfigError
	case errors.Is(err, modbusclient.ErrConnection):
		return exitConnection
	case errors.Is(err, modbusclient.ErrRequest):
		return exitRequestError
	case strings.Contains(err.Error(), "flag provided but not defined"):
		return exitConfigError
	case strings.Contains(err.Error(), "invalid value"):
		return exitConfigError
	case strings.Contains(err.Error(), "requires a value"):
		return exitConfigError
	default:
		return exitGeneralError
	}
}

func (a *App) printUsage() {
	fmt.Fprintln(a.out, `modbus-cli is a script-friendly Modbus TCP and RTU command-line client.

Usage:
  modbus-cli [global flags] <command> [flags]
  modbus-cli init-config
  modbus-cli validate-config --profile local
  modbus-cli test-connection --transport tcp --address 127.0.0.1:502
  modbus-cli read holding-registers --address 0 --quantity 2 --type float32
  modbus-cli read-point active_power
  modbus-cli write register --address 10 --type uint16 --value 42 --yes
  modbus-cli write-point breaker_closed --value on --yes
  modbus-cli watch coils --address 0 --quantity 8 --interval 1s --format jsonl
  modbus-cli completions zsh
  modbus-cli version

Commands:
  init-config       Write a starter YAML config file
  validate-config  Validate local config without connecting
  test-connection  Run transport and request diagnostics
  status           Alias for test-connection
  read             Read coils, discrete inputs, holding registers, or input registers
  write            Write coils or holding registers
  points           List configured named points
  read-point       Read a configured named point
  write-point      Write a configured named point
  watch-point      Poll a configured named point
  identify         Read device identification objects
  watch            Poll values repeatedly
  completions      Generate shell completion scripts
  version          Print version information

Common flags:
  --config      YAML config file, defaults to config.yaml
  --profile     Config profile name
  --transport   tcp or rtu
  --address     TCP host:port or serial device path on diagnostics; coil/register address on read/write/watch
  --connect-address
                TCP host:port or serial device path on read/write/watch
  --unit-id     Modbus unit/slave id
  --timeout     Request timeout
  --format      table, text, json, jsonl, or csv
  --verbose     Print high-level connection decisions
  --debug       Enable lower-level Modbus client debug logging`)
}

func (a *App) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.err)
	return fs
}

func normaliseGlobalFlags(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	var globals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 >= len(args) {
				return nil, errors.New("command is required after --")
			}
			return appendCommandGlobals(args[i+1:], globals), nil
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return appendCommandGlobals(args[i:], globals), nil
		}
		if arg == "--help" || arg == "-h" || arg == "--version" || arg == "-v" {
			return args[i:], nil
		}
		name, inlineValue, hasInlineValue := strings.Cut(arg, "=")
		switch name {
		case "--verbose", "--debug":
			if hasInlineValue {
				return nil, fmt.Errorf("%s does not take a value", name)
			}
			globals = append(globals, name)
		case "--config", "--profile", "--transport", "--address", "--connect-address", "--unit-id", "--timeout", "--format", "--baud-rate", "--data-bits", "--stop-bits", "--parity":
			value := inlineValue
			if !hasInlineValue {
				i++
				if i >= len(args) || strings.HasPrefix(args[i], "-") {
					return nil, fmt.Errorf("%s requires a value", name)
				}
				value = args[i]
			}
			if value == "" {
				return nil, fmt.Errorf("%s requires a value", name)
			}
			globals = append(globals, name, value)
		default:
			return nil, fmt.Errorf("unknown global flag %q", name)
		}
	}
	return nil, errors.New("command is required")
}

func appendCommandGlobals(args []string, globals []string) []string {
	if len(args) == 0 || len(globals) == 0 {
		return args
	}
	if !commandSupportsGlobals(args[0]) {
		return args
	}
	globals = rewriteGlobalsForCommand(args[0], globals)
	out := make([]string, 0, len(args)+len(globals))
	if commandTakesTypeBeforeFlags(args[0]) && len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		out = append(out, args[0], args[1])
		out = append(out, globals...)
		out = append(out, args[2:]...)
		return out
	}
	out = append(out, args[0])
	out = append(out, globals...)
	out = append(out, args[1:]...)
	return out
}

func commandTakesTypeBeforeFlags(command string) bool {
	switch command {
	case "read", "write", "watch", "read-point", "write-point", "watch-point":
		return true
	default:
		return false
	}
}

func rewriteGlobalsForCommand(command string, globals []string) []string {
	if command != "read" && command != "write" && command != "watch" && command != "read-point" && command != "write-point" && command != "watch-point" {
		return globals
	}
	out := append([]string(nil), globals...)
	for i := 0; i < len(out); i++ {
		if out[i] == "--address" {
			out[i] = "--connect-address"
			i++
		}
	}
	return out
}

func commandSupportsGlobals(command string) bool {
	switch command {
	case "validate-config", "test-connection", "status", "read", "write", "points", "read-point", "write-point", "watch-point", "identify", "watch":
		return true
	default:
		return false
	}
}
