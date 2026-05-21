package cli

import (
	"fmt"
	"io"
)

func (a *App) completions(args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(a.err, "Usage of completions:")
		fmt.Fprintln(a.err, "  modbus-cli completions bash|zsh")
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: modbus-cli completions bash|zsh")
	}
	return writeCompletion(a.out, args[0])
}

func writeCompletion(w io.Writer, shell string) error {
	switch shell {
	case "bash":
		_, err := fmt.Fprintln(w, `# bash completion for modbus-cli
_modbus_cli_complete() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local cmds="init-config validate-config test-connection status read write points read-point write-point watch-point sunspec identify watch completions version"
  COMPREPLY=( $(compgen -W "$cmds" -- "$cur") )
}
complete -F _modbus_cli_complete modbus-cli`)
		return err
	case "zsh":
		_, err := fmt.Fprintln(w, `#compdef modbus-cli
_arguments \
  '1:command:(init-config validate-config test-connection status read write points read-point write-point watch-point sunspec identify watch completions version)' \
  '*::arg:->args'`)
		return err
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}
