package harnesscheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureContract() Contract {
	zero := 0
	return Contract{Version: 1, Checks: []Check{{
		ID: "fixture", CWD: ".", Scopes: []string{"baseline", "focused", "required"},
		Prerequisites: []string{}, TimeoutSeconds: 5, Effects: Effects{Writes: []string{}, Network: "none"},
		Steps:       []Step{{Argv: []string{"true"}, Env: map[string]string{}}},
		LocalRecipe: []string{}, ExpectedExit: &zero, Evidence: "fixture.log",
	}}}
}

func TestContractValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Contract)
	}{
		{"version", func(c *Contract) { c.Version = 2 }},
		{"duplicates", func(c *Contract) { c.Checks = append(c.Checks, c.Checks[0]) }},
		{"timeout", func(c *Contract) { c.Checks[0].TimeoutSeconds = 0 }},
		{"argv", func(c *Contract) { c.Checks[0].Steps[0].Argv = nil }},
		{"cwd", func(c *Contract) { c.Checks[0].CWD = "../outside" }},
		{"log", func(c *Contract) { c.Checks[0].Evidence = "/tmp/outside" }},
		{"log-collision", func(c *Contract) { c.Checks[0].Evidence = "result.json" }},
		{"env", func(c *Contract) { c.Checks[0].Steps[0].Env["BAD=KEY"] = "x" }},
		{"scope", func(c *Contract) { c.Checks[0].Scopes = []string{"completion"} }},
		{"missing-exit", func(c *Contract) { c.Checks[0].ExpectedExit = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := fixtureContract()
			tc.change(&c)
			data, _ := json.Marshal(c)
			if _, err := Parse(data); err == nil {
				t.Fatal("accepted invalid contract")
			}
		})
	}
	data, _ := json.Marshal(fixtureContract())
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
	var unknown map[string]any
	_ = json.Unmarshal(data, &unknown)
	unknown["ignored"] = true
	data, _ = json.Marshal(unknown)
	if _, err := Parse(data); err == nil {
		t.Fatal("accepted unknown field")
	}
	if _, err := Parse([]byte(`{"version":1,"checks":[]} {}`)); err == nil {
		t.Fatal("accepted trailing JSON")
	}
	data, _ = json.Marshal(fixtureContract())
	data = []byte(strings.Replace(string(data), `"env":{}`, `"env":{"VALUE":null}`, 1))
	if _, err := Parse(data); err == nil {
		t.Fatal("null environment value accepted as empty string")
	}
}

func TestRepositoryContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "harness", "checks.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	checks, err := Select(c, "required", nil)
	if err != nil || len(checks) != 6 {
		t.Fatalf("required selection: %d %v", len(checks), err)
	}
}

func TestSelection(t *testing.T) {
	c := fixtureContract()
	for _, scope := range []string{"baseline", "required"} {
		got, err := Select(c, scope, nil)
		if err != nil || len(got) != 1 {
			t.Fatalf("%s: %v %v", scope, got, err)
		}
		if _, err := Select(c, scope, []string{"fixture"}); err == nil {
			t.Fatal("partial required/baseline selection accepted")
		}
	}
	for _, ids := range [][]string{nil, {"absent"}, {"fixture", "fixture"}} {
		if _, err := Select(c, "focused", ids); err == nil {
			t.Fatalf("accepted focused IDs %v", ids)
		}
	}
	if _, err := Select(c, "focused", []string{"fixture"}); err != nil {
		t.Fatal(err)
	}
}

func TestTidyUsesIndependentSnapshotRatherThanGitIndex(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "harness", "checks.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range c.Checks {
		if check.ID == "tidy-check" {
			if len(check.Steps) != 1 || len(check.Prerequisites) != 1 || check.Prerequisites[0] != "go" {
				t.Fatal("tidy still depends on a staged Git index rather than pre/post module identity")
			}
			return
		}
	}
	t.Fatal("tidy check missing")
}

func TestContainerRecipeCannotPullImplicitly(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "docker-e2e.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "--pull=never") {
		t.Fatal("container check can implicitly download an image")
	}
}
