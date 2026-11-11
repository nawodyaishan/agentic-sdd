package fleet

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"agentic-sdd/internal/skillsync"
)

var (
	ErrLockHeld             = errors.New("fleet lock is held by another process")
	ErrStalePlan            = errors.New("stale plan")
	ErrInvalidPlanStructure = errors.New("invalid plan structure")
)

const (
	// LockFileName is the stable lock inode inside the backup root.
	LockFileName = ".fleet.lock"

	// LockStaleTimeout documents the diagnostic age after which a recorded
	// lock looks abandoned. Ownership itself is enforced by the operating
	// system, so the age alone never decides whether a lock is reclaimable.
	LockStaleTimeout = 30 * time.Second

	// MaxPlanBytes bounds a plan file read at 4 MiB.
	MaxPlanBytes = 4 << 20

	// RendererVersion identifies the rendering algorithm in plan and
	// provenance records. Bump it whenever a renderer change can alter the
	// bytes installed for the same source and config.
	RendererVersion = "1"
)

// ProposedOp describes a single proposed change to an agent profile file.
type ProposedOp struct {
	InstanceID        string `json:"instanceId"`
	File              string `json:"file"`
	Op                string `json:"op"` // "create", "update", "remove", "noop", "conflict"
	SourceRef         string `json:"sourceRef,omitempty"`
	SourceDigest      string `json:"sourceDigest,omitempty"`
	Model             string `json:"model,omitempty"`
	PriorModel        string `json:"priorModel,omitempty"`
	DesiredDigest     string `json:"desiredDigest,omitempty"`
	CurrentDigest     string `json:"currentDigest,omitempty"`
	LastAppliedDigest string `json:"lastAppliedDigest,omitempty"`
	Provenance        string `json:"provenance,omitempty"`
	Adopt             bool   `json:"adopt,omitempty"`
	Reason            string `json:"reason,omitempty"`
}

// Plan binds the canonical inputs, targets, and proposed operations for a fleet transaction.
type Plan struct {
	PlanID             string            `json:"planId"`
	SourceRev          string            `json:"sourceRev"`
	ConfigDigest       string            `json:"configDigest"`
	ConfigPath         string            `json:"configPath,omitempty"`
	RepoPath           string            `json:"repoPath,omitempty"`
	Scope              string            `json:"scope,omitempty"`
	RendererVersion    string            `json:"rendererVersion"`
	TargetRoot         string            `json:"targetRoot"`
	FileList           []string          `json:"fileList"`
	LastKnownInstalled map[string]string `json:"lastKnownInstalled"`
	ProposedOps        []ProposedOp      `json:"proposedOps"`
	PermNotes          []string          `json:"permNotes"`
}

// canonicalPayload enforces the exact field order:
// sourceRev, configDigest, rendererVersion, targetRoot, fileList, lastKnownInstalled, proposedOps, permNotes.
// Machine-specific paths (ConfigPath, RepoPath) are excluded so planID is deterministic
// across environments with identical content and configuration.
type canonicalPayload struct {
	SourceRev          string            `json:"sourceRev"`
	ConfigDigest       string            `json:"configDigest"`
	RendererVersion    string            `json:"rendererVersion"`
	TargetRoot         string            `json:"targetRoot"`
	FileList           []string          `json:"fileList"`
	LastKnownInstalled map[string]string `json:"lastKnownInstalled"`
	ProposedOps        []ProposedOp      `json:"proposedOps"`
	PermNotes          []string          `json:"permNotes"`
}

// CanonicalBytes returns the canonical, deterministic JSON representation of the plan
// (sorted keys, explicit field order, normalized slices) used to compute the planID.
func CanonicalBytes(p *Plan) ([]byte, error) {
	fileList := make([]string, len(p.FileList))
	copy(fileList, p.FileList)
	sort.Strings(fileList)

	ops := make([]ProposedOp, len(p.ProposedOps))
	copy(ops, p.ProposedOps)
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].File != ops[j].File {
			return ops[i].File < ops[j].File
		}
		return ops[i].InstanceID < ops[j].InstanceID
	})

	permNotes := make([]string, len(p.PermNotes))
	copy(permNotes, p.PermNotes)
	sort.Strings(permNotes)

	installed := make(map[string]string, len(p.LastKnownInstalled))
	for k, v := range p.LastKnownInstalled {
		installed[k] = v
	}

	payload := canonicalPayload{
		SourceRev:          p.SourceRev,
		ConfigDigest:       p.ConfigDigest,
		RendererVersion:    p.RendererVersion,
		TargetRoot:         p.TargetRoot,
		FileList:           fileList,
		LastKnownInstalled: installed,
		ProposedOps:        ops,
		PermNotes:          permNotes,
	}

	return json.Marshal(payload)
}

// ComputePlanID calculates the SHA-256 hash of the canonical plan payload.
func ComputePlanID(p *Plan) (string, error) {
	b, err := CanonicalBytes(p)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Seal computes and sets the planID on the Plan.
func (p *Plan) Seal() error {
	id, err := ComputePlanID(p)
	if err != nil {
		return err
	}
	p.PlanID = id
	return nil
}

// VerifyPlanID verifies that the planID matches the recomputed canonical SHA-256 hash.
func (p *Plan) VerifyPlanID() bool {
	if p.PlanID == "" {
		return false
	}
	id, err := ComputePlanID(p)
	if err != nil {
		return false
	}
	return p.PlanID == id
}

// ToJSON serializes the plan with indentation for output or saving.
func (p *Plan) ToJSON() ([]byte, error) {
	if p.PlanID == "" {
		if err := p.Seal(); err != nil {
			return nil, err
		}
	}
	return json.MarshalIndent(p, "", "  ")
}

// SavePlan writes the canonical plan JSON to path with 0600 permissions.
// The directory is created with 0700 permissions.
// Machine-specific paths (ConfigPath/RepoPath) are omitted from planID computation so identity
// depends only on content and target specifications, not environment paths.
// Plans may contain sensitive local path details and are saved with 0600 permissions.
func SavePlan(p *Plan, path string) error {
	data, err := p.ToJSON()
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadPlan reads and unmarshals a plan JSON, verifying its planID integrity.
// It rejects non-regular files and enforces a 4 MiB limit via LimitReader.
func LoadPlan(path string) (*Plan, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: refusing non-regular plan file %s", ErrInvalidPlanStructure, filepath.Clean(path))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, MaxPlanBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read plan %s: %w", filepath.Clean(path), err)
	}
	if len(data) > MaxPlanBytes {
		return nil, fmt.Errorf("%w: plan exceeds %d bytes", ErrInvalidPlanStructure, MaxPlanBytes)
	}

	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("%w: invalid plan JSON: %v", ErrInvalidPlanStructure, err)
	}
	if !p.VerifyPlanID() {
		return nil, fmt.Errorf("%w: plan ID mismatch: plan has been altered or is corrupt", ErrInvalidPlanStructure)
	}
	return &p, nil
}

// RevalidationSet contains current system states to verify a plan is still fresh.
// All fields are strictly required by Revalidate.
type RevalidationSet struct {
	CurrentSourceRev       string
	CurrentConfigBytes     []byte
	CurrentRendererVersion string
	CurrentTargetRoot      string
	CurrentTargetHashes    map[string]string
}

// Revalidate checks a plan against the current revalidation set, failing if any state is stale.
// All fields in rev are strictly required; missing inputs fail closed.
func (p *Plan) Revalidate(rev RevalidationSet) error {
	if !p.VerifyPlanID() {
		return fmt.Errorf("%w: plan ID mismatch (corrupted or altered plan)", ErrStalePlan)
	}
	if rev.CurrentSourceRev == "" {
		return fmt.Errorf("%w: source revision is required for revalidation", ErrStalePlan)
	}
	if rev.CurrentSourceRev != p.SourceRev {
		return fmt.Errorf("%w: source revision changed (was %s, now %s)", ErrStalePlan, p.SourceRev, rev.CurrentSourceRev)
	}
	if len(rev.CurrentConfigBytes) == 0 {
		return fmt.Errorf("%w: config bytes are required for revalidation", ErrStalePlan)
	}
	h := sha256.Sum256(rev.CurrentConfigBytes)
	digest := hex.EncodeToString(h[:])
	if digest != p.ConfigDigest {
		return fmt.Errorf("%w: configuration changed (was %s, now %s)", ErrStalePlan, p.ConfigDigest, digest)
	}
	if rev.CurrentRendererVersion == "" {
		return fmt.Errorf("%w: renderer version is required for revalidation", ErrStalePlan)
	}
	if rev.CurrentRendererVersion != p.RendererVersion {
		return fmt.Errorf("%w: renderer version changed (was %s, now %s)", ErrStalePlan, p.RendererVersion, rev.CurrentRendererVersion)
	}
	if rev.CurrentTargetRoot == "" {
		return fmt.Errorf("%w: target root is required for revalidation", ErrStalePlan)
	}
	if filepath.Clean(rev.CurrentTargetRoot) != filepath.Clean(p.TargetRoot) {
		return fmt.Errorf("%w: target root changed (was %s, now %s)", ErrStalePlan, p.TargetRoot, rev.CurrentTargetRoot)
	}
	if rev.CurrentTargetHashes == nil {
		return fmt.Errorf("%w: target hashes are required for revalidation", ErrStalePlan)
	}
	for _, f := range p.FileList {
		expected := p.LastKnownInstalled[f]
		current, ok := rev.CurrentTargetHashes[f]
		if !ok || current != expected {
			return fmt.Errorf("%w: target file %q modified since plan was generated", ErrStalePlan, f)
		}
	}
	return nil
}

// CurrentTargetHashes inspects targetRoot on disk for each file in fileList
// and returns a map of filename to SHA-256 digest (or "" if file does not
// exist). Symlinks and special files are refused, never followed.
func CurrentTargetHashes(targetRoot string, fileList []string) (map[string]string, error) {
	hashes := make(map[string]string, len(fileList))
	for _, f := range fileList {
		path, err := targetPathFor(targetRoot, f)
		if err != nil {
			return nil, err
		}
		digest, exists, err := onDiskDigest(path)
		if err != nil {
			return nil, err
		}
		if !exists {
			digest = ""
		}
		hashes[f] = digest
	}
	return hashes, nil
}

// HasConflicts reports whether the plan contains any operation that must not
// be applied without a human decision.
func (p *Plan) HasConflicts() bool {
	for _, op := range p.ProposedOps {
		if op.Op == "conflict" {
			return true
		}
	}
	return false
}

// RevalidateAgainstDisk checks a plan against files on disk.
func RevalidateAgainstDisk(p *Plan, configBytes []byte, sourceRev, rendererVersion string) error {
	hashes, err := CurrentTargetHashes(p.TargetRoot, p.FileList)
	if err != nil {
		return fmt.Errorf("failed to inspect target files: %w", err)
	}
	return p.Revalidate(RevalidationSet{
		CurrentSourceRev:       sourceRev,
		CurrentConfigBytes:     configBytes,
		CurrentRendererVersion: rendererVersion,
		CurrentTargetRoot:      p.TargetRoot,
		CurrentTargetHashes:    hashes,
	})
}

// ManifestPath returns the single fleet manifest path: <backupRoot>/<stamp>/fleet-manifest.json.
func ManifestPath(backupRoot, stamp string) string {
	return filepath.Join(backupRoot, stamp, ManifestFileName)
}

// LivePointerPath returns the live fleet manifest pointer path: <home>/.agentic-sdd/fleet/current.json.
func LivePointerPath(home string) string {
	return filepath.Join(home, ".agentic-sdd", "fleet", "current.json")
}

// Lock is an exclusive operating-system lock on ${backupRoot}/.fleet.lock.
// The file is a stable inode used for ownership diagnostics only: the kernel
// holds the exclusion, so a dead process can never leave a lock that blocks a
// later apply and Release can never delete another process's lock.
type Lock struct {
	path  string
	file  *os.File
	token string
}

// AcquireLock takes the exclusive, non-blocking fleet lock at
// ${backupRoot}/.fleet.lock. Liveness is authoritative in the operating
// system: a lock left by a dead process is reclaimed as soon as that
// process's descriptor is gone, regardless of the PID or timestamp recorded in
// the file. The recorded PID and timestamp are diagnostics and identify the
// holder in ErrLockHeld. If another live process holds the lock, AcquireLock
// returns ErrLockHeld without waiting.
func AcquireLock(backupRoot string) (*Lock, error) {
	absRoot, err := filepath.Abs(backupRoot)
	if err != nil {
		return nil, err
	}
	if err := checkNoSymlinkDir(absRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absRoot, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup root for lock: %w", err)
	}
	lockPath := filepath.Join(absRoot, LockFileName)

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	ok, err := tryLockFile(f)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to acquire lock file %s: %w", lockPath, err)
	}
	if !ok {
		pid := recordedLockPID(lockPath)
		_ = f.Close()
		return nil, fmt.Errorf("%w: lock held at %s (PID %d)", ErrLockHeld, lockPath, pid)
	}

	token, err := newLockToken()
	if err != nil {
		_ = unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	content := fmt.Sprintf("%d\n%s\n%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano), token)
	if err := f.Truncate(0); err != nil {
		_ = unlockFile(f)
		_ = f.Close()
		return nil, fmt.Errorf("failed to record lock ownership: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		_ = unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = unlockFile(f)
		_ = f.Close()
		return nil, fmt.Errorf("failed to record lock ownership: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = unlockFile(f)
		_ = f.Close()
		return nil, err
	}
	return &Lock{path: lockPath, file: f, token: content}, nil
}

// Release drops this handle's lock. The recorded content is cleared only when
// the file still carries this acquisition's token, so a lock file that another
// process re-created is never modified. The lock file itself is kept: the
// stable inode is what makes the lock atomic.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	var firstErr error
	if data, err := os.ReadFile(l.path); err == nil && string(data) == l.token {
		if err := l.file.Truncate(0); err != nil {
			firstErr = err
		}
	}
	if err := unlockFile(l.file); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := l.file.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	l.file = nil
	return firstErr
}

func newLockToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate lock token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// recordedLockPID reads the diagnostic PID from a lock file. Any unreadable or
// malformed content reports 0; it never decides ownership.
func recordedLockPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	line, _, _ := strings.Cut(string(data), "\n")
	pid, _ := strconv.Atoi(strings.TrimSpace(line))
	return pid
}

// LoadConfigFile reads a fleet config file, computes its SHA-256 digest, and strictly parses it.
// It rejects non-regular files and bounds reads via LimitReader.
func LoadConfigFile(path string) (*Config, []byte, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, nil, "", fmt.Errorf("%w: refusing non-regular file %s", ErrInvalidConfig, filepath.Clean(path))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, MaxConfigBytes+1))
	if err != nil {
		return nil, nil, "", fmt.Errorf("read config %s: %w", filepath.Clean(path), err)
	}
	if len(data) > MaxConfigBytes {
		return nil, nil, "", fmt.Errorf("%w: config exceeds %d bytes", ErrInvalidConfig, MaxConfigBytes)
	}
	cfg, err := parseConfigBytes(data)
	if err != nil {
		return nil, nil, "", err
	}
	h := sha256.Sum256(data)
	return cfg, data, hex.EncodeToString(h[:]), nil
}

// BuildPlan constructs a deterministic Plan from config and targets. It is
// read-only: it resolves and renders in memory, compares the three drift
// states (last applied, current installed, newly desired) and reports
// unmanaged collisions as conflicts unless adopt is explicit.
func BuildPlan(configPath, repo, home string, adopt bool) (*Plan, error) {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(absHome) {
		return nil, fmt.Errorf("home directory must be absolute: %s", filepath.Clean(home))
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}
	absRepo := ""
	if repo != "" {
		absRepo, err = filepath.Abs(repo)
		if err != nil {
			return nil, err
		}
	}

	cfg, _, configDigest, err := LoadConfigFile(absConfig)
	if err != nil {
		return nil, err
	}
	pack, err := skillsync.OpenCodePackFS(absRepo)
	if err != nil {
		return nil, fmt.Errorf("open canonical OpenCode pack: %w", err)
	}
	resolved, err := Resolve(pack, cfg)
	if err != nil {
		return nil, err
	}

	targetRoot := TargetRootForHome(absHome)
	if err := checkTargetParents(targetRoot, absHome); err != nil {
		return nil, err
	}
	live, err := ReadLiveManifest(absHome)
	if err != nil {
		return nil, err
	}
	if live == nil {
		// A lost live pointer must not turn every managed file into an
		// unmanaged collision: replay the recorded fleet transactions.
		live, err = RebuildLiveManifest(absHome)
		if err != nil {
			return nil, err
		}
	}
	if live != nil {
		if err := validateLiveManifest(live, absHome, targetRoot); err != nil {
			return nil, err
		}
		if err := live.Validate(live.ID, absHome, live.BackupRoot, targetRoot); err != nil {
			return nil, fmt.Errorf("live fleet manifest: %w", err)
		}
	}

	rendered := make(map[string]Rendered, len(resolved))
	for _, r := range resolved {
		if !r.Enabled {
			continue
		}
		out, err := RenderResolved(r)
		if err != nil {
			return nil, err
		}
		rendered[r.InstanceID] = out
	}

	var fileList []string
	ops := make([]ProposedOp, 0, len(resolved))
	lastKnown := make(map[string]string, len(resolved))

	for _, r := range resolved {
		fileName := r.InstanceID + ".md"
		target, err := targetPathFor(targetRoot, fileName)
		if err != nil {
			return nil, err
		}
		fileList = append(fileList, fileName)

		current, exists, err := onDiskDigest(target)
		if err != nil {
			return nil, err
		}
		lastKnown[fileName] = current

		sourceRef := SourceRef{Kind: r.SourceKind, Name: r.SourceName}.String()
		priorModel := ""
		rec, hasRec := live.RecordFor(target)
		if hasRec {
			if rec.Model != "" {
				priorModel = rec.Model
			}
		}
		if priorModel == "" && exists {
			if content, readErr := os.ReadFile(target); readErr == nil {
				priorModel = ExtractModelFromContent(content)
			}
		}

		op := ProposedOp{
			InstanceID:    r.InstanceID,
			File:          fileName,
			SourceRef:     sourceRef,
			SourceDigest:  r.SourceDigest,
			Model:         r.Model,
			PriorModel:    priorModel,
			CurrentDigest: current,
		}
		if hasRec {
			op.Provenance = rec.Provenance
			if rec.IsManaged() {
				op.LastAppliedDigest = rec.DesiredDigest
			}
		} else {
			op.Provenance = ProvenanceUnknown
		}

		if !r.Enabled {
			classifyDisabled(&op, hasRec, rec, exists, current, adopt)
			ops = append(ops, op)
			continue
		}

		desired := rendered[r.InstanceID]
		op.DesiredDigest = desired.DesiredDigest
		classifyEnabled(&op, hasRec, rec, exists, current, adopt)
		ops = append(ops, op)
	}

	var permNotes []string
	for _, r := range resolved {
		if IsReservedName(r.InstanceID) {
			permNotes = append(permNotes, fmt.Sprintf("warning: instance %q uses reserved SDD name", r.InstanceID))
		}
		if r.Enabled {
			if d, ok := rendered[r.InstanceID]; ok {
				pDigest, pErr := PermissionDigest(d.Bytes)
				if pErr == nil && pDigest != "" {
					permNotes = append(permNotes, fmt.Sprintf("instance %s: permissions %s", r.InstanceID, shortDigest(pDigest)))
				}
			}
		}
	}
	for _, op := range ops {
		if op.PriorModel != "" && op.Model != "" && op.PriorModel != op.Model {
			permNotes = append(permNotes, fmt.Sprintf("warning: instance %s model route changed from %q to %q", op.InstanceID, op.PriorModel, op.Model))
		}
	}
	sort.Strings(permNotes)

	plan := &Plan{
		SourceRev:          sourceRevision(resolved),
		ConfigDigest:       configDigest,
		ConfigPath:         absConfig,
		RepoPath:           absRepo,
		Scope:              ScopeGlobal,
		RendererVersion:    RendererVersion,
		TargetRoot:         targetRoot,
		FileList:           fileList,
		LastKnownInstalled: lastKnown,
		ProposedOps:        ops,
		PermNotes:          permNotes,
	}

	if err := plan.Seal(); err != nil {
		return nil, err
	}
	return plan, nil
}

// classifyDisabled implements the explicit-disable row: a clean managed
// profile is removed with a recoverable backup, a drifted managed profile
// requires the same explicit --adopt decision as an overwrite, and an
// unmanaged same-name file is preserved even when the instance is disabled.
// Absent keys never reach here, so absence alone can never delete anything.
func classifyDisabled(op *ProposedOp, hasRec bool, rec FileRecord, exists bool, current string, adopt bool) {
	switch {
	case hasRec && rec.IsManaged() && exists:
		if current == rec.DesiredDigest {
			op.Op = "remove"
			return
		}
		if adopt {
			op.Op = "remove"
			op.Adopt = true
			op.Reason = "explicit adoption of manually drifted content before managed removal"
			return
		}
		op.Op = "conflict"
		op.Reason = "managed file drifted from the last applied digest; use --adopt to remove it with a backup"
	case hasRec && rec.IsManaged():
		op.Op = "noop"
		op.Reason = "managed profile is already absent"
	case exists:
		op.Op = "noop"
		op.Reason = "unmanaged file preserved (not removed)"
	default:
		op.Op = "noop"
		op.Reason = "not installed"
	}
}

// classifyEnabled implements the three-state drift table. current is the
// digest observed on disk ("" when absent); lastApplied comes from the live
// sidecar record. Any manual drift is a conflict unless --adopt explicitly
// adopts content that already equals the desired bytes.
func classifyEnabled(op *ProposedOp, hasRec bool, rec FileRecord, exists bool, current string, adopt bool) {
	if hasRec && rec.IsManaged() {
		lastApplied := rec.DesiredDigest
		switch {
		case current == lastApplied && op.DesiredDigest == current:
			op.Op = "noop"
		case current == lastApplied:
			op.Op = "update"
		case exists && op.DesiredDigest == current && adopt:
			op.Op = "update"
			op.Adopt = true
			op.Reason = "explicit adoption of manually drifted content that already matches the desired bytes"
		case !exists:
			op.Op = "conflict"
			op.Reason = "managed file is missing; restore it or remove its recorded ownership before applying"
		case op.DesiredDigest == lastApplied:
			op.Op = "conflict"
			op.Reason = "manual drift: installed file differs from the last applied digest"
		default:
			op.Op = "conflict"
			op.Reason = "manual drift conflicts with the desired change; no speculative overwrite"
		}
		return
	}
	if hasRec {
		// A record with legacy/unknown provenance must never be treated as a
		// clean managed file: require explicit adoption.
		if adopt {
			op.Op = "update"
			op.Adopt = true
			op.Reason = "explicit adoption of legacy/unknown provenance"
			return
		}
		op.Op = "conflict"
		op.Reason = "legacy/unknown provenance; use --adopt after review"
		return
	}
	switch {
	case !exists:
		op.Op = "create"
	case adopt:
		op.Op = "update"
		op.Adopt = true
		op.Reason = "explicit adoption of an unmanaged file"
	default:
		op.Op = "conflict"
		op.Reason = "unmanaged file already exists; use --adopt after review"
	}
}

func shortDigest(digest string) string {
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

// sourceRevision binds the exact canonical source bytes the resolver read for
// the whole config, including disabled instances.
func sourceRevision(resolved []Resolved) string {
	var b strings.Builder
	for _, r := range resolved {
		ref := SourceRef{Kind: r.SourceKind, Name: r.SourceName}.String()
		fmt.Fprintf(&b, "%s\x00%s\x00%s\n", r.InstanceID, ref, r.SourceDigest)
	}
	return digestBytes([]byte(b.String()))
}

// checkTargetParents refuses a target root outside home and any existing
// non-directory (including symlink) component between it and home. Unlike a
// bare loop up to home, it terminates safely when home does not exist yet,
// which plan/status allow.
func checkTargetParents(root, home string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absHome, absRoot)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing target root %s outside home %s", absRoot, absHome)
	}
	for p := absRoot; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing symlink path component %s", filepath.Clean(p))
			}
			if !info.IsDir() {
				return fmt.Errorf("refusing non-directory path component %s", filepath.Clean(p))
			}
		}
		if p == absHome {
			return nil
		}
		parent := filepath.Dir(p)
		if parent == p {
			return fmt.Errorf("target root %s is not under home %s", absRoot, absHome)
		}
	}
}
