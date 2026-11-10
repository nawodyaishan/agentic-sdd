package appops

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestAgentRoutingMatchesDocumentedPolicy prevents the curated display table
// from drifting from opencode-pack/routing.md. It reads the repository file as
// test input only; the production package performs no filesystem access.
func TestAgentRoutingMatchesDocumentedPolicy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "opencode-pack", "routing.md"))
	if err != nil {
		t.Fatalf("read routing policy: %v", err)
	}

	rows := parseRoutingRows(t, string(data))
	if len(rows) == 0 {
		t.Fatal("routing policy table parsed no rows; the fixture format changed")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row.id] = true
		routing, ok := AgentRoutingFor(row.id)
		if !ok {
			t.Errorf("routing.md documents %q but no curated routing entry exists", row.id)
			continue
		}
		if routing.Mode == "" || routing.Note == "" {
			t.Errorf("%s: routing mode/note is empty", row.id)
		}
		if !strings.Contains(routing.Note, "policy, not automatic failover") {
			t.Errorf("%s: routing note lacks the manual-failover policy statement", row.id)
		}
		if row.id == "sdd-orchestrator" {
			if routing.Model != "Active session model" {
				t.Errorf("orchestrator model = %q, want the active session model", routing.Model)
			}
			if !strings.Contains(strings.ToLower(routing.Note), "human-selected session model") {
				t.Errorf("orchestrator note = %q, want the human-selected fallback wording", routing.Note)
			}
			continue
		}
		if routing.Model != row.model {
			t.Errorf("%s: curated model %q != documented %q", row.id, routing.Model, row.model)
		}
		if routing.Provider != routingProviderForModel(row.model) {
			t.Errorf("%s: provider %q does not match the curated model prefix %q", row.id, routing.Provider, row.model)
		}
		checkDocumentedAlternatives(t, row, routing)
	}

	for _, id := range AgentRoutingIDs() {
		if isPresetID(id) {
			continue
		}
		if !seen[id] {
			t.Errorf("curated routing entry %q has no row in routing.md", id)
		}
	}
}

// checkDocumentedAlternatives compares the curated fallback list with the
// table's "Documented alternatives" cell. Policy-pointer cells are represented
// by a note instead of invented model names.
func checkDocumentedAlternatives(t *testing.T, row routingRow, routing AgentRouting) {
	t.Helper()
	alternatives := strings.ToLower(row.alternatives)
	switch {
	case strings.Contains(alternatives, "claude-agent exception"):
		if !strings.Contains(routing.Note, "Claude-agent exception") {
			t.Errorf("%s: note does not mention the Claude-agent exception", row.id)
		}
		if len(routing.Fallbacks) == 0 {
			t.Errorf("%s: Claude exception entry has no documented fallbacks", row.id)
		}
	case strings.Contains(alternatives, "beam cross-lane"):
		if !strings.Contains(routing.Note, "Beam cross-lane rule") {
			t.Errorf("%s: note does not mention the Beam cross-lane rule", row.id)
		}
	case strings.Contains(alternatives, "human-selected session model"):
		if len(routing.Fallbacks) != 0 {
			t.Errorf("%s: orchestrator must not claim a model fallback ladder", row.id)
		}
	default:
		documented := backtickedModels(row.alternatives)
		if len(documented) == 0 {
			t.Errorf("%s: documented alternatives contain no parsable model", row.id)
			return
		}
		if strings.Join(documented, ",") != strings.Join(routing.Fallbacks, ",") {
			t.Errorf("%s: curated fallbacks %v != documented %v", row.id, routing.Fallbacks, documented)
		}
	}
}

// TestPresetRoutingMatchesFrontmatter keeps the four preset entries aligned
// with their shipped profile frontmatter and the documented ladder.
func TestPresetRoutingMatchesFrontmatter(t *testing.T) {
	presetDir := filepath.Join("..", "..", "opencode-pack", "presets", "free-first")
	for _, id := range []string{
		"sdd-implement-1",
		"sdd-implement-2",
		"sdd-test-implement-1",
		"sdd-test-implement-2",
	} {
		data, err := os.ReadFile(filepath.Join(presetDir, id+".md"))
		if err != nil {
			t.Fatalf("read preset %s: %v", id, err)
		}
		model := frontmatterModel(string(data))
		if model == "" {
			t.Fatalf("preset %s has no frontmatter model", id)
		}
		routing, ok := AgentRoutingFor(id)
		if !ok {
			t.Fatalf("preset %s has no curated routing entry", id)
		}
		if routing.Model != model {
			t.Errorf("%s: curated model %q != frontmatter %q", id, routing.Model, model)
		}
		wantLadder := []string{
			"opencode/glm-5.3-flash-free",
			"opencode/mimo-v2.6-flash-free",
			"opencode-go/deepseek-v4.1-flash",
			"opencode-go/glm-5.3-flash",
			"opencode-go/minimax-m3.1-flash",
			"opencode-go/mimo-v2.6-flash",
		}
		if strings.Join(routing.Fallbacks, ",") != strings.Join(wantLadder, ",") {
			t.Errorf("%s: fallback ladder %v != documented ladder %v", id, routing.Fallbacks, wantLadder)
		}
		if !strings.Contains(routing.Note, "authorization") {
			t.Errorf("%s: preset note does not mention the authorization boundary", id)
		}
	}
}

func TestAgentRoutingDefensiveCopy(t *testing.T) {
	first, ok := AgentRoutingFor("sdd-debug-diagnose")
	if !ok {
		t.Fatal("missing routing entry")
	}
	first.Fallbacks[0] = "mutated"
	second, _ := AgentRoutingFor("sdd-debug-diagnose")
	if second.Fallbacks[0] == "mutated" {
		t.Fatal("AgentRoutingFor returned a mutable internal slice")
	}
}

func TestAgentRoutingUnknownProfile(t *testing.T) {
	if _, ok := AgentRoutingFor("local-unlisted-profile"); ok {
		t.Fatal("unlisted profile must not inherit curated routing")
	}
}

type routingRow struct {
	id           string
	mode         string
	model        string
	alternatives string
}

func parseRoutingRows(t *testing.T, text string) []routingRow {
	t.Helper()
	var rows []routingRow
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "| `sdd-") {
			continue
		}
		cells := strings.Split(trimmed, "|")
		if len(cells) < 6 {
			t.Errorf("unexpected routing row: %q", line)
			continue
		}
		id := strings.Trim(strings.TrimSpace(cells[1]), "`")
		mode := strings.TrimSpace(cells[2])
		modelCell := strings.TrimSpace(cells[3])
		model := ""
		if models := backtickedModels(modelCell); len(models) > 0 {
			model = models[0]
		} else {
			model = strings.Trim(modelCell, "`")
		}
		if strings.HasPrefix(strings.ToLower(model), "none") {
			model = "Active session model"
		}
		rows = append(rows, routingRow{
			id:           id,
			mode:         mode,
			model:        model,
			alternatives: strings.TrimSpace(cells[4]),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	return rows
}

func backtickedModels(cell string) []string {
	var models []string
	for {
		start := strings.IndexByte(cell, '`')
		if start < 0 {
			break
		}
		rest := cell[start+1:]
		end := strings.IndexByte(rest, '`')
		if end < 0 {
			break
		}
		models = append(models, rest[:end])
		cell = rest[end+1:]
	}
	return models
}

func frontmatterModel(text string) string {
	inFrontmatter := false
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(line, "model:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "model:"))
		}
	}
	return ""
}

func isPresetID(id string) bool {
	switch id {
	case "sdd-implement-1", "sdd-implement-2", "sdd-test-implement-1", "sdd-test-implement-2":
		return true
	default:
		return false
	}
}
