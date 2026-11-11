package fleet

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"agentic-sdd/internal/skillsync"
)

// goldenFleetMapping is the spec R1-R2 reconciliation contract for the 18
// historical machine names. Five overlap canonical agent files, two use the
// reviewed read-only adapter sources under adapters/ (B6 Option A), vision
// maps to the canonical visual-inspector role, four are existing free-first
// presets and six reuse an implementation preset with only a model change.
var goldenFleetMapping = map[string]string{
	"sdd-architecture-review-9r": "agent:sdd-architecture-review-9r",
	"sdd-code-review-9r":         "agent:sdd-code-review-9r",
	"sdd-review-verification-1":  "agent:sdd-review-verification-1",
	"sdd-review-verification-2":  "agent:sdd-review-verification-2",
	"sdd-lightweight":            "agent:sdd-lightweight",
	"sdd-implement-1":            "preset:sdd-implement-1",
	"sdd-implement-2":            "preset:sdd-implement-2",
	"sdd-test-implement-1":       "preset:sdd-test-implement-1",
	"sdd-test-implement-2":       "preset:sdd-test-implement-2",
	"sdd-implement-9r-1":         "preset:sdd-implement-1",
	"sdd-implement-9r-2":         "preset:sdd-implement-2",
	"sdd-implement-9r-cheap-1":   "preset:sdd-implement-1",
	"sdd-implement-9r-cheap-2":   "preset:sdd-implement-2",
	"sdd-implement-free-1":       "preset:sdd-implement-1",
	"sdd-implement-free-2":       "preset:sdd-implement-2",
	"sdd-spec-9r":                "agent:sdd-spec-9r",
	"sdd-plan-9r":                "agent:sdd-plan-9r",
	"vision":                     "agent:sdd-visual-inspector",
}

var fixtureAgentNames = []string{
	"sdd-architecture-review-9r",
	"sdd-code-review-9r",
	"sdd-lightweight",
	"sdd-orchestrator",
	"sdd-review-verification-1",
	"sdd-review-verification-2",
	"sdd-visual-inspector",
	"real",
}

var fixtureAdapterNames = []string{
	"sdd-plan-9r",
	"sdd-spec-9r",
}

var fixturePresetNames = []string{
	"sdd-implement-1",
	"sdd-implement-2",
	"sdd-test-implement-1",
	"sdd-test-implement-2",
}

func fixtureSource(name string) []byte {
	return []byte("---\nmodel: opencode/deepseek-v4.1-flash-free\n---\n\nfixture body: " + name + "\n")
}

func fixturePack() fstest.MapFS {
	pack := fstest.MapFS{}
	for _, name := range fixtureAgentNames {
		pack["agents/"+name+".md"] = &fstest.MapFile{Data: fixtureSource(name)}
	}
	for _, name := range fixtureAdapterNames {
		pack["adapters/"+name+".md"] = &fstest.MapFile{Data: fixtureSource(name)}
	}
	for _, name := range fixturePresetNames {
		pack["presets/free-first/"+name+".md"] = &fstest.MapFile{Data: fixtureSource(name)}
	}
	return pack
}

func configFromMapping(mapping map[string]string, include func(string) bool) *Config {
	instances := make(map[string]Instance, len(mapping))
	for id, source := range mapping {
		if include != nil && !include(id) {
			continue
		}
		instances[id] = Instance{Source: source, Model: "opencode/deepseek-v4.1-flash-free"}
	}
	return &Config{
		SchemaVersion: SchemaVersion,
		Fleet:         "personal",
		Preset:        "opencode-go",
		Instances:     instances,
	}
}

func TestFleetResolve(t *testing.T) {
	t.Run("golden 18-name mapping", func(t *testing.T) {
		if len(goldenFleetMapping) != 18 {
			t.Fatalf("golden mapping has %d names, want 18", len(goldenFleetMapping))
		}
		agentRefs, presetRefs := 0, 0
		for _, source := range goldenFleetMapping {
			switch {
			case strings.HasPrefix(source, "agent:"):
				agentRefs++
			case strings.HasPrefix(source, "preset:"):
				presetRefs++
			default:
				t.Errorf("unexpected source namespace in %q", source)
			}
		}
		if agentRefs != 8 || presetRefs != 10 {
			t.Fatalf("mapping namespaces = %d agent / %d preset, want 8/10", agentRefs, presetRefs)
		}

		cfg := configFromMapping(goldenFleetMapping, nil)
		got, err := Resolve(fixturePack(), cfg)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if len(got) != 18 {
			t.Fatalf("resolved %d instances, want 18", len(got))
		}
		byID := make(map[string]Resolved, len(got))
		for _, resolved := range got {
			byID[resolved.InstanceID] = resolved
		}
		for id, source := range goldenFleetMapping {
			resolved, ok := byID[id]
			if !ok {
				t.Errorf("instance %q was not resolved", id)
				continue
			}
			ref, err := ParseSourceRef(source)
			if err != nil {
				t.Fatalf("ParseSourceRef(%q) error = %v", source, err)
			}
			if resolved.SourceKind != ref.Kind || resolved.SourceName != ref.Name {
				t.Errorf("instance %q resolved to %s:%s, want %s", id, resolved.SourceKind, resolved.SourceName, ref)
			}
			wantBytes := fixtureSource(ref.Name)
			if !bytes.Equal(resolved.SourceBytes, wantBytes) {
				t.Errorf("instance %q source bytes differ from walked file", id)
			}
			if wantDigest := fmt.Sprintf("%x", sha256.Sum256(wantBytes)); resolved.SourceDigest != wantDigest {
				t.Errorf("instance %q digest = %q, want %q", id, resolved.SourceDigest, wantDigest)
			}
			if resolved.Model != "opencode/deepseek-v4.1-flash-free" {
				t.Errorf("instance %q model = %q", id, resolved.Model)
			}
			if !resolved.Enabled {
				t.Errorf("instance %q should be enabled by default", id)
			}
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].InstanceID > got[i].InstanceID {
				t.Fatalf("resolved instances are not sorted: %q before %q", got[i-1].InstanceID, got[i].InstanceID)
			}
		}
	})

	t.Run("adapter sources resolve from adapters directory", func(t *testing.T) {
		for _, id := range fixtureAdapterNames {
			t.Run(id, func(t *testing.T) {
				cfg := &Config{
					SchemaVersion: SchemaVersion,
					Instances:     map[string]Instance{id: {Source: "agent:" + id}},
				}
				got, err := Resolve(fixturePack(), cfg)
				if err != nil {
					t.Fatalf("Resolve() error = %v", err)
				}
				if len(got) != 1 {
					t.Fatalf("resolved %d instances, want 1", len(got))
				}
				if got[0].SourceKind != KindAgent || got[0].SourceName != id {
					t.Fatalf("resolved to %s:%s, want agent:%s", got[0].SourceKind, got[0].SourceName, id)
				}
				if !bytes.Equal(got[0].SourceBytes, fixtureSource(id)) {
					t.Fatal("adapter source bytes do not come from adapters/")
				}
			})
		}
	})

	t.Run("vision maps to the canonical visual-inspector role", func(t *testing.T) {
		cfg := &Config{
			SchemaVersion: SchemaVersion,
			Instances:     map[string]Instance{"vision": {Source: goldenFleetMapping["vision"]}},
		}
		got, err := Resolve(fixturePack(), cfg)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if got[0].SourceName != "sdd-visual-inspector" {
			t.Fatalf("vision source = %s:%s, want agent:sdd-visual-inspector", got[0].SourceKind, got[0].SourceName)
		}

		// A vision adapter would be a second, divergent role definition. The
		// single mapping is the reviewed B6 decision: no vision.md exists.
		_, err = Resolve(fixturePack(), &Config{
			SchemaVersion: SchemaVersion,
			Instances:     map[string]Instance{"vision": {Source: "agent:vision"}},
		})
		if err == nil {
			t.Fatal("Resolve() accepted agent:vision, want unknown source")
		}
		wantErrorContains(t, err, "unknown source", "agent:vision")
	})

	t.Run("duplicate source names across roots are rejected", func(t *testing.T) {
		pack := fixturePack()
		const name = "sdd-lightweight"
		pack["adapters/"+name+".md"] = &fstest.MapFile{Data: fixtureSource(name)}
		_, err := Resolve(pack, &Config{
			SchemaVersion: SchemaVersion,
			Instances:     map[string]Instance{name: {Source: "agent:" + name}},
		})
		if err == nil {
			t.Fatal("Resolve() accepted a duplicate agent source in agents/ and adapters/")
		}
		wantErrorContains(t, err, "duplicate", "agent:"+name)
	})

	t.Run("wrong namespace is unknown", func(t *testing.T) {
		cfg := &Config{
			SchemaVersion: SchemaVersion,
			Instances: map[string]Instance{
				"sdd-implement-1": {Source: "agent:sdd-implement-1"},
			},
		}
		_, err := Resolve(fixturePack(), cfg)
		if err == nil {
			t.Fatal("Resolve() succeeded, want unknown source")
		}
		wantErrorContains(t, err, "unknown source", "agent:sdd-implement-1")
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("error does not wrap ErrInvalidConfig: %v", err)
		}
	})

	t.Run("alias to alias is rejected", func(t *testing.T) {
		cfg := &Config{
			SchemaVersion: SchemaVersion,
			Instances: map[string]Instance{
				"alias-one": {Source: "agent:alias-two"},
				"alias-two": {Source: "agent:real"},
			},
		}
		_, err := Resolve(fixturePack(), cfg)
		if err == nil {
			t.Fatal("Resolve() succeeded, want unknown source")
		}
		wantErrorContains(t, err, "unknown source", "agent:alias-two")
	})

	t.Run("cycle is rejected", func(t *testing.T) {
		cfg := &Config{
			SchemaVersion: SchemaVersion,
			Instances: map[string]Instance{
				"cycle-a": {Source: "agent:cycle-b"},
				"cycle-b": {Source: "agent:cycle-a"},
			},
		}
		_, err := Resolve(fixturePack(), cfg)
		if err == nil {
			t.Fatal("Resolve() succeeded, want unknown source")
		}
		wantErrorContains(t, err, "unknown source")
	})

	t.Run("model override preserves source bytes", func(t *testing.T) {
		pack := fixturePack()
		first := &Config{
			SchemaVersion: SchemaVersion,
			Instances: map[string]Instance{
				"sdd-lightweight": {Source: "agent:sdd-lightweight", Model: "opencode/deepseek-v4.1-flash-free"},
			},
		}
		second := &Config{
			SchemaVersion: SchemaVersion,
			Instances: map[string]Instance{
				"sdd-lightweight": {Source: "agent:sdd-lightweight", Model: "9router/ag/gemini-3.8-flash-high#high"},
			},
		}
		a, err := Resolve(pack, first)
		if err != nil {
			t.Fatalf("Resolve(first) error = %v", err)
		}
		b, err := Resolve(pack, second)
		if err != nil {
			t.Fatalf("Resolve(second) error = %v", err)
		}
		if !bytes.Equal(a[0].SourceBytes, b[0].SourceBytes) || a[0].SourceDigest != b[0].SourceDigest {
			t.Fatal("model-only change altered the canonical source bytes or digest")
		}
		if a[0].Model == b[0].Model {
			t.Fatal("model override was not carried into Resolved.Model")
		}
	})

	t.Run("enabled defaults true and explicit false is preserved", func(t *testing.T) {
		disabled := false
		cfg := &Config{
			SchemaVersion: SchemaVersion,
			Instances: map[string]Instance{
				"sdd-orchestrator": {Source: "agent:sdd-orchestrator"},
				"sdd-lightweight":  {Source: "agent:sdd-lightweight", Enabled: &disabled},
			},
		}
		got, err := Resolve(fixturePack(), cfg)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		byID := make(map[string]Resolved, len(got))
		for _, resolved := range got {
			byID[resolved.InstanceID] = resolved
		}
		if !byID["sdd-orchestrator"].Enabled {
			t.Fatal("omitted enabled must resolve to true")
		}
		if byID["sdd-lightweight"].Enabled {
			t.Fatal("explicit enabled false must resolve to false")
		}
		if byID["sdd-orchestrator"].Model != "" {
			t.Fatalf("orchestrator model = %q, want empty canonical default", byID["sdd-orchestrator"].Model)
		}
	})

	t.Run("missing source directories", func(t *testing.T) {
		t.Run("adapters", func(t *testing.T) {
			pack := fstest.MapFS{
				"agents/real.md":                        &fstest.MapFile{Data: fixtureSource("real")},
				"presets/free-first/sdd-implement-1.md": &fstest.MapFile{Data: fixtureSource("sdd-implement-1")},
			}
			_, err := Resolve(pack, &Config{
				SchemaVersion: SchemaVersion,
				Instances:     map[string]Instance{"a": {Source: "agent:real"}},
			})
			if err == nil {
				t.Fatal("Resolve() succeeded, want missing adapters directory error")
			}
			wantErrorContains(t, err, "adapters")
		})
		t.Run("presets", func(t *testing.T) {
			pack := fstest.MapFS{
				"agents/real.md":          &fstest.MapFile{Data: fixtureSource("real")},
				"adapters/sdd-spec-9r.md": &fstest.MapFile{Data: fixtureSource("sdd-spec-9r")},
			}
			_, err := Resolve(pack, &Config{
				SchemaVersion: SchemaVersion,
				Instances:     map[string]Instance{"a": {Source: "agent:real"}},
			})
			if err == nil {
				t.Fatal("Resolve() succeeded, want missing directory error")
			}
			wantErrorContains(t, err, "presets/free-first")
		})
	})

	t.Run("directory pack and special file refusal", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "agents"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "adapters"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "presets", "free-first"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "agents", "sdd-lightweight.md"), fixtureSource("sdd-lightweight"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "adapters", "sdd-spec-9r.md"), fixtureSource("sdd-spec-9r"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "presets", "free-first", "sdd-implement-1.md"), fixtureSource("sdd-implement-1"), 0644); err != nil {
			t.Fatal(err)
		}
		cfg := &Config{
			SchemaVersion: SchemaVersion,
			Instances: map[string]Instance{
				"sdd-lightweight": {Source: "agent:sdd-lightweight"},
				"sdd-spec-9r":     {Source: "agent:sdd-spec-9r"},
				"sdd-implement-1": {Source: "preset:sdd-implement-1"},
			},
		}
		got, err := Resolve(os.DirFS(dir), cfg)
		if err != nil {
			t.Fatalf("Resolve(directory pack) error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("resolved %d instances, want 3", len(got))
		}

		if err := os.Symlink(
			filepath.Join(dir, "agents", "sdd-lightweight.md"),
			filepath.Join(dir, "agents", "link.md"),
		); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		_, err = Resolve(os.DirFS(dir), &Config{
			SchemaVersion: SchemaVersion,
			Instances:     map[string]Instance{"link": {Source: "agent:link"}},
		})
		if err == nil {
			t.Fatal("Resolve() followed a symlinked source, want special-file error")
		}
		wantErrorContains(t, err, "special file")
	})
}

func TestFleetResolveRealPack(t *testing.T) {
	sources := []struct {
		name string
		repo string
	}{
		{name: "embedded", repo: ""},
		{name: "on-disk", repo: filepath.Join("..", "..")},
	}
	cfg := configFromMapping(goldenFleetMapping, nil)

	for _, source := range sources {
		t.Run(source.name, func(t *testing.T) {
			pack, err := skillsync.OpenCodePackFS(source.repo)
			if err != nil {
				t.Fatalf("open pack: %v", err)
			}
			got, err := Resolve(pack, cfg)
			if err != nil {
				t.Fatalf("Resolve(real pack) error = %v", err)
			}
			if len(got) != 18 {
				t.Fatalf("resolved %d real-pack instances, want 18", len(got))
			}
			byID := make(map[string]Resolved, len(got))
			for _, resolved := range got {
				byID[resolved.InstanceID] = resolved
				if len(resolved.SourceBytes) == 0 {
					t.Errorf("instance %q has empty source bytes", resolved.InstanceID)
				}
				if resolved.SourceDigest == "" {
					t.Errorf("instance %q has empty source digest", resolved.InstanceID)
				}

				// Every real-pack instance must render: canonical body and
				// permission set byte-identical, only model: changed.
				rendered, err := RenderResolved(resolved)
				if err != nil {
					t.Fatalf("instance %q: RenderResolved: %v", resolved.InstanceID, err)
				}
				if rendered.DesiredDigest != digestBytes(rendered.Bytes) {
					t.Errorf("instance %q: DesiredDigest is not the digest of rendered bytes", resolved.InstanceID)
				}
				equal, err := PermissionSetsEqual(resolved.SourceBytes, rendered.Bytes)
				if err != nil {
					t.Fatalf("instance %q: PermissionSetsEqual: %v", resolved.InstanceID, err)
				}
				if !equal {
					t.Errorf("instance %q: model render changed the permission set", resolved.InstanceID)
				}
				sourceParts, err := splitFrontmatter(resolved.SourceBytes)
				if err != nil {
					t.Fatalf("instance %q: split source: %v", resolved.InstanceID, err)
				}
				renderedParts, err := splitFrontmatter(rendered.Bytes)
				if err != nil {
					t.Fatalf("instance %q: split rendered: %v", resolved.InstanceID, err)
				}
				if !bytes.Equal(sourceParts.body, renderedParts.body) {
					t.Errorf("instance %q: model render changed the body", resolved.InstanceID)
				}
				sourceFields, sourceModels := frontmatterWithoutModel(sourceParts.fields)
				renderedFields, renderedModels := frontmatterWithoutModel(renderedParts.fields)
				if sourceModels != 1 || renderedModels != 1 {
					t.Errorf("instance %q: model fields = %d/%d, want 1/1", resolved.InstanceID, sourceModels, renderedModels)
				}
				if !bytes.Equal(sourceFields, renderedFields) {
					t.Errorf("instance %q: model render changed frontmatter outside model", resolved.InstanceID)
				}
			}

			// vision is one mapping to the canonical read-only inspector role,
			// not a second divergent vision prompt.
			if byID["vision"].SourceName != "sdd-visual-inspector" {
				t.Errorf("vision resolved to %s:%s, want agent:sdd-visual-inspector", byID["vision"].SourceKind, byID["vision"].SourceName)
			}
			visualBytes, err := fs.ReadFile(pack, "agents/sdd-visual-inspector.md")
			if err != nil {
				t.Fatalf("read canonical visual-inspector source: %v", err)
			}
			if !bytes.Equal(byID["vision"].SourceBytes, visualBytes) {
				t.Error("vision did not resolve to the canonical visual-inspector bytes")
			}
			for _, adapter := range fixtureAdapterNames {
				resolved, ok := byID[adapter]
				if !ok {
					t.Fatalf("adapter instance %q was not resolved", adapter)
				}
				if resolved.SourceKind != KindAgent || resolved.SourceName != adapter {
					t.Errorf("adapter %q resolved to %s:%s, want agent:%s", adapter, resolved.SourceKind, resolved.SourceName, adapter)
				}
				adapterBytes, err := fs.ReadFile(pack, "adapters/"+adapter+".md")
				if err != nil {
					t.Fatalf("read adapter source %q: %v", adapter, err)
				}
				if !bytes.Equal(resolved.SourceBytes, adapterBytes) {
					t.Errorf("adapter %q did not resolve from adapters/", adapter)
				}
				if !bytes.HasPrefix(resolved.SourceBytes, []byte("---\n")) {
					t.Errorf("adapter %q does not look like a profile", adapter)
				}
			}
		})
	}

	// The embedded and on-disk pack must serve identical adapter sources, so a
	// checkout and an installed binary cannot silently diverge.
	embedded, err := skillsync.OpenCodePackFS("")
	if err != nil {
		t.Fatalf("open embedded pack: %v", err)
	}
	onDisk, err := skillsync.OpenCodePackFS(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("open on-disk pack: %v", err)
	}
	for _, adapter := range fixtureAdapterNames {
		rel := "adapters/" + adapter + ".md"
		embeddedBytes, err := fs.ReadFile(embedded, rel)
		if err != nil {
			t.Fatalf("embedded read %s: %v", rel, err)
		}
		onDiskBytes, err := fs.ReadFile(onDisk, rel)
		if err != nil {
			t.Fatalf("on-disk read %s: %v", rel, err)
		}
		if !bytes.Equal(embeddedBytes, onDiskBytes) {
			t.Errorf("%s differs between the embedded and on-disk pack", rel)
		}
	}
}
