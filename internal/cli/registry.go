package cli

import "github.com/DishanRajapaksha/industrial-cli-kit/command"

var cliRegistry = command.Registry{
	Binary: appName,
	GlobalFlags: []command.Flag{
		{Name: "config", TakesValue: true, Summary: "YAML config file"},
		{Name: "profile", TakesValue: true, Summary: "config profile name"},
		{Name: "transport", TakesValue: true, Summary: "tcp or rtu"},
		{Name: "connect-address", TakesValue: true, Summary: "TCP address or serial device"},
		{Name: "unit-id", TakesValue: true, Summary: "Modbus unit identifier"},
		{Name: "timeout", TakesValue: true, Summary: "request timeout"},
		{Name: "format", TakesValue: true, Summary: "output format"},
		{Name: "baud-rate", TakesValue: true, Summary: "RTU baud rate"},
		{Name: "data-bits", TakesValue: true, Summary: "RTU data bits"},
		{Name: "stop-bits", TakesValue: true, Summary: "RTU stop bits"},
		{Name: "parity", TakesValue: true, Summary: "RTU parity"},
		{Name: "verbose", Summary: "print connection decisions"},
		{Name: "debug", Summary: "enable protocol debug logging"},
	},
	Commands: []command.Command{
		{Name: "init-config", Summary: "Write a starter YAML config file", Flags: registryFlags("output", "force")},
		{Name: "validate-config", Summary: "Validate local config without connecting"},
		{Name: "test-connection", Summary: "Run transport and request diagnostics"},
		{Name: "status", Summary: "Alias for test-connection"},
		{Name: "read", Summary: "Read Modbus values", LeadingArgs: 1, Subcommands: readRegistryCommands()},
		{Name: "write", Summary: "Write Modbus values", LeadingArgs: 1, Subcommands: writeRegistryCommands()},
		{Name: "points", Summary: "List configured named points"},
		{Name: "read-point", Summary: "Read a configured named point", LeadingArgs: 1},
		{Name: "write-point", Summary: "Write a configured named point", LeadingArgs: 1, Flags: registryFlags("value", "yes", "dry-run")},
		{Name: "watch-point", Summary: "Poll a configured named point", LeadingArgs: 1, Flags: registryFlags("interval", "duration")},
		{Name: "sunspec", Summary: "Scan or read SunSpec models", LeadingArgs: 1, Subcommands: sunSpecRegistryCommands()},
		{Name: "identify", Summary: "Read device identification objects", Flags: registryFlags("level")},
		{Name: "watch", Summary: "Poll Modbus values", LeadingArgs: 1, Flags: registryFlags("interval", "duration"), Subcommands: readRegistryCommands()},
		{Name: "completions", Summary: "Generate shell completion scripts", LeadingArgs: 1},
		{Name: "help", Summary: "Print help"},
		{Name: "version", Summary: "Print version information"},
	},
}

func readRegistryCommands() []command.Command {
	flags := registryFlags("address", "quantity", "type", "byte-order", "word-order")
	return []command.Command{
		{Name: "coils", Summary: "Read coils", Flags: flags},
		{Name: "discrete-inputs", Summary: "Read discrete inputs", Flags: flags},
		{Name: "holding-registers", Summary: "Read holding registers", Flags: flags},
		{Name: "input-registers", Summary: "Read input registers", Flags: flags},
	}
}

func writeRegistryCommands() []command.Command {
	flags := registryFlags("address", "value", "type", "byte-order", "word-order", "yes", "dry-run")
	return []command.Command{
		{Name: "coil", Summary: "Write one coil", Flags: flags},
		{Name: "coils", Summary: "Write multiple coils", Flags: flags},
		{Name: "register", Summary: "Write one register", Flags: flags},
		{Name: "registers", Summary: "Write multiple registers", Flags: flags},
	}
}

func sunSpecRegistryCommands() []command.Command {
	return []command.Command{
		{Name: "scan", Summary: "Scan SunSpec models"},
		{Name: "models", Summary: "Alias for SunSpec scan"},
		{Name: "read", Summary: "Read a SunSpec point", Flags: registryFlags("model", "point")},
	}
}

func registryFlags(names ...string) []command.Flag {
	flags := make([]command.Flag, 0, len(names))
	for _, name := range names {
		takesValue := name != "force" && name != "yes" && name != "dry-run"
		flags = append(flags, command.Flag{Name: name, TakesValue: takesValue})
	}
	return flags
}

func registryGlobalFlag(name string) (command.Flag, bool) {
	for _, flag := range cliRegistry.GlobalFlags {
		if flag.Name == name {
			return flag, true
		}
	}
	return command.Flag{}, false
}
