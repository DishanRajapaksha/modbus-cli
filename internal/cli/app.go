package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DishanRajapaksha/industrial-cli-kit/command"
	"github.com/DishanRajapaksha/industrial-cli-kit/exitcode"
	"github.com/DishanRajapaksha/modbus-cli/internal/config"
	"github.com/DishanRajapaksha/modbus-cli/internal/modbusclient"
	"github.com/DishanRajapaksha/modbus-cli/internal/output"
)

const (
	appName           = "modbus-cli"
	exitSuccess       = int(exitcode.Success)
	exitGeneralError  = int(exitcode.General)
	exitConfigError   = int(exitcode.Config)
	exitConnection    = int(exitcode.Connection)
	exitRequestError  = int(exitcode.Request)
	exitWriteRejected = int(exitcode.Rejected)
	exitTimeout       = int(exitcode.Timeout)
	exitOutputError   = int(exitcode.Output)
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
	case "sunspec":
		err = a.sunspec(args[1:])
	case "identify":
		err = a.identify(args[1:])
	case "watch":
		err = a.watch(args[1:])
	case "completions":
		err = a.completions(args[1:])
	default:
		a.writeRegistryUsageTo(a.err)
		fmt.Fprintf(a.err, "unknown command %q\n", args[0])
		return exitConfigError
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
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(strings.ToLower(err.Error()), "timeout"):
		return exitTimeout
	case errors.Is(err, output.ErrOutput):
		return exitOutputError
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
	a.writeRegistryUsage()
}

func (a *App) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.err)
	return fs
}

func normaliseGlobalFlags(args []string) ([]string, error) {
	return command.NormalizeGlobalFlagsForRegistry(args, cliRegistry)
}
