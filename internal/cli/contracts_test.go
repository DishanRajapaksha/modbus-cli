package cli

import (
	"context"
	"reflect"
	"testing"
)

func TestSnapshotFormatsExcludeJSONL(t *testing.T) {
	if err := validateSnapshotFormat("jsonl"); err == nil {
		t.Fatal("snapshot commands must reject jsonl")
	}
	for _, format := range []string{"table", "text", "json", "csv"} {
		if err := validateSnapshotFormat(format); err != nil {
			t.Fatalf("snapshot format %q rejected: %v", format, err)
		}
	}
}

func TestStreamFormatsUseLineDelimitedOutput(t *testing.T) {
	for _, format := range []string{"text", "jsonl", "csv"} {
		if err := validateStreamFormat(format); err != nil {
			t.Fatalf("stream format %q rejected: %v", format, err)
		}
	}
	for _, format := range []string{"table", "json"} {
		if err := validateStreamFormat(format); err == nil {
			t.Fatalf("stream format %q must be rejected", format)
		}
	}
}

func TestConnectionAddressHasUnambiguousGlobalFlag(t *testing.T) {
	got, err := normaliseGlobalFlags([]string{"--connect-address", "192.0.2.10:502", "read", "coils"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"read", "coils", "--connect-address", "192.0.2.10:502"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalised arguments = %#v, want %#v", got, want)
	}
	if _, err := normaliseGlobalFlags([]string{"--address", "192.0.2.10:502", "read", "coils"}); err == nil {
		t.Fatal("ambiguous global --address must be rejected")
	}
}

func TestTimeoutExitCodeIsSharedContractValue(t *testing.T) {
	if got := mapExitCode(context.DeadlineExceeded); got != exitTimeout {
		t.Fatalf("timeout exit code = %d, want %d", got, exitTimeout)
	}
	if exitTimeout != 8 || exitWriteRejected != 7 || exitOutputError != 9 {
		t.Fatalf("shared exit-code contract changed: rejected=%d timeout=%d output=%d", exitWriteRejected, exitTimeout, exitOutputError)
	}
}
