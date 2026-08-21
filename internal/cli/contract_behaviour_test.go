package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/DishanRajapaksha/industrial-cli-kit/contracttest"
	"github.com/DishanRajapaksha/industrial-cli-kit/exitcode"
	"github.com/DishanRajapaksha/modbus-cli/internal/config"
)

func runContractCLI(args ...string) contracttest.Result {
	var out, errOut bytes.Buffer
	code := NewAppWithFactory(&out, &errOut, fakeFactory{}).Run(args)
	return contracttest.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
}

func writeContractConfig(t *testing.T) string {
	t.Helper()
	contents, err := config.StarterConfigYAML()
	if err != nil {
		t.Fatalf("render starter config: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write contract config: %v", err)
	}
	return path
}

func TestSharedGlobalFlagContract(t *testing.T) {
	configPath := writeContractConfig(t)
	run := func(args ...string) contracttest.Result {
		return runContractCLI(append([]string{"--config", configPath}, args...)...)
	}
	contracttest.GlobalFlagPositioning(t, run, contracttest.GlobalFlagOptions{
		Success: []string{"validate-config"},
		Probes:  []contracttest.GlobalFlagProbe{{Name: "config", Value: configPath}, {Name: "verbose"}},
	})
	contracttest.RejectsUnsupportedGlobalFlag(t, run)
}

func TestSharedSnapshotFormatContract(t *testing.T) {
	configPath := writeContractConfig(t)
	run := func(args ...string) contracttest.Result {
		return runContractCLI(append([]string{"--config", configPath}, args...)...)
	}
	contracttest.Formats(t, run, contracttest.FormatOptions{
		Command:      []string{"validate-config"},
		Kind:         contracttest.SnapshotFormats,
		AcceptedCode: int(exitcode.Success),
	})
}

func TestSharedStreamFormatContract(t *testing.T) {
	configPath := writeContractConfig(t)
	args := []string{"--config", configPath, "watch", "holding-registers", "--address", "0", "--duration", "1s"}
	run := func(extra ...string) contracttest.Result {
		return runContractCLI(append(append([]string{}, args...), extra...)...)
	}
	for _, format := range []string{"text", "jsonl", "csv"} {
		result := run("--format", format)
		if result.Code == int(exitcode.Config) {
			t.Errorf("watch --format %s rejected: %s", format, result.Stderr)
		}
	}
	for _, format := range []string{"yaml", "xml", "json"} {
		result := run("--format", format)
		if result.Code != int(exitcode.Config) {
			t.Errorf("watch --format %s exit = %d, want %d; stderr=%s", format, result.Code, int(exitcode.Config), result.Stderr)
		}
	}
}

func TestSharedOutputSeparation(t *testing.T) {
	contracttest.OutputSeparation(t, runContractCLI, []string{"completions", "bash"})
	contracttest.ErrorSeparation(t, runContractCLI, []string{"definitely-not-a-command"}, exitcode.Config)
}

func TestSharedUsageExitCodeContract(t *testing.T) {
	contracttest.ExitCodes(t, runContractCLI, contracttest.UsageExitCodeScenarios())
}

func TestSharedMutatingSafetyContract(t *testing.T) {
	configPath := writeContractConfig(t)
	base := []string{"--config", configPath, "write", "register", "--address", "10", "--type", "uint16", "--value", "42"}
	run := func(extra ...string) contracttest.Result {
		return runContractCLI(append(append([]string{}, base...), extra...)...)
	}
	dryRun := run()
	if dryRun.Code != int(exitcode.Success) || dryRun.Stderr != "" {
		t.Fatalf("default write = %+v, want dry-run success without diagnostics", dryRun)
	}
	explicit := run("--dry-run")
	if explicit.Code != int(exitcode.Success) {
		t.Fatalf("explicit --dry-run exit = %d; stderr=%s", explicit.Code, explicit.Stderr)
	}
	conflict := run("--yes", "--dry-run")
	if conflict.Code != int(exitcode.Config) {
		t.Fatalf("--yes --dry-run exit = %d, want 2; stderr=%s", conflict.Code, conflict.Stderr)
	}
	authorized := run("--yes")
	if authorized.Code != int(exitcode.Success) {
		t.Fatalf("--yes exit = %d; stderr=%s", authorized.Code, authorized.Stderr)
	}
}
