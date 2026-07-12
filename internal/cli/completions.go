package cli

import (
	"fmt"
	"strings"

	"github.com/DishanRajapaksha/industrial-cli-kit/completion"
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
	return completion.Write(a.out, strings.ToLower(strings.TrimSpace(args[0])), cliRegistry)
}
