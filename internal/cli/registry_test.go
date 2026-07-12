package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/industrial-cli-kit/command"
	"github.com/DishanRajapaksha/industrial-cli-kit/completion"
)

func TestRegistryMatchesDispatcher(t *testing.T) {
	dispatched := []string{
		"init-config", "validate-config", "test-connection", "status", "read", "write",
		"points", "read-point", "write-point", "watch-point", "sunspec", "identify",
		"watch", "completions", "help", "version",
	}
	registered := map[string]bool{}
	for _, command := range cliRegistry.Commands {
		if registered[command.Name] {
			t.Fatalf("duplicate registry command %q", command.Name)
		}
		registered[command.Name] = true
	}
	for _, name := range dispatched {
		if !registered[name] {
			t.Errorf("dispatcher command %q is not registered", name)
		}
	}
	for name := range registered {
		found := false
		for _, candidate := range dispatched {
			if candidate == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("registered command %q is not dispatched", name)
		}
	}
}

func TestRegistryGlobalFlagsMatchNormalizer(t *testing.T) {
	for _, global := range cliRegistry.GlobalFlags {
		args := []string{"--" + global.Name}
		if global.TakesValue {
			args = append(args, "value")
		}
		args = append(args, "status")
		normalised, err := normaliseGlobalFlags(args)
		if err != nil {
			t.Errorf("registered global flag --%s is rejected: %v", global.Name, err)
			continue
		}
		if len(normalised) == 0 || normalised[0] != "status" {
			t.Errorf("normalising --%s produced %v", global.Name, normalised)
		}
	}
}

func TestRegistryNormalizerPreservesLeadingArguments(t *testing.T) {
	got, err := normaliseGlobalFlags([]string{
		"--connect-address", "192.0.2.10:502", "read", "holding-registers", "--address", "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"read", "holding-registers", "--connect-address", "192.0.2.10:502", "--address", "0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normaliseGlobalFlags() = %#v, want %#v", got, want)
	}

	got, err = normaliseGlobalFlags([]string{"--profile", "local", "read-point", "active_power", "--format", "json"})
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"read-point", "active_power", "--profile", "local", "--format", "json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normaliseGlobalFlags() = %#v, want %#v", got, want)
	}
}

func TestRegistryContainsNestedOperationsAndAccurateFlags(t *testing.T) {
	commands := map[string]struct {
		leading int
		subs    []string
	}{
		"read":        {leading: 1, subs: []string{"coils", "discrete-inputs", "holding-registers", "input-registers"}},
		"write":       {leading: 1, subs: []string{"coil", "coils", "register", "registers"}},
		"watch":       {leading: 1, subs: []string{"coils", "discrete-inputs", "holding-registers", "input-registers"}},
		"sunspec":     {leading: 1, subs: []string{"scan", "models", "read"}},
		"read-point":  {leading: 1},
		"write-point": {leading: 1},
		"watch-point": {leading: 1},
	}

	for _, registered := range cliRegistry.Commands {
		expected, ok := commands[registered.Name]
		if !ok {
			continue
		}
		if registered.LeadingArgs != expected.leading {
			t.Errorf("%s LeadingArgs=%d, want %d", registered.Name, registered.LeadingArgs, expected.leading)
		}
		if len(expected.subs) == 0 {
			continue
		}
		found := map[string]bool{}
		for _, subcommand := range registered.Subcommands {
			found[subcommand.Name] = true
		}
		for _, name := range expected.subs {
			if !found[name] {
				t.Errorf("%s registry missing subcommand %q", registered.Name, name)
			}
		}
	}

	watchPoint := registryCommand(t, "watch-point")
	assertFlag(t, watchPoint.Flags, "interval", true)
	assertFlag(t, watchPoint.Flags, "duration", true)
	for _, flag := range watchPoint.Flags {
		if flag.Name == "count" {
			t.Fatal("watch-point registry still exposes nonexistent --count")
		}
	}

	writePoint := registryCommand(t, "write-point")
	assertFlag(t, writePoint.Flags, "yes", false)
	assertFlag(t, writePoint.Flags, "dry-run", false)
}

func TestGeneratedCompletionsContainNestedCommandsAndSafetyFlags(t *testing.T) {
	var out bytes.Buffer
	if err := completion.Write(&out, "bash", cliRegistry); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"read:holding-registers", "write:register", "sunspec:models", "sunspec:read",
		"--model", "--point", "--yes", "--dry-run", "--duration",
		"complete -F _modbus_cli_completion modbus-cli",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("completion output missing %q", want)
		}
	}
}

func registryCommand(t *testing.T, name string) command.Command {
	t.Helper()
	for _, registered := range cliRegistry.Commands {
		if registered.Name == name {
			return registered
		}
	}
	t.Fatalf("registry command %q not found", name)
	return command.Command{}
}

func assertFlag(t *testing.T, flags []command.Flag, name string, takesValue bool) {
	t.Helper()
	for _, flag := range flags {
		if flag.Name == name {
			if flag.TakesValue != takesValue {
				t.Fatalf("flag --%s TakesValue=%v, want %v", name, flag.TakesValue, takesValue)
			}
			return
		}
	}
	t.Fatalf("flag --%s not found", name)
}
