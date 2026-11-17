package fleet

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// RenderInput is one canonical profile plus the model selected for an
// instance. SourceBytes must be the exact bytes read from
// opencode-pack/agents/*.md, opencode-pack/adapters/*.md or
// opencode-pack/presets/free-first/*.md.
// An empty Model means "leave the canonical model untouched", which is the
// correct input for sdd-orchestrator and for instances that inherit.
type RenderInput struct {
	InstanceID  string
	SourceName  string
	SourceBytes []byte
	Model       string
}

// Rendered is the staged agent file. Bytes are the complete file to install;
// DesiredDigest is the SHA-256 of exactly those staged bytes (the plan's
// desiredDigest), never of the source or the destination.
type Rendered struct {
	InstanceID    string
	Bytes         []byte
	DesiredDigest string
	ModelChanged  bool
}

// Render starts from the canonical source bytes and changes only the
// single top-level model: scalar. The prompt body, every other frontmatter
// field and the permission block stay byte-identical. It fails closed if the
// state machine would touch anything else or if the post-render permission
// set differs from the source.
func Render(in RenderInput) (Rendered, error) {
	label := in.label()
	if len(in.SourceBytes) == 0 {
		return Rendered{}, fmt.Errorf("render %s: empty source bytes", label)
	}
	if err := validateModel(in.Model); err != nil {
		return Rendered{}, fmt.Errorf("render %s: %w", label, err)
	}
	parts, err := splitFrontmatter(in.SourceBytes)
	if err != nil {
		return Rendered{}, fmt.Errorf("render %s: %w", label, err)
	}
	fields, changed, err := replaceModel(parts.fields, parts.eol, in.Model)
	if err != nil {
		return Rendered{}, fmt.Errorf("render %s: %w", label, err)
	}
	out := joinFrontmatter(parts, fields)
	if err := VerifyModelOnly(in.SourceBytes, out); err != nil {
		return Rendered{}, fmt.Errorf("render %s: %w", label, err)
	}
	return Rendered{
		InstanceID:    in.InstanceID,
		Bytes:         out,
		DesiredDigest: digestBytes(out),
		ModelChanged:  changed,
	}, nil
}

func (in RenderInput) label() string {
	if in.InstanceID != "" {
		return in.InstanceID
	}
	if in.SourceName != "" {
		return in.SourceName
	}
	return "source"
}

// RenderResolved renders one instance resolved by the B1 resolver contract.
// It keeps the renderer independent of the resolver's walk while giving the
// plan/apply path one obvious bridge.
func RenderResolved(r Resolved) (Rendered, error) {
	return Render(RenderInput{
		InstanceID:  r.InstanceID,
		SourceName:  r.SourceName,
		SourceBytes: r.SourceBytes,
		Model:       r.Model,
	})
}

// frontmatterParts keeps the original byte spans so reassembly cannot
// rewrite the body or the delimiters.
type frontmatterParts struct {
	prefix []byte // opening --- line, including its line ending
	fields []byte // bytes between the delimiters
	close  []byte // closing --- line, including its line ending if present
	body   []byte // everything after the closing delimiter line
	eol    string // opening delimiter's line ending
}

func splitFrontmatter(data []byte) (frontmatterParts, error) {
	var parts frontmatterParts
	if len(data) == 0 {
		return parts, fmt.Errorf("frontmatter: empty file")
	}
	contentEnd, next, ok := lineAt(data, 0)
	if !ok || !isDelimiterLine(data[:contentEnd]) {
		return parts, fmt.Errorf("frontmatter: missing opening delimiter")
	}
	if next == contentEnd {
		return parts, fmt.Errorf("frontmatter: missing closing delimiter")
	}
	parts.eol = "\n"
	if contentEnd > 0 && data[contentEnd-1] == '\r' {
		parts.eol = "\r\n"
	}
	parts.prefix = data[:next]

	for start := next; ; {
		contentEnd, lineNext, ok := lineAt(data, start)
		if !ok {
			return parts, fmt.Errorf("frontmatter: missing closing delimiter")
		}
		if isDelimiterLine(data[start:contentEnd]) {
			parts.fields = data[next:start]
			parts.close = data[start:lineNext]
			parts.body = data[lineNext:]
			return parts, nil
		}
		if lineNext == contentEnd {
			return parts, fmt.Errorf("frontmatter: missing closing delimiter")
		}
		start = lineNext
	}
}

// lineAt returns the end of the line's content and the start of the next
// line. contentEnd points at \r or \n (or len(data)) and next is after the
// line ending, so callers can preserve CRLF exactly.
func lineAt(data []byte, start int) (contentEnd, next int, ok bool) {
	if start > len(data) {
		return 0, 0, false
	}
	i := start
	for i < len(data) && data[i] != '\n' {
		i++
	}
	contentEnd = i
	if i < len(data) {
		return contentEnd, i + 1, true
	}
	return contentEnd, contentEnd, true
}

func isDelimiterLine(content []byte) bool {
	return bytes.Equal(bytes.TrimSuffix(content, []byte("\r")), []byte("---"))
}

func joinFrontmatter(parts frontmatterParts, fields []byte) []byte {
	out := make([]byte, 0, len(parts.prefix)+len(fields)+len(parts.close)+len(parts.body))
	out = append(out, parts.prefix...)
	out = append(out, fields...)
	out = append(out, parts.close...)
	out = append(out, parts.body...)
	return out
}

type fmLine struct {
	start      int
	contentEnd int
	end        int
}

func scanFMLines(fields []byte) []fmLine {
	var lines []fmLine
	for start := 0; start < len(fields); {
		contentEnd, next, _ := lineAt(fields, start)
		lines = append(lines, fmLine{start: start, contentEnd: contentEnd, end: next})
		start = next
	}
	return lines
}

// replaceModel rewrites exactly one top-level model: scalar. A missing field
// is inserted after mode: (or at the end of the frontmatter), using the
// file's own line ending. An equal value is left untouched so a re-render is
// byte-identical.
func replaceModel(fields []byte, eol, desired string) ([]byte, bool, error) {
	if desired == "" {
		return fields, false, nil
	}
	lines := scanFMLines(fields)
	modelIdx, modeIdx := -1, -1
	for i, line := range lines {
		content := bytes.TrimSuffix(fields[line.start:line.contentEnd], []byte("\r"))
		key, value, ok := topLevelKey(content)
		if !ok {
			continue
		}
		switch key {
		case "model":
			if modelIdx >= 0 {
				return nil, false, fmt.Errorf("frontmatter: duplicate model field")
			}
			modelIdx = i
			val := strings.TrimSpace(value)
			if strings.HasPrefix(val, "|") || strings.HasPrefix(val, ">") {
				return nil, false, fmt.Errorf("frontmatter: block scalar not supported for model")
			}
			if scalarValue(value) == desired {
				return fields, false, nil
			}
		case "mode":
			if modeIdx < 0 {
				modeIdx = i
			}
		}
	}

	if modelIdx >= 0 {
		line := lines[modelIdx]
		out := make([]byte, 0, len(fields)+len(desired)+8)
		out = append(out, fields[:line.start]...)
		out = append(out, "model: "...)
		out = append(out, desired...)
		out = append(out, fmLineEnding(fields, line)...)
		out = append(out, fields[line.end:]...)
		return out, true, nil
	}

	insertAt := len(fields)
	if modeIdx >= 0 {
		insertAt = lines[modeIdx].end
	}
	out := make([]byte, 0, len(fields)+len(desired)+10)
	out = append(out, fields[:insertAt]...)
	if insertAt > 0 && fields[insertAt-1] != '\n' {
		out = append(out, eol...)
	}
	out = append(out, "model: "...)
	out = append(out, desired...)
	out = append(out, eol...)
	out = append(out, fields[insertAt:]...)
	return out, true, nil
}

func topLevelKey(content []byte) (key, value string, ok bool) {
	if len(content) == 0 || content[0] == ' ' || content[0] == '\t' || content[0] == '#' {
		return "", "", false
	}
	i := bytes.IndexByte(content, ':')
	if i <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(string(content[:i]))
	if key == "" {
		return "", "", false
	}
	return key, string(content[i+1:]), true
}

// scalarValue removes surrounding quotes and a trailing YAML comment. A #
// that is part of the scalar (as in model#variant) is preserved.
func scalarValue(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		quote := s[0]
		for i := 1; i < len(s); i++ {
			if s[i] == quote {
				return s[1:i]
			}
		}
		return s[1:]
	}
	if i := strings.IndexByte(s, '#'); i >= 0 {
		if i == 0 {
			return ""
		}
		if s[i-1] == ' ' || s[i-1] == '\t' {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

// permissionRule is one normalized { action, resource, effect } entry.
type permissionRule struct {
	Action   string
	Resource string
	Effect   string
}

// VerifyModelOnly proves that rendered differs from source only in the
// top-level model: scalar: the body is byte-identical, the rest of the
// frontmatter is byte-identical once model lines are removed, and the
// normalized permission set is unchanged. Any ambiguity fails closed.
func VerifyModelOnly(source, rendered []byte) error {
	sourceParts, err := splitFrontmatter(source)
	if err != nil {
		return fmt.Errorf("source %w", err)
	}
	renderedParts, err := splitFrontmatter(rendered)
	if err != nil {
		return fmt.Errorf("rendered %w", err)
	}
	if !bytes.Equal(sourceParts.body, renderedParts.body) {
		return fmt.Errorf("render: body bytes changed")
	}
	sourceFields, sourceModels := frontmatterWithoutModel(sourceParts.fields)
	renderedFields, renderedModels := frontmatterWithoutModel(renderedParts.fields)
	if sourceModels > 1 || renderedModels > 1 {
		return fmt.Errorf("frontmatter: duplicate model field")
	}
	sourceRules, err := parsePermissions(sourceParts.fields)
	if err != nil {
		return fmt.Errorf("source %w", err)
	}
	renderedRules, err := parsePermissions(renderedParts.fields)
	if err != nil {
		return fmt.Errorf("rendered %w", err)
	}
	sourceNormalized, err := normalizePermissionRules(sourceRules)
	if err != nil {
		return fmt.Errorf("source %w", err)
	}
	renderedNormalized, err := normalizePermissionRules(renderedRules)
	if err != nil {
		return fmt.Errorf("rendered %w", err)
	}
	if !equalPermissionRules(sourceNormalized, renderedNormalized) {
		return fmt.Errorf("render: permission set changed")
	}
	if !bytes.Equal(sourceFields, renderedFields) {
		return fmt.Errorf("render: frontmatter changed outside model")
	}
	return nil
}

// fmLineEnding returns the line's original ending, including a CR that
// belongs to a CRLF pair.
func fmLineEnding(fields []byte, line fmLine) []byte {
	start := line.contentEnd
	if start > line.start && fields[start-1] == '\r' {
		start--
	}
	return fields[start:line.end]
}

func frontmatterWithoutModel(fields []byte) ([]byte, int) {
	var out []byte
	models := 0
	for _, line := range scanFMLines(fields) {
		content := bytes.TrimSuffix(fields[line.start:line.contentEnd], []byte("\r"))
		if key, _, ok := topLevelKey(content); ok && key == "model" {
			models++
			continue
		}
		out = append(out, fields[line.start:line.end]...)
	}
	return out, models
}

// PermissionSetsEqual compares the normalized permission sets of two agent
// files. Order, quoting and CRLF/LF differences are ignored; any action,
// resource or effect delta is a difference.
func PermissionSetsEqual(a, b []byte) (bool, error) {
	aParts, err := splitFrontmatter(a)
	if err != nil {
		return false, err
	}
	bParts, err := splitFrontmatter(b)
	if err != nil {
		return false, err
	}
	aRules, err := parsePermissions(aParts.fields)
	if err != nil {
		return false, err
	}
	bRules, err := parsePermissions(bParts.fields)
	if err != nil {
		return false, err
	}
	aNormalized, err := normalizePermissionRules(aRules)
	if err != nil {
		return false, err
	}
	bNormalized, err := normalizePermissionRules(bRules)
	if err != nil {
		return false, err
	}
	return equalPermissionRules(aNormalized, bNormalized), nil
}

// PermissionDigest is a deterministic digest of the normalized permission
// set. Plans can bind it without copying rule text.
func PermissionDigest(data []byte) (string, error) {
	parts, err := splitFrontmatter(data)
	if err != nil {
		return "", err
	}
	rules, err := parsePermissions(parts.fields)
	if err != nil {
		return "", err
	}
	normalized, err := normalizePermissionRules(rules)
	if err != nil {
		return "", err
	}
	return digestPermissionRules(normalized), nil
}

// parsePermissions reads the pack's fixed YAML subset: a top-level
// permissions: key followed by list items with action and indented resource
// and effect fields. It rejects anything it does not understand so a policy
// change cannot be mistaken for a model-only render.
func parsePermissions(fields []byte) ([]permissionRule, error) {
	var rules []permissionRule
	inPermissions := false
	for _, line := range scanFMLines(fields) {
		content := bytes.TrimSuffix(fields[line.start:line.contentEnd], []byte("\r"))
		trimmed := strings.TrimSpace(string(content))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indented := content[0] == ' ' || content[0] == '\t'
		if strings.HasPrefix(trimmed, "- ") {
			if !inPermissions {
				return nil, fmt.Errorf("permission list item outside permissions block")
			}
			key, value, ok := parseScalar(strings.TrimPrefix(trimmed, "- "))
			if !ok || key != "action" {
				return nil, fmt.Errorf("malformed permission action")
			}
			rules = append(rules, permissionRule{Action: value})
			continue
		}
		key, value, ok := parseScalar(trimmed)
		if !ok {
			return nil, fmt.Errorf("unparsable frontmatter line")
		}
		if !indented {
			inPermissions = key == "permissions"
			continue
		}
		if !inPermissions || len(rules) == 0 {
			return nil, fmt.Errorf("unexpected indented frontmatter field %q", key)
		}
		switch key {
		case "resource":
			rules[len(rules)-1].Resource = value
		case "effect":
			rules[len(rules)-1].Effect = value
		default:
			return nil, fmt.Errorf("unexpected permission field %q", key)
		}
	}
	for i, rule := range rules {
		if rule.Action == "" {
			return nil, fmt.Errorf("permission rule %d missing action", i+1)
		}
		if rule.Resource == "" {
			return nil, fmt.Errorf("permission rule %d missing resource", i+1)
		}
		if rule.Effect == "" {
			return nil, fmt.Errorf("permission rule %d missing effect", i+1)
		}
	}
	return rules, nil
}

func parseScalar(text string) (key, value string, ok bool) {
	i := strings.IndexByte(text, ':')
	if i <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(text[:i])
	if key == "" {
		return "", "", false
	}
	val := strings.TrimSpace(text[i+1:])
	if strings.HasPrefix(val, "|") || strings.HasPrefix(val, ">") {
		return "", "", false
	}
	return key, scalarValue(val), true
}

// ExtractModelFromContent parses frontmatter from content and returns the top-level model scalar.
func ExtractModelFromContent(content []byte) string {
	parts, err := splitFrontmatter(content)
	if err != nil {
		return ""
	}
	for _, line := range scanFMLines(parts.fields) {
		c := bytes.TrimSuffix(parts.fields[line.start:line.contentEnd], []byte("\r"))
		if key, val, ok := topLevelKey(c); ok && key == "model" {
			return scalarValue(val)
		}
	}
	return ""
}

func normalizePermissionRules(rules []permissionRule) ([]permissionRule, error) {
	seen := map[string]string{}
	out := make([]permissionRule, 0, len(rules))
	for _, rule := range rules {
		normalized := permissionRule{
			Action:   strings.ToLower(strings.TrimSpace(rule.Action)),
			Resource: strings.TrimSpace(rule.Resource),
			Effect:   strings.ToLower(strings.TrimSpace(rule.Effect)),
		}
		key := normalized.Action + "\x00" + normalized.Resource
		if previous, ok := seen[key]; ok {
			if previous != normalized.Effect {
				return nil, fmt.Errorf("conflicting permission rules for %s %s", normalized.Action, normalized.Resource)
			}
			continue
		}
		seen[key] = normalized.Effect
		out = append(out, normalized)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Action != out[j].Action {
			return out[i].Action < out[j].Action
		}
		if out[i].Resource != out[j].Resource {
			return out[i].Resource < out[j].Resource
		}
		return out[i].Effect < out[j].Effect
	})
	return out, nil
}

func equalPermissionRules(a, b []permissionRule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func digestPermissionRules(rules []permissionRule) string {
	h := sha256.New()
	for _, rule := range rules {
		h.Write([]byte(rule.Action))
		h.Write([]byte{0})
		h.Write([]byte(rule.Resource))
		h.Write([]byte{0})
		h.Write([]byte(rule.Effect))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}
