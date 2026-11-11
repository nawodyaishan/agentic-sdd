package fleet

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func mustParseConfig(t *testing.T, body string) *Config {
	t.Helper()
	cfg, err := ParseConfig(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	return cfg
}

func parseConfigError(t *testing.T, body string) error {
	t.Helper()
	_, err := ParseConfig(strings.NewReader(body))
	if err == nil {
		t.Fatal("ParseConfig() succeeded, want error")
	}
	return err
}

func wantErrorContains(t *testing.T, err error, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("error %q does not contain %q", err, part)
		}
	}
}

func TestFleetParse(t *testing.T) {
	t.Run("valid config and defaults", func(t *testing.T) {
		cfg := mustParseConfig(t, `{
  "schemaVersion": 1,
  "fleet": "personal",
  "preset": "opencode-go",
  "instances": {
    "sdd-orchestrator": {"source": "agent:sdd-orchestrator"},
    "sdd-implement-1": {"source": "preset:sdd-implement-1", "model": "opencode-go/deepseek-v4.1-flash#high", "enabled": true},
    "sdd-implement-free-1": {"source": "preset:sdd-implement-1", "model": "opencode/deepseek-v4.1-flash-free", "enabled": false}
  }
}`)
		if cfg.SchemaVersion != SchemaVersion {
			t.Fatalf("SchemaVersion = %d, want %d", cfg.SchemaVersion, SchemaVersion)
		}
		if cfg.Fleet != "personal" || cfg.Preset != "opencode-go" {
			t.Fatalf("labels = %q/%q, want personal/opencode-go", cfg.Fleet, cfg.Preset)
		}
		if _, ok := cfg.Instances["sdd-orchestrator"]; !ok {
			t.Fatal("sdd-orchestrator instance missing")
		}
		if !cfg.Instances["sdd-orchestrator"].IsEnabled() {
			t.Fatal("omitted enabled must default to true")
		}
		if !cfg.Instances["sdd-implement-1"].IsEnabled() {
			t.Fatal("explicit enabled true must stay true")
		}
		if cfg.Instances["sdd-implement-free-1"].IsEnabled() {
			t.Fatal("explicit enabled false must stay false")
		}
		if got := cfg.Instances["sdd-implement-1"].Model; got != "opencode-go/deepseek-v4.1-flash#high" {
			t.Fatalf("model = %q", got)
		}
	})

	t.Run("null enabled and empty model semantics", func(t *testing.T) {
		cfg := mustParseConfig(t, `{
  "schemaVersion": 1,
  "instances": {
    "sdd-orchestrator": {"source": "agent:sdd-orchestrator", "enabled": null, "model": ""}
  }
}`)
		inst := cfg.Instances["sdd-orchestrator"]
		if !inst.IsEnabled() {
			t.Fatal("explicit null enabled must default to true")
		}
		if inst.Model != "" {
			t.Fatalf("expected empty model, got %q", inst.Model)
		}
	})

	t.Run("oversize", func(t *testing.T) {
		body := `{"schemaVersion":1,"instances":{}}`
		body += strings.Repeat(" ", MaxConfigBytes+1-len(body))
		err := parseConfigError(t, body)
		wantErrorContains(t, err, "exceeds", fmt.Sprint(MaxConfigBytes))
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("error does not wrap ErrInvalidConfig: %v", err)
		}
	})

	t.Run("invalid UTF-8", func(t *testing.T) {
		data := append([]byte(`{"schemaVersion":1,"instances":{},"x":"`), 0xff)
		data = append(data, []byte(`"}`)...)
		_, err := ParseConfig(bytes.NewReader(data))
		if err == nil {
			t.Fatal("ParseConfig() succeeded, want error")
		}
		wantErrorContains(t, err, "UTF-8")
	})

	t.Run("duplicate keys", func(t *testing.T) {
		cases := []struct {
			name string
			body string
		}{
			{"top level", `{"schemaVersion":1,"schemaVersion":1,"instances":{}}`},
			{"instance", `{"schemaVersion":1,"instances":{"a":{"source":"agent:a","source":"agent:b"}}}`},
			{"instances map", `{"schemaVersion":1,"instances":{"a":{"source":"agent:a"},"a":{"source":"agent:b"}}}`},
			{"nested unknown object", `{"schemaVersion":1,"instances":{},"extra":{"a":1,"a":2}}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := parseConfigError(t, tc.body)
				wantErrorContains(t, err, "duplicate key")
			})
		}
	})

	t.Run("unknown fields", func(t *testing.T) {
		cases := []struct {
			name string
			body string
			key  string
		}{
			{"top level", `{"schemaVersion":1,"instances":{},"extra":true}`, "extra"},
			{"instance", `{"schemaVersion":1,"instances":{"a":{"source":"agent:a","extra":true}}}`, "extra"},
			{"policy array", `{"schemaVersion":1,"instances":{"a":{"source":"agent:a","permissions":["edit"]}}}`, "permissions"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := parseConfigError(t, tc.body)
				wantErrorContains(t, err, "unknown field", tc.key)
			})
		}
	})

	t.Run("policy keys rejected as unknown fields", func(t *testing.T) {
		keys := []string{
			"permissions", "shell", "execute", "subagent", "fallback",
			"fallbacks", "automaticDelegation", "spending", "providerSwitching",
		}
		for _, key := range keys {
			for _, scope := range []string{"top", "instance"} {
				t.Run(scope+"/"+key, func(t *testing.T) {
					var body string
					if scope == "top" {
						body = fmt.Sprintf(`{"schemaVersion":1,"instances":{},"%s":true}`, key)
					} else {
						body = fmt.Sprintf(`{"schemaVersion":1,"instances":{"a":{"source":"agent:a","%s":true}}}`, key)
					}
					err := parseConfigError(t, body)
					wantErrorContains(t, err, "unknown field", key)
				})
			}
		}
	})

	t.Run("traversal rejected", func(t *testing.T) {
		cases := []struct {
			name string
			body string
		}{
			{"instance id dotdot", `{"schemaVersion":1,"instances":{"../evil":{"source":"agent:a"}}}`},
			{"instance id slash", `{"schemaVersion":1,"instances":{"a/b":{"source":"agent:a"}}}`},
			{"instance id backslash", `{"schemaVersion":1,"instances":{"a\\b":{"source":"agent:a"}}}`},
			{"source dotdot", `{"schemaVersion":1,"instances":{"a":{"source":"agent:../evil"}}}`},
			{"source slash", `{"schemaVersion":1,"instances":{"a":{"source":"agent:a/b"}}}`},
			{"source backslash", `{"schemaVersion":1,"instances":{"a":{"source":"agent:a\\b"}}}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := parseConfigError(t, tc.body)
				wantErrorContains(t, err, "path separators")
			})
		}
	})

	t.Run("invalid instance IDs", func(t *testing.T) {
		ids := []string{"SDD-foo", "sdd_foo", "-sdd", strings.Repeat("a", 65)}
		for _, id := range ids {
			body := fmt.Sprintf(`{"schemaVersion":1,"instances":{%q:{"source":"agent:a"}}}`, id)
			err := parseConfigError(t, body)
			wantErrorContains(t, err, "invalid instance ID")
		}
	})

	t.Run("unsupported schema", func(t *testing.T) {
		cases := []struct {
			name string
			body string
		}{
			{"version 2", `{"schemaVersion":2,"instances":{}}`},
			{"missing", `{"instances":{}}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := parseConfigError(t, tc.body)
				wantErrorContains(t, err, "unsupported schemaVersion")
			})
		}
		err := parseConfigError(t, `{"schemaVersion":"1","instances":{}}`)
		wantErrorContains(t, err, "cannot unmarshal")
	})

	t.Run("invalid models", func(t *testing.T) {
		models := []string{
			"opencode/deepseek v4",
			"opencode/",
			"opencode//deepseek",
			"opencode/foo/#high",
			"opencode/foo#",
			"opencode/foo##high",
			"/deepseek",
			"OpenCode/foo",
		}
		for _, model := range models {
			body := fmt.Sprintf(`{"schemaVersion":1,"instances":{"a":{"source":"agent:a","model":%q}}}`, model)
			err := parseConfigError(t, body)
			wantErrorContains(t, err, "model")
		}
		err := parseConfigError(t, `{"schemaVersion":1,"instances":{"a":{"source":"agent:a","model":123}}}`)
		wantErrorContains(t, err, "cannot unmarshal")
	})

	t.Run("missing source", func(t *testing.T) {
		err := parseConfigError(t, `{"schemaVersion":1,"instances":{"a":{}}}`)
		wantErrorContains(t, err, "source is required")
	})

	t.Run("trailing data", func(t *testing.T) {
		err := parseConfigError(t, `{"schemaVersion":1,"instances":{}}{}`)
		wantErrorContains(t, err, "unexpected data")
	})

	t.Run("diagnostics do not echo values", func(t *testing.T) {
		err := parseConfigError(t, `{"schemaVersion":1,"instances":{"a":{"source":"agent:a","apiKey":"supersecret-token"}}}`)
		wantErrorContains(t, err, "unknown field", "apiKey")
		if strings.Contains(err.Error(), "supersecret-token") {
			t.Fatalf("error echoed a secret value: %v", err)
		}
	})
}
