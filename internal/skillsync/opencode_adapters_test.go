package skillsync

// Feature 014 B6 / Option A: reviewed fleet adapters live in
// opencode-pack/adapters/. They are resolver-only render sources, not part of
// the Feature 009 `--with-opencode` install set, which reads agents/ only.
// This test pins that exclusion so a later change cannot silently grow the
// installed profile count beyond 14.

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWithOpenCodeExcludesFleetAdapters(t *testing.T) {
	roots := openCodePackRoots(t)
	wantAdapters := []string{"sdd-plan-9r.md", "sdd-spec-9r.md"}
	for _, rootName := range []string{"embedded", "on-disk"} {
		root := roots[rootName]
		entries, err := fs.ReadDir(root, "adapters")
		if err != nil {
			t.Fatalf("%s: read adapters: %v", rootName, err)
		}
		var names []string
		for _, entry := range entries {
			if entry.IsDir() {
				t.Fatalf("%s: adapters/%s is a directory", rootName, entry.Name())
			}
			names = append(names, entry.Name())
		}
		if fmt.Sprint(names) != fmt.Sprint(wantAdapters) {
			t.Fatalf("%s: adapters = %v, want %v", rootName, names, wantAdapters)
		}
		for _, name := range names {
			if _, err := fs.Stat(root, "agents/"+name); err == nil {
				t.Fatalf("%s: adapters/%s also exists in the installed agents/ set", rootName, name)
			}
		}
	}

	home := t.TempDir()
	var out bytes.Buffer
	if err := SyncWithOptions("", home, true, Options{WithOpenCode: true}, &out); err != nil {
		t.Fatalf("apply --with-opencode: %v", err)
	}
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatalf("read installed agents: %v", err)
	}
	if len(entries) != 14 {
		t.Fatalf("installed %d agent profiles, want 14", len(entries))
	}
	for _, name := range wantAdapters {
		if _, err := os.Stat(filepath.Join(agentsDir, name)); err == nil {
			t.Fatalf("%s was installed by --with-opencode; adapters are fleet-only", name)
		}
	}
}
