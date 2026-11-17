package appops

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

func TestMCPPlanRequestValidationAndImmutability(t *testing.T) {
	// Empty home is rejected.
	if _, err := NewMCPPlanRequest("", "", nil, nil); err == nil {
		t.Fatal("expected error for empty home")
	}

	tempDir := t.TempDir()
	relHome := filepath.Base(tempDir)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Relative home and config path are absolutized.
	relConfig := filepath.Join(relHome, "custom-opencode.json")
	req, err := NewMCPPlanRequest(relHome, relConfig, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(req.Home()) {
		t.Fatalf("home was not absolutized: %q", req.Home())
	}
	if !filepath.IsAbs(req.ConfigPath()) {
		t.Fatalf("config path was not absolutized: %q", req.ConfigPath())
	}
	if req.Home() != filepath.Join(cwd, relHome) {
		t.Fatalf("home resolved to %q, want %q", req.Home(), filepath.Join(cwd, relHome))
	}

	// Empty config path remains empty.
	reqEmptyConfig, err := NewMCPPlanRequest(tempDir, "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqEmptyConfig.ConfigPath() != "" {
		t.Fatalf("expected empty config path, got %q", reqEmptyConfig.ConfigPath())
	}

	// Selected slice immutability.
	origSelected := []string{"codegraph", "context7"}
	origAuth := map[string]string{"context7": "oauth"}
	reqCopy, err := NewMCPPlanRequest(tempDir, "", origSelected, origAuth)
	if err != nil {
		t.Fatal(err)
	}

	// Mutate original slice and map after construction.
	origSelected[0] = "mutated"
	origAuth["context7"] = "mutated"
	origAuth["extra"] = "injected"

	if reqCopy.Selected()[0] != "codegraph" {
		t.Fatalf("Selected was mutated via original slice: got %q", reqCopy.Selected()[0])
	}
	if reqCopy.AuthChoices()["context7"] != "oauth" {
		t.Fatalf("AuthChoices was mutated via original map: got %q", reqCopy.AuthChoices()["context7"])
	}
	if _, ok := reqCopy.AuthChoices()["extra"]; ok {
		t.Fatal("AuthChoices had extra key injected from original map")
	}

	// Mutate slice returned by accessor.
	sel := reqCopy.Selected()
	sel[0] = "mutated-accessor"
	if reqCopy.Selected()[0] != "codegraph" {
		t.Fatalf("Selected was mutated via accessor slice: got %q", reqCopy.Selected()[0])
	}

	// Mutate map returned by accessor.
	auth := reqCopy.AuthChoices()
	auth["context7"] = "mutated-accessor"
	if reqCopy.AuthChoices()["context7"] != "oauth" {
		t.Fatalf("AuthChoices was mutated via accessor map: got %q", reqCopy.AuthChoices()["context7"])
	}
}

func TestMCPApplyRequestValidationAndImmutability(t *testing.T) {
	tempDir := t.TempDir()
	validPlan := &skillsync.MCPPlan{
		TargetPath: filepath.Join(tempDir, "opencode.json"),
		Format:     "json",
		Schema:     "v2",
		BaseDigest: "1111111111111111111111111111111111111111111111111111111111111111",
		PlanDigest: "2222222222222222222222222222222222222222222222222222222222222222",
		Selected:   []string{"codegraph"},
		AuthChoices: map[string]string{
			"context7": "oauth",
		},
		Entries: []skillsync.MCPPlanEntry{
			{Name: "codegraph", Op: "add", Reason: "managed server add"},
		},
	}
	token := mcpPlanPreviewToken(validPlan)

	// Empty home rejected.
	if _, err := NewMCPApplyRequest("", "", validPlan, token, validPlan.BaseDigest, ""); err == nil {
		t.Fatal("expected error for empty home")
	}

	// Nil plan rejected.
	if _, err := NewMCPApplyRequest(tempDir, "", nil, token, validPlan.BaseDigest, ""); err == nil {
		t.Fatal("expected error for nil plan")
	}

	// Empty token rejected.
	if _, err := NewMCPApplyRequest(tempDir, "", validPlan, "", validPlan.BaseDigest, ""); err == nil {
		t.Fatal("expected error for empty token")
	}

	// Empty expect digest rejected.
	if _, err := NewMCPApplyRequest(tempDir, "", validPlan, token, "", ""); err == nil {
		t.Fatal("expected error for empty expect digest")
	}

	// Mismatched expect digest rejected.
	if _, err := NewMCPApplyRequest(tempDir, "", validPlan, token, "different-digest", ""); err == nil {
		t.Fatal("expected error for mismatched expect digest")
	}

	// Relative openCodeBin absolutized.
	req, err := NewMCPApplyRequest(tempDir, "", validPlan, token, validPlan.BaseDigest, "bin/opencode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(req.OpenCodeBin()) {
		t.Fatalf("openCodeBin was not absolutized: %q", req.OpenCodeBin())
	}

	// Plan clone immutability.
	mutablePlan := validPlan.Clone()
	reqImmutable, err := NewMCPApplyRequest(tempDir, "", mutablePlan, token, mutablePlan.BaseDigest, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate mutablePlan after request construction.
	mutablePlan.Selected[0] = "mutated-after-req"
	mutablePlan.AuthChoices["context7"] = "mutated-after-req"
	mutablePlan.TargetPath = "mutated-path"

	if reqImmutable.Plan().Selected[0] != "codegraph" {
		t.Fatalf("Plan.Selected was mutated through source plan: %q", reqImmutable.Plan().Selected[0])
	}
	if reqImmutable.Plan().AuthChoices["context7"] != "oauth" {
		t.Fatalf("Plan.AuthChoices was mutated through source plan: %q", reqImmutable.Plan().AuthChoices["context7"])
	}
	if reqImmutable.Plan().TargetPath != validPlan.TargetPath {
		t.Fatalf("Plan.TargetPath was mutated through source plan: %q", reqImmutable.Plan().TargetPath)
	}

	// Mutate plan returned by accessor.
	planFromAccessor := reqImmutable.Plan()
	planFromAccessor.Selected[0] = "mutated-accessor"
	if reqImmutable.Plan().Selected[0] != "codegraph" {
		t.Fatalf("Plan.Selected was mutated through accessor: %q", reqImmutable.Plan().Selected[0])
	}
}

func TestMCPRestorePreviewRequestValidation(t *testing.T) {
	tempDir := t.TempDir()
	validBackupID := "20261113T000000.000000000Z"

	// Empty home rejected.
	if _, err := NewMCPRestorePreviewRequest("", "", validBackupID); err == nil {
		t.Fatal("expected error for empty home")
	}

	// Empty backupID rejected.
	if _, err := NewMCPRestorePreviewRequest(tempDir, "", ""); err == nil {
		t.Fatal("expected error for empty backupID")
	}

	// Malformed backupID rejected.
	if _, err := NewMCPRestorePreviewRequest(tempDir, "", "invalid-id"); err == nil {
		t.Fatal("expected error for malformed backupID")
	}

	// Valid backupID accepted and paths absolutized.
	req, err := NewMCPRestorePreviewRequest(tempDir, "relative-config.json", validBackupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(req.Home()) {
		t.Fatalf("home was not absolutized: %q", req.Home())
	}
	if !filepath.IsAbs(req.ConfigPath()) {
		t.Fatalf("configPath was not absolutized: %q", req.ConfigPath())
	}
	if req.BackupID() != validBackupID {
		t.Fatalf("backupID = %q, want %q", req.BackupID(), validBackupID)
	}
}

func TestMCPRestoreRequestValidation(t *testing.T) {
	tempDir := t.TempDir()
	validBackupID := "20261113T000000.000000000Z"
	token := "some-token"
	expectDigest := "1111111111111111111111111111111111111111111111111111111111111111"

	// Empty home rejected.
	if _, err := NewMCPRestoreRequest("", "", validBackupID, token, expectDigest); err == nil {
		t.Fatal("expected error for empty home")
	}

	// Empty backupID rejected.
	if _, err := NewMCPRestoreRequest(tempDir, "", "", token, expectDigest); err == nil {
		t.Fatal("expected error for empty backupID")
	}

	// Malformed backupID rejected.
	if _, err := NewMCPRestoreRequest(tempDir, "", "bad-id", token, expectDigest); err == nil {
		t.Fatal("expected error for malformed backupID")
	}

	// Empty token rejected.
	if _, err := NewMCPRestoreRequest(tempDir, "", validBackupID, "", expectDigest); err == nil {
		t.Fatal("expected error for empty token")
	}

	// Empty expectDigest rejected.
	if _, err := NewMCPRestoreRequest(tempDir, "", validBackupID, token, ""); err == nil {
		t.Fatal("expected error for empty expectDigest")
	}

	// Valid request accepted.
	req, err := NewMCPRestoreRequest(tempDir, "", validBackupID, token, expectDigest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Home() != tempDir {
		t.Fatalf("home = %q, want %q", req.Home(), tempDir)
	}
	if req.BackupID() != validBackupID {
		t.Fatalf("backupID = %q, want %q", req.BackupID(), validBackupID)
	}
	if req.Token() != token {
		t.Fatalf("token = %q, want %q", req.Token(), token)
	}
	if req.ExpectDigest() != expectDigest {
		t.Fatalf("expectDigest = %q, want %q", req.ExpectDigest(), expectDigest)
	}
}

func TestMCPRestoreListValidationAndBehavior(t *testing.T) {
	// Empty home rejected.
	if _, err := NewMCPRestoreListRequest(""); err == nil {
		t.Fatal("expected error for empty home")
	}

	tempDir := t.TempDir()
	req, err := NewMCPRestoreListRequest(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Home() != tempDir {
		t.Fatalf("Home() = %q, want %q", req.Home(), tempDir)
	}

	// Non-existent backup dir returns empty list without error.
	backups, err := MCPRestoreList(req)
	if err != nil {
		t.Fatalf("MCPRestoreList: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected 0 backups, got %d", len(backups))
	}

	// Populate backup directory with valid and invalid entries.
	backupBase := filepath.Join(tempDir, ".agentic-sdd", "mcp", "backups")
	idOlder := "20261113T000000.000000000Z"
	idNewer := "20261113T010000.000000000Z"
	idInvalid := "invalid-id"
	for _, id := range []string{idOlder, idNewer, idInvalid} {
		if err := os.MkdirAll(filepath.Join(backupBase, id), 0700); err != nil {
			t.Fatal(err)
		}
	}
	// Also a regular file.
	if err := os.WriteFile(filepath.Join(backupBase, "not-a-dir"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	backups, err = MCPRestoreList(req)
	if err != nil {
		t.Fatalf("MCPRestoreList: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 valid backups, got %d", len(backups))
	}
	// Sorted newest first: idNewer, then idOlder.
	if backups[0].ID != idNewer {
		t.Fatalf("backups[0].ID = %q, want %q", backups[0].ID, idNewer)
	}
	if backups[1].ID != idOlder {
		t.Fatalf("backups[1].ID = %q, want %q", backups[1].ID, idOlder)
	}
}

func TestMCPPlanPreviewNoWrite(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", "{\n  \"mcp\": {\n    \"servers\": {}\n  }\n}\n")
	origBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	req, err := NewMCPPlanRequest(fixture.home, configPath, []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("NewMCPPlanRequest: %v", err)
	}

	res, err := MCPPlanPreview(req)
	if err != nil {
		t.Fatalf("MCPPlanPreview: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if res.Token == "" {
		t.Fatal("expected non-empty preview token")
	}

	// Preview must perform no writes: no .agentic-sdd created and target bytes identical.
	fixture.assertNoMCPWrite(t)
	assertMCPConfigUnchanged(t, configPath, string(origBytes))
}

func TestMCPApplyAndRestoreCallEngineOnFixtures(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", "{\n  \"mcp\": {\n    \"servers\": {}\n  }\n}\n")
	origBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create fake opencode binary returning 2.1.0 for the version probe.
	binDir := t.TempDir()
	fakeBin := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\necho '2.1.0'\n"
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	planReq, err := NewMCPPlanRequest(fixture.home, configPath, []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("NewMCPPlanRequest: %v", err)
	}
	planRes, err := MCPPlanPreview(planReq)
	if err != nil {
		t.Fatalf("MCPPlanPreview: %v", err)
	}

	applyReq, err := NewMCPApplyRequest(
		fixture.home,
		configPath,
		planRes.Plan,
		planRes.Token,
		planRes.Plan.BaseDigest,
		fakeBin,
	)
	if err != nil {
		t.Fatalf("NewMCPApplyRequest: %v", err)
	}

	applyRes, err := MCPApply(applyReq)
	if err != nil {
		t.Fatalf("MCPApply: %v", err)
	}
	if applyRes.Result == nil {
		t.Fatal("expected non-nil apply result")
	}
	if applyRes.Result.InstalledDigest == "" {
		t.Fatal("expected non-empty installed digest")
	}
	if applyRes.Result.BackupID == "" {
		t.Fatal("expected non-empty backup ID")
	}

	// Config file was modified by apply.
	newBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(newBytes) == string(origBytes) {
		t.Fatal("config file was not updated by apply")
	}

	// Restore preview.
	restorePrevReq, err := NewMCPRestorePreviewRequest(fixture.home, configPath, applyRes.Result.BackupID)
	if err != nil {
		t.Fatalf("NewMCPRestorePreviewRequest: %v", err)
	}
	restorePrevRes, err := MCPRestorePreview(restorePrevReq)
	if err != nil {
		t.Fatalf("MCPRestorePreview: %v", err)
	}
	if restorePrevRes.Preview == nil {
		t.Fatal("expected non-nil restore preview")
	}
	if restorePrevRes.Token == "" {
		t.Fatal("expected non-empty restore preview token")
	}

	// Restore apply.
	restoreReq, err := NewMCPRestoreRequest(
		fixture.home,
		configPath,
		applyRes.Result.BackupID,
		restorePrevRes.Token,
		restorePrevRes.Preview.BaseDigest,
	)
	if err != nil {
		t.Fatalf("NewMCPRestoreRequest: %v", err)
	}

	restoreRes, err := MCPRestore(restoreReq)
	if err != nil {
		t.Fatalf("MCPRestore: %v", err)
	}
	if restoreRes.Result == nil {
		t.Fatal("expected non-nil restore result")
	}

	// After restore of the added server, config matches the original state.
	restoredBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restoredBytes) != string(origBytes) {
		t.Fatalf("config file was not restored:\ngot:\n%s\nwant:\n%s", restoredBytes, origBytes)
	}
}

func TestMCPPlanPreviewToken(t *testing.T) {
	// Nil plan returns empty string.
	if token := mcpPlanPreviewToken(nil); token != "" {
		t.Fatalf("expected empty token for nil plan, got %q", token)
	}

	basePlan := &skillsync.MCPPlan{
		TargetPath: "/home/user/.config/opencode/opencode.json",
		Format:     "json",
		Schema:     "v2",
		BaseDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PlanDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Selected:   []string{"codegraph", "context7"},
		AuthChoices: map[string]string{
			"context7": "oauth",
			"exa":      "keyless",
		},
		Entries: []skillsync.MCPPlanEntry{
			{Name: "codegraph", Op: "add", Reason: "new server", NewHash: "h1"},
			{Name: "context7", Op: "change", Reason: "auth change", OldHash: "h2", NewHash: "h3"},
		},
		RuntimeVersion:  "1.0.0",
		RuntimeEvidence: "local probe",
		NoOp:            false,
	}

	token1 := mcpPlanPreviewToken(basePlan)
	token2 := mcpPlanPreviewToken(basePlan)
	if token1 == "" {
		t.Fatal("expected non-empty token")
	}
	if token1 != token2 {
		t.Fatalf("token is not deterministic: %q != %q", token1, token2)
	}
	if len(token1) != 64 {
		t.Fatalf("expected SHA-256 hex string (length 64), got length %d (%q)", len(token1), token1)
	}

	// Deterministic regardless of map iteration order.
	planReorderedMap := basePlan.Clone()
	planReorderedMap.AuthChoices = map[string]string{
		"exa":      "keyless",
		"context7": "oauth",
	}
	if mcpPlanPreviewToken(planReorderedMap) != token1 {
		t.Fatal("token depended on map iteration order")
	}

	// Any field change produces a different token.
	checkDifferent := func(desc string, mutate func(*skillsync.MCPPlan)) {
		t.Helper()
		p := basePlan.Clone()
		mutate(p)
		if mcpPlanPreviewToken(p) == token1 {
			t.Fatalf("token did not change when %s was mutated", desc)
		}
	}

	checkDifferent("TargetPath", func(p *skillsync.MCPPlan) { p.TargetPath += "c" })
	checkDifferent("Format", func(p *skillsync.MCPPlan) { p.Format = "jsonc" })
	checkDifferent("Schema", func(p *skillsync.MCPPlan) { p.Schema = "v1" })
	checkDifferent("BaseDigest", func(p *skillsync.MCPPlan) { p.BaseDigest = "cccc" })
	checkDifferent("PlanDigest", func(p *skillsync.MCPPlan) { p.PlanDigest = "dddd" })
	checkDifferent("Selected", func(p *skillsync.MCPPlan) { p.Selected = []string{"codegraph"} })
	checkDifferent("AuthChoices", func(p *skillsync.MCPPlan) { p.AuthChoices["context7"] = "api-key" })
	checkDifferent("Entries", func(p *skillsync.MCPPlan) { p.Entries[0].Op = "leave" })
	checkDifferent("RuntimeVersion", func(p *skillsync.MCPPlan) { p.RuntimeVersion = "2.0.0" })
	checkDifferent("NoOp", func(p *skillsync.MCPPlan) { p.NoOp = true })
}

func TestMCPRestorePreviewToken(t *testing.T) {
	// Nil preview returns empty string.
	if token := mcpRestorePreviewToken(nil); token != "" {
		t.Fatalf("expected empty token for nil preview, got %q", token)
	}

	basePreview := &skillsync.MCPRestorePreview{
		BackupID:   "20261113T000000.000000000Z",
		TargetPath: "/home/user/.config/opencode/opencode.json",
		BaseDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Entries: []skillsync.MCPPlanEntry{
			{Name: "codegraph", Op: "restore", Reason: "managed server restore", OldHash: "h1", NewHash: "h2"},
		},
		Conflicts: []string{},
	}

	token1 := mcpRestorePreviewToken(basePreview)
	token2 := mcpRestorePreviewToken(basePreview)
	if token1 == "" {
		t.Fatal("expected non-empty token")
	}
	if token1 != token2 {
		t.Fatalf("token is not deterministic: %q != %q", token1, token2)
	}

	// Field mutations change token.
	pChanged := *basePreview
	pChanged.BackupID = "20261113T000001.000000000Z"
	if mcpRestorePreviewToken(&pChanged) == token1 {
		t.Fatal("token did not change when BackupID was changed")
	}

	pChangedTarget := *basePreview
	pChangedTarget.TargetPath = "/different/path"
	if mcpRestorePreviewToken(&pChangedTarget) == token1 {
		t.Fatal("token did not change when TargetPath was changed")
	}

	pChangedDigest := *basePreview
	pChangedDigest.BaseDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if mcpRestorePreviewToken(&pChangedDigest) == token1 {
		t.Fatal("token did not change when BaseDigest was changed")
	}

	pChangedConflicts := *basePreview
	pChangedConflicts.Conflicts = []string{"context7"}
	if mcpRestorePreviewToken(&pChangedConflicts) == token1 {
		t.Fatal("token did not change when Conflicts was changed")
	}
}

func TestMCPApplyTokenAndDigestRevalidation(t *testing.T) {
	tempDir := t.TempDir()
	validPlan := &skillsync.MCPPlan{
		TargetPath: filepath.Join(tempDir, "opencode.json"),
		Format:     "json",
		Schema:     "v2",
		BaseDigest: "1111111111111111111111111111111111111111111111111111111111111111",
		PlanDigest: "2222222222222222222222222222222222222222222222222222222222222222",
		Selected:   []string{"codegraph"},
	}
	correctToken := mcpPlanPreviewToken(validPlan)

	req, err := NewMCPApplyRequest(tempDir, "", validPlan, correctToken, validPlan.BaseDigest, "")
	if err != nil {
		t.Fatal(err)
	}

	// Empty token in apply request returns ErrNeedsPreview.
	reqEmptyToken := req
	reqEmptyToken.token = ""
	if _, err := MCPApply(reqEmptyToken); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPApply with empty token error = %v, want ErrNeedsPreview", err)
	}

	// Mismatched token in apply request returns ErrNeedsPreview.
	reqWrongToken := req
	reqWrongToken.token = strings.Repeat("0", 64)
	if _, err := MCPApply(reqWrongToken); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPApply with mismatched token error = %v, want ErrNeedsPreview", err)
	}

	// Empty expect digest returns ErrNeedsPreview.
	reqEmptyDigest := req
	reqEmptyDigest.expectDigest = ""
	if _, err := MCPApply(reqEmptyDigest); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPApply with empty expect digest error = %v, want ErrNeedsPreview", err)
	}

	// Mismatched expect digest returns ErrNeedsPreview.
	reqWrongDigest := req
	reqWrongDigest.expectDigest = strings.Repeat("9", 64)
	if _, err := MCPApply(reqWrongDigest); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPApply with mismatched expect digest error = %v, want ErrNeedsPreview", err)
	}
}

func TestMCPRestoreTokenAndDigestRevalidation(t *testing.T) {
	tempDir := t.TempDir()
	validBackupID := "20261113T000000.000000000Z"
	token := "valid-token"
	expectDigest := "1111111111111111111111111111111111111111111111111111111111111111"

	req, err := NewMCPRestoreRequest(tempDir, "", validBackupID, token, expectDigest)
	if err != nil {
		t.Fatal(err)
	}

	// Empty token returns ErrNeedsPreview.
	reqEmptyToken := req
	reqEmptyToken.token = ""
	if _, err := MCPRestore(reqEmptyToken); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPRestore with empty token error = %v, want ErrNeedsPreview", err)
	}

	// Empty expect digest returns ErrNeedsPreview.
	reqEmptyDigest := req
	reqEmptyDigest.expectDigest = ""
	if _, err := MCPRestore(reqEmptyDigest); !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPRestore with empty expect digest error = %v, want ErrNeedsPreview", err)
	}
}

func TestMCPWriteSourceIsOfflineOnly(t *testing.T) {
	source, err := os.ReadFile("mcp_write.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{`"os/exec"`, `"net/http"`, `"net"`, "os.UserHomeDir", "os.Getenv"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("internal/appops/mcp_write.go imports or calls %s; the MCP-B seam must stay offline", forbidden)
		}
	}
	// Verify no adopt or replace fields in write request structs.
	for _, forbiddenWord := range []string{"Adopt", "adopt", "Replace", "replace"} {
		if strings.Contains(text, forbiddenWord) {
			t.Fatalf("internal/appops/mcp_write.go contains %q; adopt/replace are deferred", forbiddenWord)
		}
	}
}

// mcpTestOpenCodeRuntime writes a fake OpenCode binary that reports the runtime
// version the apply probe accepts, so a fixture write can reach the transaction
// phase without a real installation.
func mcpTestOpenCodeRuntime(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "opencode")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho '2.1.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// corruptMCPOwnershipManifest replaces the private ownership manifest with
// invalid JSON after a preview. The next write fails at the post-rename
// manifest step, so the engine has to roll the target entries back.
func corruptMCPOwnershipManifest(t *testing.T, home string) {
	t.Helper()
	dir := filepath.Join(home, ".agentic-sdd", "mcp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestMCPApplyFailureKeepsStructuredResult proves a failed apply still returns
// the engine's record. The backup identity, the target and the completed
// rollback are facts the caller has to be able to show, so appops must not
// replace them with a bare error; only the error says the write failed.
func TestMCPApplyFailureKeepsStructuredResult(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", "{\n  \"mcp\": {\n    \"servers\": {}\n  }\n}\n")
	bin := mcpTestOpenCodeRuntime(t)

	planReq, err := NewMCPPlanRequest(fixture.home, configPath, []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("NewMCPPlanRequest: %v", err)
	}
	planRes, err := MCPPlanPreview(planReq)
	if err != nil {
		t.Fatalf("MCPPlanPreview: %v", err)
	}
	baseDigest := planRes.Plan.BaseDigest

	corruptMCPOwnershipManifest(t, fixture.home)

	applyReq, err := NewMCPApplyRequest(fixture.home, configPath, planRes.Plan, planRes.Token, baseDigest, bin)
	if err != nil {
		t.Fatalf("NewMCPApplyRequest: %v", err)
	}
	res, err := MCPApply(applyReq)
	if err == nil {
		t.Fatal("apply with a corrupt ownership manifest was reported as success")
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("error = %q, want an explicit rolled-back report", err.Error())
	}
	if res.Result == nil {
		t.Fatal("failed apply dropped the engine result; the caller cannot report the rollback")
	}
	if res.Result.TargetPath != configPath {
		t.Fatalf("result target = %q, want %q", res.Result.TargetPath, configPath)
	}
	if res.Result.BackupID == "" || res.Result.BackupPath == "" {
		t.Fatalf("failed apply lost the backup identity: %+v", res.Result)
	}
	if _, statErr := os.Stat(res.Result.BackupPath); statErr != nil {
		t.Fatalf("reported backup path is not present: %v", statErr)
	}
	if !res.Result.Rollback.Attempted || !res.Result.Rollback.Complete {
		t.Fatalf("rollback facts lost: %+v", res.Result.Rollback)
	}
	if res.Result.Rollback.Error != "" {
		t.Fatalf("complete rollback carried an error: %q", res.Result.Rollback.Error)
	}
	if res.Result.InstalledDigest != baseDigest {
		t.Fatalf("rolled-back result reports digest %q, want the base digest %q", res.Result.InstalledDigest, baseDigest)
	}
	if len(res.Result.Applied) != 0 {
		t.Fatalf("rolled-back apply claimed applied changes: %v", res.Result.Applied)
	}

	// The rollback is real: the managed entry is gone from the target again.
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "codegraph") {
		t.Fatalf("rollback left the managed entry behind:\n%s", after)
	}
	root := map[string]any{}
	if err := json.Unmarshal(after, &root); err != nil {
		t.Fatalf("target after rollback is not JSON: %v", err)
	}
	mcpSection, ok := root["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("target after rollback lost the mcp section:\n%s", after)
	}
	servers, ok := mcpSection["servers"].(map[string]any)
	if !ok || len(servers) != 0 {
		t.Fatalf("servers after rollback = %v, want the original empty object", mcpSection["servers"])
	}
}

// TestMCPRestoreFailureKeepsStructuredResult proves the restore seam reports the
// same partial facts as apply: a failed restore returns its own safety backup
// and the rollback state along with the error.
func TestMCPRestoreFailureKeepsStructuredResult(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", "{\n  \"mcp\": {\n    \"servers\": {}\n  }\n}\n")
	bin := mcpTestOpenCodeRuntime(t)

	planReq, err := NewMCPPlanRequest(fixture.home, configPath, []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("NewMCPPlanRequest: %v", err)
	}
	planRes, err := MCPPlanPreview(planReq)
	if err != nil {
		t.Fatalf("MCPPlanPreview: %v", err)
	}
	applyReq, err := NewMCPApplyRequest(fixture.home, configPath, planRes.Plan, planRes.Token, planRes.Plan.BaseDigest, bin)
	if err != nil {
		t.Fatalf("NewMCPApplyRequest: %v", err)
	}
	applyRes, err := MCPApply(applyReq)
	if err != nil {
		t.Fatalf("MCPApply: %v", err)
	}
	appliedBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	previewReq, err := NewMCPRestorePreviewRequest(fixture.home, configPath, applyRes.Result.BackupID)
	if err != nil {
		t.Fatalf("NewMCPRestorePreviewRequest: %v", err)
	}
	previewRes, err := MCPRestorePreview(previewReq)
	if err != nil {
		t.Fatalf("MCPRestorePreview: %v", err)
	}

	corruptMCPOwnershipManifest(t, fixture.home)

	restoreReq, err := NewMCPRestoreRequest(
		fixture.home, configPath, applyRes.Result.BackupID, previewRes.Token, previewRes.Preview.BaseDigest,
	)
	if err != nil {
		t.Fatalf("NewMCPRestoreRequest: %v", err)
	}
	res, err := MCPRestore(restoreReq)
	if err == nil {
		t.Fatal("restore with a corrupt ownership manifest was reported as success")
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("error = %q, want an explicit rolled-back report", err.Error())
	}
	if res.Result == nil {
		t.Fatal("failed restore dropped the engine result; the caller cannot report the rollback")
	}
	if res.Result.TargetPath != configPath {
		t.Fatalf("result target = %q, want %q", res.Result.TargetPath, configPath)
	}
	if res.Result.BackupID == "" || res.Result.BackupID == applyRes.Result.BackupID {
		t.Fatalf("failed restore backup id = %q, want its own safety backup", res.Result.BackupID)
	}
	if _, statErr := os.Stat(res.Result.BackupPath); statErr != nil {
		t.Fatalf("reported backup path is not present: %v", statErr)
	}
	if !res.Result.Rollback.Attempted || !res.Result.Rollback.Complete {
		t.Fatalf("rollback facts lost: %+v", res.Result.Rollback)
	}
	if len(res.Result.Applied) != 0 {
		t.Fatalf("rolled-back restore claimed applied changes: %v", res.Result.Applied)
	}

	// The rollback reproduced the applied bytes exactly.
	assertMCPConfigUnchanged(t, configPath, string(appliedBytes))
}

// TestMCPRefusalsReportNoEngineResult keeps the refusal path distinct from a
// failed transaction: a request appops rejects before the engine runs carries
// no facts, so a caller cannot present it as a recorded write.
func TestMCPRefusalsReportNoEngineResult(t *testing.T) {
	tempDir := t.TempDir()
	plan := &skillsync.MCPPlan{
		TargetPath: filepath.Join(tempDir, "opencode.json"),
		Format:     "json",
		Schema:     "v2",
		BaseDigest: "1111111111111111111111111111111111111111111111111111111111111111",
		PlanDigest: "2222222222222222222222222222222222222222222222222222222222222222",
		Selected:   []string{"codegraph"},
	}
	req, err := NewMCPApplyRequest(tempDir, "", plan, mcpPlanPreviewToken(plan), plan.BaseDigest, "")
	if err != nil {
		t.Fatal(err)
	}
	// A cleared token is a refusal: appops never calls the engine, so it has no
	// facts to report.
	req.token = ""
	applyRes, err := MCPApply(req)
	if !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPApply refusal error = %v, want ErrNeedsPreview", err)
	}
	if applyRes.Result != nil {
		t.Fatalf("refused apply reported engine facts: %+v", applyRes.Result)
	}

	restoreReq, err := NewMCPRestoreRequest(tempDir, "", "20261113T000000.000000000Z", "token", plan.BaseDigest)
	if err != nil {
		t.Fatal(err)
	}
	restoreReq.token = ""
	restoreRes, err := MCPRestore(restoreReq)
	if !errors.Is(err, ErrNeedsPreview) {
		t.Fatalf("MCPRestore refusal error = %v, want ErrNeedsPreview", err)
	}
	if restoreRes.Result != nil {
		t.Fatalf("refused restore reported engine facts: %+v", restoreRes.Result)
	}
}
