package cli

import (
	"io"

	sharedhelp "github.com/DishanRajapaksha/industrial-cli-kit/help"
)

func (a *App) writeRegistryUsage() {
	a.writeRegistryUsageTo(a.out)
}

func (a *App) writeRegistryUsageTo(w io.Writer) {
	_ = sharedhelp.Write(w, cliRegistry, sharedhelp.Options{
		Description: "modbus-cli is a script-friendly Modbus TCP and RTU command-line client.",
		Usage: []string{
			"modbus-cli [global flags] <command> [flags]",
		},
		Examples: []string{
			"modbus-cli init-config",
			"modbus-cli validate-config --profile local",
			"modbus-cli test-connection --transport tcp --connect-address 127.0.0.1:502",
			"modbus-cli read holding-registers --address 0 --quantity 2 --type float32",
			"modbus-cli read-point active_power",
			"modbus-cli write register --address 10 --type uint16 --value 42 --yes",
			"modbus-cli write-point breaker_closed --value on --yes",
			"modbus-cli sunspec scan",
			"modbus-cli watch coils --address 0 --quantity 8 --interval 1s --format jsonl",
			"modbus-cli completions zsh",
		},
	})
}
