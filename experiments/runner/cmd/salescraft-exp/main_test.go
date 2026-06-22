package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadBuildStateAnnotatedNextEligibleHeader(t *testing.T) {
	workspace := t.TempDir()
	content := strings.Join([]string{
		"# Build State",
		"",
		"## Status: ready",
		"",
		"## Completed",
		"- [x] foundation/root-config",
		"",
		"## In Progress",
		"- (none)",
		"",
		"## Blocked",
		"- (none)",
		"",
		"## Next Eligible (Phase 4 - Domain Features)",
		"- domain/relationships (deps: base/contacts-module)",
		"- domain/products (deps: core/auth-module)",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(workspace, "BUILD_STATE.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := readBuildState(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if state.Complete {
		t.Fatal("expected annotated Next Eligible header to prevent complete=true")
	}
	if state.NextEligibleSummary == "none" {
		t.Fatalf("expected next eligible items, got %q", state.NextEligibleSummary)
	}
}

