package fleet

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"agentic-sdd/internal/skillsync"
)

// File preview operations. A preview never carries an approval; a separate
// preview token and a human confirmation guard any later write.
const (
	FileOpCreate   = "create"
	FileOpUpdate   = "update"
	FileOpRemove   = "remove"
	FileOpNoOp     = "noop"
	FileOpConflict = "conflict"
)

// FilePreview is one read-only file projection: the bytes currently on disk
// and the bytes the operation would produce. Old and new digests are SHA-256
// of exactly those byte slices. Removed marks a managed removal and Conflict
// marks a proposal that needs a human decision; neither is resolved here.
type FilePreview struct {
	InstanceID string
	Path       string
	Op         string
	Reason     string
	OldBytes   []byte
	NewBytes   []byte
	OldDigest  string
	NewDigest  string
	Removed    bool
	Conflict   bool
}

// EncodeConfig serializes a fleet Config deterministically: struct field order
// for the document, sorted instance keys, two-space indentation and exactly one
// trailing newline. It performs no validation and no filesystem access; callers
// that persist the result must parse it strictly first.
func EncodeConfig(c *Config) ([]byte, error) {
	if c == nil {
		return nil, configError("config is nil")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode fleet config: %w", err)
	}
	return append(data, '\n'), nil
}

// PreviewSource is the read-only personal-source preview against the embedded
// canonical pack. It returns the validated, deterministically encoded new
// bytes; the parsed old bytes and its digest are the compare-and-swap base.
//
// It refuses a non-regular or symlinked current file, bounds both sides at
// MaxConfigBytes, parses the new bytes strictly, resolves every source and
// renders every instance so an undeployable draft cannot look ready. Nothing is
// written.
func PreviewSource(path string, newBytes []byte) (FilePreview, error) {
	return PreviewSourceWithRepo(path, newBytes, "")
}

// PreviewSourceWithRepo is PreviewSource with an explicit canonical pack root.
// An empty repo selects the embedded pack, matching the default CLI path.
func PreviewSourceWithRepo(path string, newBytes []byte, repo string) (FilePreview, error) {
	if path == "" {
		return FilePreview{}, configError("config path is required")
	}
	if len(newBytes) > MaxConfigBytes {
		return FilePreview{}, configError("config exceeds %d bytes", MaxConfigBytes)
	}
	oldBytes, oldDigest, exists, err := readSourceConfigBytes(path)
	if err != nil {
		return FilePreview{}, err
	}
	cfg, err := parseConfigBytes(newBytes)
	if err != nil {
		return FilePreview{}, err
	}
	normalized, err := EncodeConfig(cfg)
	if err != nil {
		return FilePreview{}, err
	}
	absRepo := ""
	if repo != "" {
		absRepo, err = filepath.Abs(repo)
		if err != nil {
			return FilePreview{}, err
		}
	}
	pack, err := skillsync.OpenCodePackFS(absRepo)
	if err != nil {
		return FilePreview{}, fmt.Errorf("open canonical OpenCode pack: %w", err)
	}
	resolved, err := Resolve(pack, cfg)
	if err != nil {
		return FilePreview{}, err
	}
	for _, r := range resolved {
		if _, err := RenderResolved(r); err != nil {
			return FilePreview{}, err
		}
	}
	preview := FilePreview{
		Path:      path,
		OldBytes:  oldBytes,
		NewBytes:  normalized,
		OldDigest: oldDigest,
		NewDigest: digestBytes(normalized),
		Op:        FileOpUpdate,
	}
	switch {
	case !exists:
		preview.Op = FileOpCreate
	case preview.OldDigest == preview.NewDigest:
		preview.Op = FileOpNoOp
	}
	return preview, nil
}

// readSourceConfigBytes reads the current personal-source bytes without
// following a symlink and without exceeding MaxConfigBytes.
func readSourceConfigBytes(path string) ([]byte, string, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", false, nil
		}
		return nil, "", false, err
	}
	if !info.Mode().IsRegular() {
		return nil, "", false, fmt.Errorf("%w: refusing non-regular file %s", ErrUnsafeTarget, filepath.Clean(path))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", false, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxConfigBytes+1))
	if err != nil {
		return nil, "", false, fmt.Errorf("read config %s: %w", filepath.Clean(path), err)
	}
	if len(data) > MaxConfigBytes {
		return nil, "", false, configError("config exceeds %d bytes", MaxConfigBytes)
	}
	return data, digestBytes(data), true, nil
}

// PlanFiles projects every proposed operation of a plan as old-versus-desired
// bytes. It rehydrates the config from plan.ConfigPath and the canonical pack
// from plan.RepoPath, then reuses the pure apply-time validators so a plan that
// no longer matches the source/config/render is reported stale instead of
// drawn against unrelated state.
//
// It never recomputes or reseals the plan identity, never writes, never
// acquires the engine lock and never touches Git. Removals come first, then
// conflicts, then routine updates, creates and no-ops.
func PlanFiles(plan *Plan) ([]FilePreview, error) {
	if plan == nil {
		return nil, fmt.Errorf("fleet plan files: plan is nil")
	}
	if !plan.VerifyPlanID() {
		return nil, fmt.Errorf("%w: plan ID mismatch (corrupted or altered plan)", ErrStalePlan)
	}
	if plan.RendererVersion != RendererVersion {
		return nil, fmt.Errorf("%w: renderer version changed (plan %s, current %s)", ErrStalePlan, plan.RendererVersion, RendererVersion)
	}
	if plan.Scope != "" && plan.Scope != ScopeGlobal {
		return nil, fmt.Errorf("%w: unsupported scope %q", ErrStalePlan, plan.Scope)
	}
	if plan.ConfigPath == "" {
		return nil, fmt.Errorf("%w: plan has no config path; regenerate it", ErrStalePlan)
	}
	if !filepath.IsAbs(plan.TargetRoot) {
		return nil, fmt.Errorf("%w: plan target root is not absolute: %s", ErrStalePlan, filepath.Clean(plan.TargetRoot))
	}
	cfg, _, configDigest, err := LoadConfigFile(plan.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("%w: reload configuration: %v", ErrStalePlan, err)
	}
	if configDigest != plan.ConfigDigest {
		return nil, fmt.Errorf("%w: configuration changed (plan %s, current %s)", ErrStalePlan, plan.ConfigDigest, configDigest)
	}
	pack, err := skillsync.OpenCodePackFS(plan.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("fleet plan files: open canonical OpenCode pack: %w", err)
	}
	resolved, err := Resolve(pack, cfg)
	if err != nil {
		return nil, err
	}
	if got := sourceRevision(resolved); got != plan.SourceRev {
		return nil, fmt.Errorf("%w: canonical source changed (plan %s, current %s)", ErrStalePlan, plan.SourceRev, got)
	}
	rendered := make(map[string]Rendered, len(resolved))
	for _, r := range resolved {
		if !r.Enabled {
			continue
		}
		rendered[r.InstanceID], err = RenderResolved(r)
		if err != nil {
			return nil, err
		}
	}
	opByFile, err := indexPlanOps(plan)
	if err != nil {
		return nil, err
	}
	if err := validatePlanSymmetry(plan, resolved, rendered, opByFile); err != nil {
		return nil, err
	}

	previews := make([]FilePreview, 0, len(plan.ProposedOps))
	for _, op := range plan.ProposedOps {
		switch op.Op {
		case FileOpCreate, FileOpUpdate, FileOpRemove, FileOpNoOp, FileOpConflict:
		default:
			return nil, fmt.Errorf("%w: unknown plan operation %q for %s", ErrPlanConflict, op.Op, op.File)
		}
		target, err := targetPathFor(plan.TargetRoot, op.File)
		if err != nil {
			return nil, err
		}
		oldBytes, oldDigest, _, err := readTargetBytes(target)
		if err != nil {
			return nil, err
		}
		preview := FilePreview{
			InstanceID: op.InstanceID,
			Path:       target,
			Op:         op.Op,
			Reason:     op.Reason,
			OldBytes:   oldBytes,
			OldDigest:  oldDigest,
			Removed:    op.Op == FileOpRemove,
			Conflict:   op.Op == FileOpConflict,
		}
		if out, ok := rendered[op.InstanceID]; ok && op.Op != FileOpRemove {
			preview.NewBytes = out.Bytes
			preview.NewDigest = out.DesiredDigest
		}
		previews = append(previews, preview)
	}
	sort.SliceStable(previews, func(i, j int) bool {
		ri, rj := fileOpRank(previews[i].Op), fileOpRank(previews[j].Op)
		if ri != rj {
			return ri < rj
		}
		return previews[i].Path < previews[j].Path
	})
	return previews, nil
}

// readTargetBytes reads a managed target without following a symlink. A
// missing file reports (nil, "", false, nil); a special file is refused.
func readTargetBytes(path string) ([]byte, string, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", false, nil
		}
		return nil, "", false, err
	}
	if !info.Mode().IsRegular() {
		return nil, "", false, fmt.Errorf("%w: refusing special file or symlink %s", ErrUnsafeTarget, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", false, err
	}
	return data, digestBytes(data), true, nil
}

func fileOpRank(op string) int {
	switch op {
	case FileOpRemove:
		return 0
	case FileOpConflict:
		return 1
	case FileOpUpdate:
		return 2
	case FileOpCreate:
		return 3
	case FileOpNoOp:
		return 4
	default:
		return 5
	}
}

// DiffKind labels one bounded line-diff row.
type DiffKind string

const (
	// DiffEqual is a line present in both sides.
	DiffEqual DiffKind = "equal"
	// DiffAdd is a line only in the new side.
	DiffAdd DiffKind = "add"
	// DiffDel is a line only in the old side.
	DiffDel DiffKind = "del"
	// DiffOmitted marks that the bounded output omitted further rows.
	DiffOmitted DiffKind = "omitted"
)

// DiffLine is one stable diff row. OldLine and NewLine are 1-based line
// numbers on their own side; the other side is 0 when the row does not exist
// there. Text is the line content without its line ending, clipped at
// MaxDiffLineBytes at a rune boundary.
type DiffLine struct {
	Kind    DiffKind
	OldLine int
	NewLine int
	Text    string
}

const (
	// MaxDiffLines bounds the number of rows LineDiff returns.
	MaxDiffLines = 20000
	// MaxDiffLineBytes bounds one rendered line's text.
	MaxDiffLineBytes = 2048
	// maxDiffLCSCells bounds the exact-diff work; beyond it the middle is
	// reported as a bounded replacement block instead of an O(n*m) table.
	maxDiffLCSCells = 1 << 20
)

// LineDiff computes a bounded, pure line diff between two byte slices. It
// ignores CRLF versus LF differences, preserves Unicode text, clips over-long
// lines and never allocates an unbounded table: common prefixes and suffixes
// are trimmed first, the middle uses an exact LCS only within a fixed cell
// budget, and the emitted rows are capped at MaxDiffLines.
func LineDiff(old, new []byte) []DiffLine {
	oldLines := splitDiffLines(old)
	newLines := splitDiffLines(new)

	prefix := 0
	for prefix < len(oldLines) && prefix < len(newLines) && oldLines[prefix] == newLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(oldLines)-prefix && suffix < len(newLines)-prefix &&
		oldLines[len(oldLines)-1-suffix] == newLines[len(newLines)-1-suffix] {
		suffix++
	}

	emit := &diffEmitter{}
	for i := 0; i < prefix; i++ {
		emit.add(DiffEqual, i+1, i+1, oldLines[i])
	}
	middleOld := oldLines[prefix : len(oldLines)-suffix]
	middleNew := newLines[prefix : len(newLines)-suffix]
	for _, op := range diffMiddle(middleOld, middleNew) {
		switch op.kind {
		case DiffEqual:
			emit.add(DiffEqual, prefix+op.oldIndex+1, prefix+op.newIndex+1, op.text)
		case DiffDel:
			emit.add(DiffDel, prefix+op.oldIndex+1, 0, op.text)
		case DiffAdd:
			emit.add(DiffAdd, 0, prefix+op.newIndex+1, op.text)
		}
	}
	for i := 0; i < suffix; i++ {
		oldIndex := len(oldLines) - suffix + i
		newIndex := len(newLines) - suffix + i
		emit.add(DiffEqual, oldIndex+1, newIndex+1, oldLines[oldIndex])
	}
	emit.finish()
	return emit.lines
}

type diffEmitter struct {
	lines   []DiffLine
	omitted int
}

func (e *diffEmitter) add(kind DiffKind, oldLine, newLine int, text string) {
	if len(e.lines) >= MaxDiffLines {
		e.omitted++
		return
	}
	e.lines = append(e.lines, DiffLine{Kind: kind, OldLine: oldLine, NewLine: newLine, Text: clipDiffLine(text)})
}

func (e *diffEmitter) finish() {
	if e.omitted == 0 {
		return
	}
	e.lines = append(e.lines, DiffLine{
		Kind: DiffOmitted,
		Text: fmt.Sprintf("... %d more diff line(s) omitted (limit %d)", e.omitted, MaxDiffLines),
	})
}

func clipDiffLine(text string) string {
	if len(text) <= MaxDiffLineBytes {
		return text
	}
	cut := MaxDiffLineBytes
	for cut > 0 && cut < len(text) && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "…"
}

// splitDiffLines splits on LF and strips one trailing CR per line, so a CRLF
// and an LF file with the same text compare equal.
func splitDiffLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	text := strings.TrimSuffix(string(data), "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

type diffOp struct {
	kind     DiffKind
	oldIndex int
	newIndex int
	text     string
}

// diffMiddle diffs two already-trimmed line ranges. Above the cell budget it
// emits every old line as a deletion followed by every new line as an
// addition: still complete and bounded, just without LCS alignment.
func diffMiddle(a, b []string) []diffOp {
	if len(a) == 0 {
		ops := make([]diffOp, 0, len(b))
		for j, line := range b {
			ops = append(ops, diffOp{kind: DiffAdd, newIndex: j, text: line})
		}
		return ops
	}
	if len(b) == 0 {
		ops := make([]diffOp, 0, len(a))
		for i, line := range a {
			ops = append(ops, diffOp{kind: DiffDel, oldIndex: i, text: line})
		}
		return ops
	}
	if len(a) > maxDiffLCSCells/len(b) {
		ops := make([]diffOp, 0, len(a)+len(b))
		for i, line := range a {
			ops = append(ops, diffOp{kind: DiffDel, oldIndex: i, text: line})
		}
		for j, line := range b {
			ops = append(ops, diffOp{kind: DiffAdd, newIndex: j, text: line})
		}
		return ops
	}

	cols := len(b) + 1
	dp := make([]int32, (len(a)+1)*cols)
	at := func(i, j int) int32 { return dp[i*cols+j] }
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				dp[i*cols+j] = at(i+1, j+1) + 1
			case at(i+1, j) >= at(i, j+1):
				dp[i*cols+j] = at(i+1, j)
			default:
				dp[i*cols+j] = at(i, j+1)
			}
		}
	}

	ops := make([]diffOp, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{kind: DiffEqual, oldIndex: i, newIndex: j, text: a[i]})
			i++
			j++
		case at(i+1, j) >= at(i, j+1):
			ops = append(ops, diffOp{kind: DiffDel, oldIndex: i, text: a[i]})
			i++
		default:
			ops = append(ops, diffOp{kind: DiffAdd, newIndex: j, text: b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		ops = append(ops, diffOp{kind: DiffDel, oldIndex: i, text: a[i]})
	}
	for ; j < len(b); j++ {
		ops = append(ops, diffOp{kind: DiffAdd, newIndex: j, text: b[j]})
	}
	return ops
}
