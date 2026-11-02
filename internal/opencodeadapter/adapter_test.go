package opencodeadapter

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) (string, string, string, []byte) {
	t.Helper()
	top := t.TempDir()
	root, candidate, backups := filepath.Join(top, "project"), filepath.Join(top, "candidate.jsonc"), filepath.Join(top, "backups")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(backups, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := Candidate("demo", "src")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, data, 0600); err != nil {
		t.Fatal(err)
	}
	return root, candidate, backups, data
}

func TestCandidateIsInactiveAndNamespaced(t *testing.T) {
	data, err := Candidate("demo", "src")
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config["model"] != "openrouter/nvidia/nemotron-3-ultra-550b-a55b:free" || config["warming"] != false {
		t.Fatalf("not the inactive selected candidate: %s", data)
	}
	commands, ok := config["commands"].(map[string]any)
	if !ok || len(commands) < 11 {
		t.Fatal("missing core/helper commands")
	}
	for _, value := range commands {
		if value.(map[string]any)["subagent"] != false {
			t.Fatal("child command")
		}
	}
	if bytes.Contains(data, []byte("__FEATURE__")) || bytes.Contains(data, []byte("__BUILD_PATH__")) {
		t.Fatal("unbound paths")
	}
	for _, bad := range []string{"../escape", "/absolute", "src/*", ".git", ".", "src/../other"} {
		if _, err := Candidate("demo", bad); err == nil {
			t.Fatalf("accepted build path %q", bad)
		}
	}
	if _, err := Candidate("../escape", "src"); err == nil {
		t.Fatal("accepted feature escape")
	}
	direct, err := Candidate("", "src")
	if err != nil || bytes.Contains(direct, []byte("specs/")) {
		t.Fatal("direct fix invented feature grants")
	}
}

func TestPreflightProjectionAndRejectedInputs(t *testing.T) {
	data, err := Candidate("demo", "src")
	if err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(data, nil)
	if err != nil || report.OnlineReady || report.Version != 1 || len(report.Profiles) < 3 || len(report.Blockers) == 0 {
		t.Fatalf("unsafe/missing report: %+v %v", report, err)
	}
	projected, _ := json.Marshal(report)
	if !bytes.Contains(projected, []byte(`"target_runtime":"2.2.5"`)) || !bytes.Contains(projected, []byte(`"candidate_profiles"`)) {
		t.Fatal("declared candidate metadata looks like a native observation")
	}
	for _, raw := range []string{`{"api_key":"SECRET-CANARY"}`, `{"providers":{"x":{"headers":{"Authorization":"SECRET-CANARY"}}}}`, `{"model":"a","model":"b"}`, `{"x":null} trailing`} {
		_, err := Inspect([]byte(raw), nil)
		if err == nil || strings.Contains(err.Error(), "SECRET-CANARY") {
			t.Fatalf("unsafe input error: %v", err)
		}
	}
	native := []byte(`{"version":"2.2.5","profiles":["sdd-draft"],"commands":["sdd-draft"],"skills":["agentic-sdd-router"],"warnings":["invalid-policy"],"truncated":true}`)
	report, err = Inspect(data, native)
	if err != nil || report.Observation != "caller-supplied-not-verified" || report.OnlineReady || len(report.Blockers) < 3 {
		t.Fatalf("promoted projection: %+v %v", report, err)
	}
	if _, err := Inspect(data, []byte(`{"version":"2.2.5","headers":{"Authorization":"SECRET-CANARY"}}`)); err == nil || strings.Contains(err.Error(), "SECRET-CANARY") {
		t.Fatal("unknown native fields leaked/accepted")
	}
}

func TestConfigPreviewApplyNoopAndRollback(t *testing.T) {
	root, candidate, backups, data := fixture(t)
	preview, err := Preview(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Original != "absent" || preview.Noop || preview.Candidate == "" {
		t.Fatalf("bad preview: %+v", preview)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode")); !os.IsNotExist(err) {
		t.Fatal("preview wrote config")
	}
	tx, err := Apply(root, candidate, backups, preview.Original, preview.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, ".opencode", "opencode.jsonc")
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(data, got) || tx.Phase != "installed" || tx.Backup == "" {
		t.Fatalf("not installed: %+v %v", tx, err)
	}
	before, _ := os.Stat(dest)
	p, err := Preview(root, candidate)
	if err != nil || !p.Noop {
		t.Fatal("not noop", err)
	}
	noop, err := Apply(root, candidate, backups, p.Original, p.Candidate)
	if err != nil || noop.Backup != "" {
		t.Fatal("noop made backup", err)
	}
	after, _ := os.Stat(dest)
	if !os.SameFile(before, after) {
		t.Fatal("noop rewrote file")
	}
	entries, _ := os.ReadDir(backups)
	if len(entries) != 1 {
		t.Fatal("duplicate backup")
	}
	if _, err := Rollback(root, tx.Backup); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("new config not removed")
	}
	if _, err := Rollback(root, tx.Backup); err != nil {
		t.Fatal("repeat rollback", err)
	}
}

func TestExistingConfigCommentsModeAndInterveningEdit(t *testing.T) {
	root, candidate, backups, _ := fixture(t)
	dir := filepath.Join(root, ".opencode")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "opencode.jsonc")
	old := []byte("// keep exactly\n{\"custom\": \"keep\",}\n")
	if err := os.WriteFile(dest, old, 0640); err != nil {
		t.Fatal(err)
	}
	p, err := Preview(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, candidate, backups, "wrong", p.Candidate); err == nil {
		t.Fatal("missing original pin")
	}
	tx, err := Apply(root, candidate, backups, p.Original, p.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte(`{"user":"later edit"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Rollback(root, tx.Backup); err == nil {
		t.Fatal("discarded later edit")
	}
	data, err := os.ReadFile(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Rollback(root, tx.Backup); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	mode, _ := os.Stat(dest)
	if !bytes.Equal(got, old) || mode.Mode().Perm() != 0640 {
		t.Fatal("lost comments/unrelated bytes/mode")
	}
}

func TestUnsafePathsSensitiveSourceAndChangedCandidate(t *testing.T) {
	root, candidate, backups, data := fixture(t)
	p, err := Preview(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, append(data, ' '), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, candidate, backups, p.Original, p.Candidate); err == nil {
		t.Fatal("accepted changed candidate")
	}
	if err := os.WriteFile(candidate, data, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(root), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(link, candidate); err == nil {
		t.Fatal("symlink root")
	}
	if err := os.Mkdir(filepath.Join(root, ".opencode"), 0700); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(root, ".opencode", "candidate.jsonc")
	if err := os.WriteFile(inside, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(root, inside); err == nil {
		t.Fatal("active discovery candidate")
	}
	if err := os.WriteFile(filepath.Join(root, ".opencode", "opencode.jsonc"), []byte(`{"secret":"SECRET-CANARY"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(root, candidate); err == nil || strings.Contains(err.Error(), "SECRET-CANARY") {
		t.Fatal("accepted/leaked sensitive original")
	}
}

func TestCandidateCannotEnableReviewWritesOrRemoveGuardPermissions(t *testing.T) {
	data, err := Candidate("demo", "src")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"review-write", "missing-deny", "unresolved-path", "auto-compaction", "background-compaction"} {
		var c map[string]any
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatal(err)
		}
		a := c["agents"].(map[string]any)["sdd-review"].(map[string]any)
		switch kind {
		case "review-write":
			a["permissions"] = append(a["permissions"].([]any), map[string]any{"action": "edit", "resource": "*", "effect": "allow"})
		case "missing-deny":
			a["permissions"] = []any{}
		case "unresolved-path":
			a["system"] = "unresolved __FEATURE__"
		case "auto-compaction":
			c["compaction"].(map[string]any)["auto"] = true
		case "background-compaction":
			c["experimental"].(map[string]any)["background_compaction"] = true
		}
		bad, _ := json.Marshal(c)
		if _, err := Inspect(bad, nil); err == nil {
			t.Errorf("accepted %s", kind)
		}
	}
}

func TestPreparedBackupSurvivesFailureAndInterveningWriter(t *testing.T) {
	for _, change := range []bool{false, true} {
		t.Run(map[bool]string{false: "failure", true: "intervening-edit"}[change], func(t *testing.T) {
			root, candidate, backups, _ := fixture(t)
			dir := filepath.Join(root, ".opencode")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(dir, "opencode.jsonc")
			old := []byte(`{"unrelated":"old"}`)
			if err := os.WriteFile(dest, old, 0600); err != nil {
				t.Fatal(err)
			}
			p, err := Preview(root, candidate)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := apply(root, candidate, backups, p.Original, p.Candidate, func() error {
				if change {
					return os.WriteFile(dest, []byte(`{"unrelated":"user edit"}`), 0600)
				}
				return os.ErrPermission
			})
			if err == nil || tx.Backup == "" {
				t.Fatal("failed transaction missing retained backup", err)
			}
			backup, err := os.ReadFile(filepath.Join(tx.Backup, "original"))
			if err != nil || !bytes.Equal(backup, old) {
				t.Fatal("old bytes not backed up before replacement")
			}
			current, _ := os.ReadFile(dest)
			if change {
				if !bytes.Contains(current, []byte("user edit")) {
					t.Fatal("discarded user edit")
				}
				if _, err := Rollback(root, tx.Backup); err == nil {
					t.Fatal("rollback discarded user edit")
				}
			} else if !bytes.Equal(current, old) {
				t.Fatal("failed preparation changed destination")
			}
		})
	}
}

func TestReadInputRefusesCredentialStoreNames(t *testing.T) {
	for _, name := range []string{"auth.json", "credentials.json", "opencode.db", "opencode.db-wal", ".env.local", "private.pem"} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte("SECRET-CANARY"), 0600); err != nil {
			t.Fatal(err)
		}
		if data, err := ReadInput(path); err == nil || len(data) != 0 || strings.Contains(err.Error(), "SECRET-CANARY") {
			t.Errorf("credential path admitted: %s", name)
		}
	}
}

func TestJSONCValidationPreservesStringMeaningAndRejectsAmbiguity(t *testing.T) {
	raw := []byte("\xef\xbb\xbf// note\n{\"text\":\"a,} // literal\",\"url\":\"https://example.invalid\",\"arr\":[1,2,], /* note */}\n")
	m, err := parseConfig(raw)
	if err != nil || m["text"] != "a,} // literal" || m["url"] != "https://example.invalid" || len(m["arr"].([]any)) != 2 {
		t.Fatal("changed JSONC values", err)
	}
	for _, bad := range []string{`{"api\u005fkey":"SECRET-CANARY"}`, `{"nested":{"x":1,"x":2}}`, `{/* missing close`, strings.Repeat(`{"x":`, 40) + `0` + strings.Repeat(`}`, 40)} {
		if _, err := parseConfig([]byte(bad)); err == nil || strings.Contains(err.Error(), "SECRET-CANARY") {
			t.Errorf("accepted/leaked ambiguous input: %v", err)
		}
	}
}

func TestRollbackRejectsCorruptedBackupAndForeignManifest(t *testing.T) {
	root, candidate, backups, _ := fixture(t)
	dir := filepath.Join(root, ".opencode")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(`{"old":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := Preview(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := Apply(root, candidate, backups, p.Original, p.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tx.Backup, "original"), []byte(`{"tampered":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Rollback(root, tx.Backup); err == nil {
		t.Fatal("accepted corrupted backup")
	}
	if _, err := Rollback(t.TempDir(), tx.Backup); err == nil {
		t.Fatal("accepted foreign workspace")
	}
}
