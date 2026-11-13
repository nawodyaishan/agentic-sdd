package skillsync

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Feature 015 / MCP-A / T30 — AC-55, AC-73.
//
// `--with-opencode` must never touch OpenCode MCP configuration: a seeded
// global opencode.json(c) stays byte-identical through preview and apply, and
// no MCP state (backup, manifest or entry) appears under <home>/.agentic-sdd/mcp/.
//
// The test is fully offline and deterministic: it uses only t.TempDir homes and
// the embedded skills. No fake server, model call, package manager, vendor
// installer or remote MCP endpoint is started or contacted. The environment is
// pinned to the temporary home and HTTP(S) proxying is pointed at a closed
// port, so any accidental outbound request fails immediately instead of
// reaching the network.

// mcpCompatSeedJSON is a deterministic V2-shaped global OpenCode config with
// an unrelated top-level field and an unrelated (non-Agentic-SDD) MCP server
// that must survive byte-for-byte.
const mcpCompatSeedJSON = `{
  "$schema": "https://opencode.ai/config.json",
  "unrelatedTopLevelField": {"keep": "unchanged"},
  "mcp": {
    "servers": {
      "unrelated-server": {
        "type": "local",
        "command": ["npx", "-y", "unrelated-mcp-server"],
        "disabled": false
      }
    }
  }
}
`

// mcpCompatSeedJSONC is the same config as a JSONC file with a comment, so any
// reserialization would change the bytes even if the data stayed equivalent.
const mcpCompatSeedJSONC = `// seeded comment: byte-identical survival is the contract
{
  "unrelatedTopLevelField": {"keep": "unchanged"},
  "mcp": {
    "servers": {
      "unrelated-server": {"type": "local", "command": ["npx", "-y", "unrelated-mcp-server"], "disabled": false}
    }
  }
}
`

// mcpCompatSeedHome creates a temporary home containing the seeded config and
// pins HOME/XDG/proxy at that home so no code path can fall back to the real
// user home or reach the network. The real ~/.config/opencode is never read or
// written by this test.
func mcpCompatSeedHome(t *testing.T, file, seed string) string {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("seed config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, file), []byte(seed), 0o644); err != nil {
		t.Fatalf("seed %s: %v", file, err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	// Offline tripwire: any accidental HTTP request hits a closed local port
	// and fails fast; no fake server is started.
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("NO_PROXY", "")
	return home
}

// mcpCompatReadConfig returns the current bytes of the seeded config file.
func mcpCompatReadConfig(t *testing.T, home, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, ".config", "opencode", file))
	if err != nil {
		t.Fatalf("read seeded %s: %v", file, err)
	}
	return data
}

// mcpCompatAssertNoMCPState fails if an MCP-named path outside the managed
// client skill roots appears below home, which covers <home>/.agentic-sdd/mcp
// (manifest/backups) and any other MCP artifact a sync run might create.
func mcpCompatAssertNoMCPState(t *testing.T, home, phase string) {
	t.Helper()
	// Installed portable skills are not MCP state. Feature 018 ships a
	// legitimate search-mcp-workflow.md reference under each client's skills
	// root. The seeded config and .agentic-sdd/mcp assertions below still
	// cover the MCP contract.
	skillRoots := make(map[string]bool)
	for _, target := range clientTargets(home) {
		skillRoots[filepath.Clean(target.path)] = true
	}
	skillRoots[filepath.Join(home, ".config", "opencode", "skills")] = true
	err := filepath.WalkDir(home, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(home, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if skillRoots[filepath.Clean(path)] {
			return filepath.SkipDir
		}
		name := strings.ToLower(filepath.Base(rel))
		if strings.Contains(name, "mcp") {
			t.Errorf("%s created MCP artifact %q under home", phase, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk home for %s: %v", phase, err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".agentic-sdd", "mcp")); !os.IsNotExist(err) {
		t.Errorf("%s: <home>/.agentic-sdd/mcp must not exist (stat err = %v)", phase, err)
	}
}

// mcpCompatAssertTranscriptOffline fails if the sync transcript references MCP
// work or a remote endpoint, which would mean --with-opencode started doing MCP
// or network work. The temporary home path is masked first because it embeds
// this test's name (which contains "MCP").
func mcpCompatAssertTranscriptOffline(t *testing.T, out, home, phase string) {
	t.Helper()
	lower := strings.ToLower(strings.ReplaceAll(out, home, "<home>"))
	if strings.Contains(lower, "mcp") {
		t.Errorf("%s transcript mentions MCP work:\n%s", phase, out)
	}
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		t.Errorf("%s transcript references a remote endpoint:\n%s", phase, out)
	}
}

// TestMCPCompatWithOpenCodeLeavesOpenCodeConfigByteIdentical covers AC-55:
// preview and apply with --with-opencode leave opencode.json/opencode.jsonc
// byte-identical and create no MCP backup, manifest or entry, even when the
// agent pack is installed. Each mode runs in its own disposable temp home, so
// unrelated skill directories installed by apply are deliberately not asserted.
func TestMCPCompatWithOpenCodeLeavesOpenCodeConfigByteIdentical(t *testing.T) {
	cases := []struct {
		name string
		file string
		seed string
	}{
		{name: "json", file: "opencode.json", seed: mcpCompatSeedJSON},
		{name: "jsonc", file: "opencode.jsonc", seed: mcpCompatSeedJSONC},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := []byte(tc.seed)

			// Preview home: read-only transcript must change nothing.
			previewHome := mcpCompatSeedHome(t, tc.file, tc.seed)
			var previewOut bytes.Buffer
			if err := SyncWithOptions("", previewHome, false, Options{WithOpenCode: true}, &previewOut); err != nil {
				t.Fatalf("preview --with-opencode failed: %v", err)
			}
			previewBytes := mcpCompatReadConfig(t, previewHome, tc.file)
			if !bytes.Equal(previewBytes, want) {
				t.Errorf("preview changed %s:\ngot  %q\nwant %q", tc.file, previewBytes, want)
			}
			mcpCompatAssertNoMCPState(t, previewHome, "preview")
			mcpCompatAssertTranscriptOffline(t, previewOut.String(), previewHome, "preview")
			previewEntries, err := os.ReadDir(filepath.Join(previewHome, ".config", "opencode"))
			if err != nil {
				t.Fatalf("read preview config dir: %v", err)
			}
			for _, entry := range previewEntries {
				if entry.Name() != tc.file {
					t.Errorf("preview created %q next to the seeded config", entry.Name())
				}
			}

			// Apply home: a real install (skills plus the agent pack) must
			// still leave the MCP configuration untouched.
			applyHome := mcpCompatSeedHome(t, tc.file, tc.seed)
			var applyOut bytes.Buffer
			if err := SyncWithOptions("", applyHome, true, Options{WithOpenCode: true}, &applyOut); err != nil {
				t.Fatalf("apply --with-opencode failed: %v", err)
			}
			applyBytes := mcpCompatReadConfig(t, applyHome, tc.file)
			if !bytes.Equal(applyBytes, want) {
				t.Errorf("apply changed %s:\ngot  %q\nwant %q", tc.file, applyBytes, want)
			}
			if !bytes.Equal(previewBytes, applyBytes) {
				t.Errorf("%s differs between preview and apply:\npreview %q\napply   %q", tc.file, previewBytes, applyBytes)
			}
			mcpCompatAssertNoMCPState(t, applyHome, "apply")
			mcpCompatAssertTranscriptOffline(t, applyOut.String(), applyHome, "apply")

			// AC-55 "even when an agent pack is installed": prove the apply
			// really installed the pack next to the untouched config, and that
			// nothing else appeared in the OpenCode config directory.
			agents, err := os.ReadDir(filepath.Join(applyHome, ".config", "opencode", "agents"))
			if err != nil {
				t.Fatalf("apply did not install the agent pack: %v", err)
			}
			if len(agents) == 0 {
				t.Fatal("apply installed no agent profiles; byte-identity would be unproven")
			}
			applyEntries, err := os.ReadDir(filepath.Join(applyHome, ".config", "opencode"))
			if err != nil {
				t.Fatalf("read apply config dir: %v", err)
			}
			for _, entry := range applyEntries {
				if entry.Name() != tc.file && entry.Name() != "agents" {
					t.Errorf("apply created %q next to the seeded config", entry.Name())
				}
			}
		})
	}
}
