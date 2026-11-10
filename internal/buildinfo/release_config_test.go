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
// formula proof remains the nonpublishing snapshot task.
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

	builds := topLevelBlocks(t, config, "builds")
	byBuildID := map[string]yamlBlock{}
	for _, block := range builds {
		byBuildID[block.scalar("id")] = block
	}
	for _, id := range []string{"cli-generic", "cli-homebrew", "tui-companion"} {
		block, ok := byBuildID[id]
		if !ok {
			t.Fatalf("missing explicit build ID %q (found %v)", id, sortedKeys(byBuildID))
		}
		if block.scalar("binary") != universalName(id) {
			t.Fatalf("build %q binary = %q, want %q so the archive member is deterministic", id, block.scalar("binary"), universalName(id))
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
	}

	generic := byBuildID["cli-generic"]
	if got := generic.scalar("main"); got != "./cmd/agentic-sdd" {
		t.Fatalf("cli-generic main = %q, want ./cmd/agentic-sdd", got)
	}
	if !strings.Contains(generic.text(), "-X agentic-sdd/internal/buildinfo.DistributionMarker=generic") {
		t.Fatal("cli-generic must carry only the generic distribution marker")
	}
	if strings.Contains(generic.text(), "DistributionMarker=homebrew") {
		t.Fatal("cli-generic must not carry the Homebrew marker")
	}
	if got := strings.Count(config, "DistributionMarker=homebrew"); got != 1 {
		t.Fatalf(".goreleaser.yml carries the Homebrew marker %d times, want only the cli-homebrew build", got)
	}
	if got := strings.Count(config, "DistributionMarker=generic"); got != 2 {
		t.Fatalf(".goreleaser.yml carries the generic marker %d times, want the generic CLI and the shared companion", got)
	}

	homebrew := byBuildID["cli-homebrew"]
	if got := homebrew.scalar("main"); got != "./cmd/agentic-sdd" {
		t.Fatalf("cli-homebrew main = %q, want ./cmd/agentic-sdd", got)
	}
	if !strings.Contains(homebrew.text(), "-X agentic-sdd/internal/buildinfo.DistributionMarker=homebrew") {
		t.Fatal("cli-homebrew must carry the Homebrew distribution marker")
	}

	companion := byBuildID["tui-companion"]
	if got := companion.scalar("dir"); got != "experiments/tui" {
		t.Fatalf("tui-companion dir = %q, want experiments/tui", got)
	}
	if got := companion.scalar("main"); got != "./cmd/tui-companion" {
		t.Fatalf("tui-companion main = %q, want ./cmd/tui-companion", got)
	}
	if !strings.Contains(companion.text(), "GOTOOLCHAIN=local") {
		t.Fatal("tui-companion must build with the local toolchain policy")
	}
	if !strings.Contains(companion.text(), "DistributionMarker=generic") {
		t.Fatal("the shared companion artifact must stay generic; only the Homebrew CLI carries the marker")
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
	if len(universals) != 3 {
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

	archives := topLevelBlocks(t, config, "archives")
	if len(archives) != 2 {
		t.Fatalf("archives has %d entries, want the generic and Homebrew pairs", len(archives))
	}
	byArchiveID := map[string]yamlBlock{}
	for _, block := range archives {
		byArchiveID[block.scalar("id")] = block
	}
	// universal_binaries.replace: true removes the single-arch artifacts, so
	// the archives select the universal output IDs (see
	// specs/013-unified-tui-experience/b6-packaging-research.md §2).
	assertArchive(t, byArchiveID, "cli-generic-pair", []string{"cli-generic-universal", "tui-companion-universal"}, "_generic")
	assertArchive(t, byArchiveID, "cli-homebrew-pair", []string{"cli-homebrew-universal", "tui-companion-universal"}, "_homebrew")

	brews := topLevelBlocks(t, config, "brews")
	if len(brews) != 1 {
		t.Fatalf("brews has %d entries, want exactly one formula", len(brews))
	}
	formula := brews[0]
	if got := formula.list("ids"); !equalStrings(got, []string{"cli-homebrew-pair"}) {
		t.Fatalf("formula archive ids = %v, want only the Homebrew pair", got)
	}
	install := formula.blockScalar("install")
	for _, want := range []string{`bin.install "agentic-sdd-homebrew" => "agentic-sdd"`, `bin.install "agentic-sdd-tui"`} {
		if !strings.Contains(install, want) {
			t.Fatalf("formula install does not contain %q:\n%s", want, install)
		}
	}
	for _, forbidden := range []string{"go build", "make", "npm", "system \""} {
		if strings.Contains(install, forbidden) {
			t.Fatalf("formula install must not compile or launch at install time; found %q:\n%s", forbidden, install)
		}
	}
	if strings.Contains(config, "post_install:") {
		t.Fatal("the formula must not define a post_install hook")
	}
	if testScript := formula.blockScalar("test"); !strings.Contains(testScript, `"#{bin}/agentic-sdd-tui", "--version"`) {
		t.Fatalf("formula test must exercise the non-interactive companion version query:\n%s", testScript)
	}
}

func assertArchive(t *testing.T, archives map[string]yamlBlock, id string, wantIDs []string, suffix string) {
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
	for _, want := range []string{"{{ .ProjectName }}", "{{ .Version }}", "{{ .Os }}", "{{ .Arch }}", suffix} {
		if !strings.Contains(name, want) {
			t.Fatalf("archive %q name_template %q lacks %q", id, name, want)
		}
	}
}

func universalName(buildID string) string {
	switch buildID {
	case "cli-generic":
		return "agentic-sdd-generic"
	case "cli-homebrew":
		return "agentic-sdd-homebrew"
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

func (b yamlBlock) blockScalar(key string) string {
	var values []string
	inBlock := false
	for _, line := range b.lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "    "+key+":") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "    "+key+":"))
			if rest != "|" && rest != "|-" && rest != ">" && rest != ">-" {
				values = append(values, rest)
			}
			inBlock = true
			continue
		}
		if inBlock {
			if trimmed == "" {
				values = append(values, "")
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if indent <= 4 {
				break
			}
			values = append(values, strings.TrimSpace(line))
		}
	}
	return strings.Join(values, "\n")
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
	for _, id := range []string{"cli-generic", "cli-homebrew", "tui-companion"} {
		name := universalName(id)
		if name == "" {
			t.Fatalf("build %q has an empty universal name", id)
		}
		if names[name] {
			t.Fatalf("universal name %q is not unique", name)
		}
		names[name] = true
	}
}
