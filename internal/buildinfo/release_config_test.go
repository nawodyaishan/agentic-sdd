package buildinfo_test

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// The release config is static text here on purpose: GoReleaser and lipo are
// not available in every development environment, so these assertions guard the
// reviewed shape without pretending a snapshot ran. The real archive/universal/
// cask proof remains the nonpublishing snapshot task.
func TestGoReleaserPairContract(t *testing.T) {
	source, err := os.ReadFile("../../.goreleaser.yml")
	if err != nil {
		t.Fatalf("read .goreleaser.yml: %v", err)
	}
	config := string(source)

	if !strings.Contains(config, "version: 2") {
		t.Fatal(".goreleaser.yml must declare GoReleaser schema version 2")
	}
	if strings.Contains(config, "allow_different_binary_count") {
		t.Fatal(".goreleaser.yml must not use the allow_different_binary_count escape hatch")
	}

	// Four applications from two existing sources: canonical and legacy CLI
	// share ./cmd/agentic-sdd; canonical and legacy TUI share the nested
	// ./cmd/tui-companion. Installed names are the universal/archive names.
	builds := topLevelBlocks(t, config, "builds")
	byBuildID := map[string]yamlBlock{}
	for _, block := range builds {
		byBuildID[block.scalar("id")] = block
	}
	if len(builds) != 4 {
		t.Fatalf("builds has %d entries, want the four canonical/legacy CLI and TUI builds (found %v)", len(builds), sortedKeys(byBuildID))
	}
	for _, id := range buildIDs() {
		block, ok := byBuildID[id]
		if !ok {
			t.Fatalf("missing explicit build ID %q (found %v)", id, sortedKeys(byBuildID))
		}
		if block.scalar("binary") != universalName(id) {
			t.Fatalf("build %q binary = %q, want %q so the archive member is the installed command name", id, block.scalar("binary"), universalName(id))
		}
		if !strings.Contains(block.text(), "-X agentic-sdd/internal/buildinfo.Revision={{ .Version }}") {
			t.Fatalf("build %q does not inject buildinfo.Revision", id)
		}
		if !strings.Contains(block.text(), "-X agentic-sdd/internal/buildinfo.Commit={{ .FullCommit }}") {
			t.Fatalf("build %q does not inject the full buildinfo.Commit", id)
		}
		if !strings.Contains(block.text(), "SDD_SOURCE_DIGEST") || !strings.Contains(block.text(), ".FullCommit") {
			t.Fatalf("build %q does not inject a content digest with a clean-commit fallback", id)
		}
		if !strings.Contains(block.text(), "-X agentic-sdd/internal/buildinfo.DistributionMarker=generic") {
			t.Fatalf("build %q must carry the generic distribution marker", id)
		}
		if strings.Contains(block.text(), "DistributionMarker=homebrew") {
			t.Fatalf("build %q must not carry a Homebrew distribution marker", id)
		}
	}

	// The Homebrew marker no longer changes bare-launch behavior, so no
	// Homebrew-specific build or duplicate CLI artifact is retained.
	if got := strings.Count(config, "DistributionMarker=homebrew"); got != 0 {
		t.Fatalf(".goreleaser.yml carries the Homebrew marker %d times, want 0", got)
	}
	if got := strings.Count(config, "DistributionMarker=generic"); got != 4 {
		t.Fatalf(".goreleaser.yml carries the generic marker %d times, want one per build", got)
	}

	core := byBuildID["cli-core"]
	if got := core.scalar("main"); got != "./cmd/agentic-sdd" {
		t.Fatalf("cli-core main = %q, want ./cmd/agentic-sdd", got)
	}
	if strings.Contains(core.text(), "legacyTarget") || strings.Contains(core.text(), "legacyName") {
		t.Fatal("cli-core must not carry the legacy compatibility identity")
	}

	legacyCLI := byBuildID["cli-legacy"]
	if got := legacyCLI.scalar("main"); got != "./cmd/agentic-sdd" {
		t.Fatalf("cli-legacy main = %q, want the shared ./cmd/agentic-sdd source", got)
	}
	if got := legacyCLI.scalar("binary"); got != "agentic-sdd" {
		t.Fatalf("cli-legacy binary = %q, want agentic-sdd", got)
	}
	for _, want := range []string{
		"-X main.legacyTarget=asdx-core",
		"-X main.legacyName=agentic-sdd",
	} {
		if !strings.Contains(legacyCLI.text(), want) {
			t.Fatalf("cli-legacy does not inject %q", want)
		}
	}

	canonicalTUI := byBuildID["tui-canonical"]
	if got := canonicalTUI.scalar("dir"); got != "experiments/tui" {
		t.Fatalf("tui-canonical dir = %q, want experiments/tui", got)
	}
	if got := canonicalTUI.scalar("main"); got != "./cmd/tui-companion" {
		t.Fatalf("tui-canonical main = %q, want ./cmd/tui-companion", got)
	}
	if !strings.Contains(canonicalTUI.text(), "GOTOOLCHAIN=local") {
		t.Fatal("tui-canonical must build with the local toolchain policy")
	}
	if strings.Contains(canonicalTUI.text(), "legacyTarget") || strings.Contains(canonicalTUI.text(), "legacyName") {
		t.Fatal("tui-canonical must not carry the legacy compatibility identity")
	}

	legacyTUI := byBuildID["tui-legacy"]
	if got := legacyTUI.scalar("dir"); got != "experiments/tui" {
		t.Fatalf("tui-legacy dir = %q, want experiments/tui", got)
	}
	if got := legacyTUI.scalar("main"); got != "./cmd/tui-companion" {
		t.Fatalf("tui-legacy main = %q, want the shared nested ./cmd/tui-companion source", got)
	}
	if got := legacyTUI.scalar("binary"); got != "agentic-sdd-tui" {
		t.Fatalf("tui-legacy binary = %q, want agentic-sdd-tui", got)
	}
	if !strings.Contains(legacyTUI.text(), "GOTOOLCHAIN=local") {
		t.Fatal("tui-legacy must build with the local toolchain policy")
	}
	for _, want := range []string{
		"-X main.legacyTarget=asdx",
		"-X main.legacyName=agentic-sdd-tui",
	} {
		if !strings.Contains(legacyTUI.text(), want) {
			t.Fatalf("tui-legacy does not inject %q", want)
		}
	}

	for _, block := range builds {
		if got := block.list("goos"); !equalStrings(got, []string{"darwin"}) {
			t.Fatalf("build %q goos = %v, want [darwin]", block.scalar("id"), got)
		}
		if got := block.list("goarch"); !equalStrings(got, []string{"amd64", "arm64"}) {
			t.Fatalf("build %q goarch = %v, want [amd64 arm64]", block.scalar("id"), got)
		}
	}

	universals := topLevelBlocks(t, config, "universal_binaries")
	if len(universals) != 4 {
		t.Fatalf("universal_binaries has %d entries, want one per application", len(universals))
	}
	seenNames := map[string]bool{}
	for _, block := range universals {
		ids := block.list("ids")
		if len(ids) != 1 {
			t.Fatalf("universal %q ids = %v, want exactly one application build", block.scalar("id"), ids)
		}
		if block.scalar("replace") != "true" {
			t.Fatalf("universal %q must set replace: true", block.scalar("id"))
		}
		name := block.scalar("name_template")
		if name != universalName(ids[0]) {
			t.Fatalf("universal %q name_template = %q, want %q", block.scalar("id"), name, universalName(ids[0]))
		}
		if seenNames[name] {
			t.Fatalf("duplicate universal name_template %q", name)
		}
		seenNames[name] = true
	}

	// One paired archive with four unique members; no second distribution
	// archive remains.
	archives := topLevelBlocks(t, config, "archives")
	if len(archives) != 1 {
		t.Fatalf("archives has %d entries, want exactly one paired archive", len(archives))
	}
	byArchiveID := map[string]yamlBlock{}
	for _, block := range archives {
		byArchiveID[block.scalar("id")] = block
	}
	// universal_binaries.replace: true removes the single-arch artifacts, so
	// the archive selects the universal output IDs (see
	// specs/013-unified-tui-experience/b6-packaging-research.md §2).
	assertArchive(t, byArchiveID, "pair", []string{
		"cli-core-universal",
		"cli-legacy-universal",
		"tui-canonical-universal",
		"tui-legacy-universal",
	})

	casks := topLevelBlocks(t, config, "homebrew_casks")
	if len(casks) != 1 {
		t.Fatalf("homebrew_casks has %d entries, want exactly one cask", len(casks))
	}
	cask := casks[0]
	if got := cask.scalar("name"); got != "agentic-sdd" {
		t.Fatalf("cask name = %q, want agentic-sdd", got)
	}
	if got := cask.list("ids"); !equalStrings(got, []string{"pair"}) {
		t.Fatalf("cask archive ids = %v, want only the paired archive", got)
	}
	wantBinaries := []string{"asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui"}
	if got := cask.list("binaries"); !equalStrings(got, wantBinaries) {
		t.Fatalf("cask binaries = %v, want all four shipped names %v", got, wantBinaries)
	}
	if got := cask.scalar("directory"); got != "Casks" {
		t.Fatalf("cask directory = %q, want Casks", got)
	}
	// The binaries are unsigned, so macOS quarantine would block execution.
	// The reviewed post-install hook removes the quarantine bit from all four
	// staged executables and does nothing else.
	caskText := cask.text()
	for _, name := range wantBinaries {
		want := `system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/` + name + `"]`
		if !strings.Contains(caskText, want) {
			t.Fatalf("cask post-install hook does not strip quarantine with %q:\n%s", want, caskText)
		}
	}
	for _, forbidden := range []string{"bin.install", "go build", "make", "npm", `system "#{bin}`} {
		if strings.Contains(caskText, forbidden) {
			t.Fatalf("cask must not compile or launch at install time; found %q:\n%s", forbidden, caskText)
		}
	}
	if strings.Contains(config, "\nbrews:") {
		t.Fatal(".goreleaser.yml must not keep the deprecated brews formula section")
	}
	if strings.Contains(config, "directory: Formula") {
		t.Fatal("the tap artifact must live in Casks, not Formula")
	}
}

func assertArchive(t *testing.T, archives map[string]yamlBlock, id string, wantIDs []string) {
	t.Helper()
	block, ok := archives[id]
	if !ok {
		t.Fatalf("missing archive %q", id)
	}
	if got := block.list("ids"); !equalStrings(got, wantIDs) {
		t.Fatalf("archive %q ids = %v, want %v", id, got, wantIDs)
	}
	if got := block.list("formats"); !equalStrings(got, []string{"tar.gz"}) {
		t.Fatalf("archive %q formats = %v, want [tar.gz]", id, got)
	}
	name := block.scalar("name_template")
	for _, want := range []string{"{{ .ProjectName }}", "{{ .Version }}", "{{ .Os }}", "{{ .Arch }}"} {
		if !strings.Contains(name, want) {
			t.Fatalf("archive %q name_template %q lacks %q", id, name, want)
		}
	}
	for _, unwanted := range []string{"_generic", "_homebrew"} {
		if strings.Contains(name, unwanted) {
			t.Fatalf("archive %q name_template %q must not keep the removed %q distribution suffix", id, name, unwanted)
		}
	}
}

// buildIDs lists the four applications in canonical-then-legacy order.
func buildIDs() []string {
	return []string{"cli-core", "cli-legacy", "tui-canonical", "tui-legacy"}
}

func universalName(buildID string) string {
	switch buildID {
	case "cli-core":
		return "asdx-core"
	case "cli-legacy":
		return "agentic-sdd"
	case "tui-canonical":
		return "asdx"
	default:
		return "agentic-sdd-tui"
	}
}

// yamlBlock is one list item under a top-level GoReleaser section. It is a
// deliberately small subset parser: these tests assert our own static file, not
// general YAML.
type yamlBlock struct {
	lines []string
}

func (b yamlBlock) text() string {
	return strings.Join(b.lines, "\n")
}

func (b yamlBlock) scalar(key string) string {
	for _, line := range b.lines {
		for _, prefix := range []string{"    " + key + ":", "  - " + key + ":"} {
			if strings.HasPrefix(line, prefix) {
				return unquote(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
			}
		}
	}
	return ""
}

func (b yamlBlock) list(key string) []string {
	var values []string
	inList := false
	for _, line := range b.lines {
		if strings.HasPrefix(line, "    "+key+":") {
			inList = true
			rest := strings.TrimSpace(strings.TrimPrefix(line, "    "+key+":"))
			rest = strings.TrimPrefix(rest, "[")
			rest = strings.TrimSuffix(rest, "]")
			for _, part := range strings.Split(rest, ",") {
				if value := unquote(strings.TrimSpace(part)); value != "" {
					values = append(values, value)
				}
			}
			continue
		}
		if !inList {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if indent == 6 {
				values = append(values, unquote(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
				continue
			}
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if trimmed != "" && indent <= 4 {
			inList = false
		}
	}
	return values
}

func topLevelBlocks(t *testing.T, source, key string) []yamlBlock {
	t.Helper()
	lines := strings.Split(source, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == key+":" {
			start = i + 1
			break
		}
	}
	if start < 0 {
		t.Fatalf(".goreleaser.yml has no top-level %q section", key)
	}
	var blocks []yamlBlock
	var current *yamlBlock
	for _, line := range lines[start:] {
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			break
		}
		if strings.HasPrefix(line, "  - ") {
			if current != nil {
				blocks = append(blocks, *current)
			}
			current = &yamlBlock{lines: []string{line}}
			continue
		}
		if current != nil {
			current.lines = append(current.lines, line)
		}
	}
	if current != nil {
		blocks = append(blocks, *current)
	}
	return blocks
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]yamlBlock) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestUniversalNameMappingCoversEveryBuild(t *testing.T) {
	// Keep the helper honest: every explicit build ID must have a distinct
	// universal/archive name.
	names := map[string]bool{}
	for _, id := range buildIDs() {
		name := universalName(id)
		if name == "" {
			t.Fatalf("build %q has an empty universal name", id)
		}
		if names[name] {
			t.Fatalf("universal name %q is not unique", name)
		}
		names[name] = true
	}
	want := map[string]bool{"asdx": true, "asdx-core": true, "agentic-sdd": true, "agentic-sdd-tui": true}
	if len(names) != len(want) {
		t.Fatalf("universal names = %v, want the four shipped names %v", names, want)
	}
	for name := range want {
		if !names[name] {
			t.Fatalf("missing shipped universal name %q", name)
		}
	}
}
