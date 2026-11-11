package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"strings"
)

// Resolved is one instance bound to its canonical source bytes. Model is the
// personal override (empty means "leave the canonical model untouched") and
// Enabled is the effective enablement after the default-true rule.
type Resolved struct {
	InstanceID   string
	SourceKind   string
	SourceName   string
	SourceBytes  []byte
	SourceDigest string
	Model        string
	Enabled      bool
}

// Resolve validates cfg, walks the allow-listed agents/, adapters/ and
// presets/free-first/ sources, and resolves every instance through exactly one
// hop. Adapters are reviewed fleet-only render sources that share the agent:
// namespace with canonical agents; a name may exist in exactly one of agents/
// and adapters/. A SourceRef must name a real walked file;
// instance-to-instance aliases, cycles and unknown sources are rejected.
func Resolve(pack fs.FS, cfg *Config) ([]Resolved, error) {
	if pack == nil {
		return nil, fmt.Errorf("fleet resolve: source pack is nil")
	}
	if cfg == nil {
		return nil, configError("config is nil")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	allow, err := walkSourceAllowList(pack)
	if err != nil {
		return nil, err
	}
	ids := sortedInstanceIDs(cfg.Instances)
	resolved := make([]Resolved, 0, len(ids))
	for _, id := range ids {
		inst := cfg.Instances[id]
		ref, err := parseSourceRef(inst.Source)
		if err != nil {
			return nil, configError("instance %q: %s", id, err)
		}
		rel, ok := allow[ref.String()]
		if !ok {
			return nil, configError("instance %q: unknown source %q", id, ref.String())
		}
		data, err := fs.ReadFile(pack, rel)
		if err != nil {
			return nil, fmt.Errorf("fleet resolve: instance %q: read source %q: %w", id, ref.String(), err)
		}
		resolved = append(resolved, Resolved{
			InstanceID:   id,
			SourceKind:   ref.Kind,
			SourceName:   ref.Name,
			SourceBytes:  data,
			SourceDigest: digestBytes(data),
			Model:        inst.Model,
			Enabled:      inst.IsEnabled(),
		})
	}
	return resolved, nil
}

func walkSourceAllowList(pack fs.FS) (map[string]string, error) {
	allow := make(map[string]string)
	roots := []struct {
		kind string
		dir  string
	}{
		{KindAgent, "agents"},
		{KindAgent, "adapters"},
		{KindPreset, "presets/free-first"},
	}
	for _, root := range roots {
		entries, err := fs.ReadDir(pack, root.dir)
		if err != nil {
			return nil, fmt.Errorf("fleet resolve: read %s sources: %w", root.dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}
			if !entry.Type().IsRegular() {
				return nil, fmt.Errorf("fleet resolve: refusing special file %q", root.dir+"/"+name)
			}
			base := strings.TrimSuffix(name, ".md")
			if !sourceNameRe.MatchString(base) {
				continue
			}
			key := root.kind + ":" + base
			if previous, ok := allow[key]; ok {
				return nil, fmt.Errorf("fleet resolve: duplicate source %q in %s and %s", key, previous, root.dir+"/"+name)
			}
			allow[key] = root.dir + "/" + name
		}
	}
	return allow, nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
