// Package harnesscheck executes trusted deterministic check contracts. It is
// not a sandbox, model runtime, approval ledger or technical reviewer.
package harnesscheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"
)

type Contract struct {
	Version int     `json:"version"`
	Checks  []Check `json:"checks"`
}

type Check struct {
	ID             string   `json:"id"`
	CWD            string   `json:"cwd"`
	Scopes         []string `json:"scopes"`
	Prerequisites  []string `json:"prerequisites"`
	TimeoutSeconds int64    `json:"timeout_seconds"`
	Effects        Effects  `json:"effects"`
	Steps          []Step   `json:"steps"`
	LocalRecipe    []string `json:"local_recipe"`
	ExpectedExit   *int     `json:"expected_exit"`
	Evidence       string   `json:"evidence"`
}

type Effects struct {
	Writes  []string `json:"writes"`
	Network string   `json:"network"`
}

type Step struct {
	Argv []string          `json:"argv"`
	Env  map[string]string `json:"env"`
}

var checkID = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func relative(p string) bool {
	if p == "" || strings.ContainsAny(p, "\\\x00") || path.IsAbs(p) {
		return false
	}
	if len(p) >= 2 && p[1] == ':' && ((p[0] >= 'a' && p[0] <= 'z') || (p[0] >= 'A' && p[0] <= 'Z')) {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

func Parse(data []byte) (Contract, error) {
	var c Contract
	// No v1 contract field is nullable. encoding/json otherwise silently turns
	// null string/map entries into zero values, e.g. an empty environment value.
	tokens := json.NewDecoder(bytes.NewReader(data))
	for {
		token, err := tokens.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return c, fmt.Errorf("decode contract: %w", err)
		}
		if token == nil {
			return c, fmt.Errorf("null contract fields are not allowed")
		}
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, fmt.Errorf("decode contract: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return c, fmt.Errorf("contract must contain exactly one JSON object")
	}
	if c.Version != 1 || len(c.Checks) == 0 {
		return c, fmt.Errorf("expected version 1 and nonempty checks")
	}
	ids, logs := map[string]bool{}, map[string]bool{}
	for _, ch := range c.Checks {
		if !checkID.MatchString(ch.ID) || ids[ch.ID] {
			return c, fmt.Errorf("invalid or duplicate check ID %q", ch.ID)
		}
		ids[ch.ID] = true
		if !relative(ch.CWD) || !relative(ch.Evidence) || path.Clean(ch.Evidence) == "." || path.Clean(ch.Evidence) == "result.json" {
			return c, fmt.Errorf("%s: invalid cwd/evidence path", ch.ID)
		}
		log := path.Clean(ch.Evidence)
		for previous := range logs {
			if log == previous || strings.HasPrefix(log, previous+"/") || strings.HasPrefix(previous, log+"/") {
				return c, fmt.Errorf("colliding evidence paths")
			}
		}
		if strings.HasPrefix(log, "result.json/") {
			return c, fmt.Errorf("reserved result path")
		}
		logs[log] = true
		if ch.TimeoutSeconds < 1 || ch.TimeoutSeconds > int64((1<<63-1)/time.Second) || ch.ExpectedExit == nil || *ch.ExpectedExit != 0 {
			return c, fmt.Errorf("%s: positive bounded timeout and expected_exit 0 required", ch.ID)
		}
		if len(ch.Scopes) == 0 || ch.Prerequisites == nil || ch.LocalRecipe == nil || ch.Effects.Writes == nil || strings.TrimSpace(ch.Effects.Network) == "" || len(ch.Steps) == 0 {
			return c, fmt.Errorf("%s: missing required contract fields", ch.ID)
		}
		seen := map[string]bool{}
		for _, scope := range ch.Scopes {
			if (scope != "baseline" && scope != "focused" && scope != "required") || seen[scope] {
				return c, fmt.Errorf("invalid/duplicate scope %q", scope)
			}
			seen[scope] = true
		}
		for _, list := range [][]string{ch.Prerequisites, ch.LocalRecipe, ch.Effects.Writes} {
			for _, s := range list {
				if strings.TrimSpace(s) == "" || strings.ContainsRune(s, 0) {
					return c, fmt.Errorf("empty/invalid contract string")
				}
			}
		}
		for _, step := range ch.Steps {
			if len(step.Argv) == 0 || step.Env == nil {
				return c, fmt.Errorf("%s: argv and env required", ch.ID)
			}
			for _, arg := range step.Argv {
				if arg == "" || strings.ContainsRune(arg, 0) {
					return c, fmt.Errorf("invalid argv")
				}
			}
			for key, value := range step.Env {
				if !envName.MatchString(key) || strings.ContainsRune(value, 0) {
					return c, fmt.Errorf("invalid environment override")
				}
			}
		}
	}
	return c, nil
}

func Select(c Contract, scope string, ids []string) ([]Check, error) {
	if scope != "baseline" && scope != "focused" && scope != "required" {
		return nil, fmt.Errorf("scope must be baseline, focused or required")
	}
	if (scope == "focused") != (len(ids) > 0) {
		return nil, fmt.Errorf("only focused scope requires explicit --check IDs")
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		if wanted[id] {
			return nil, fmt.Errorf("duplicate selected ID %q", id)
		}
		wanted[id] = true
	}
	var selected []Check
	for _, ch := range c.Checks {
		for _, s := range ch.Scopes {
			if s == scope && (scope != "focused" || wanted[ch.ID]) {
				selected = append(selected, ch)
				delete(wanted, ch.ID)
				break
			}
		}
	}
	if len(selected) == 0 || len(wanted) > 0 {
		return nil, fmt.Errorf("empty selection or unknown/ineligible check IDs")
	}
	return selected, nil
}
