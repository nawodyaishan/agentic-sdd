package appops

import (
	"errors"
	"path/filepath"

	"agentic-sdd/internal/skillsync"
)

// InstallRequest is an immutable, validated request for one install preview
// or apply. Its fields are unexported so home, repo, options and the client
// selection cannot change after validation; construct it with
// NewInstallRequest and read it through the accessors.
type InstallRequest struct {
	repo  string
	home  string
	apply bool
	opts  skillsync.Options
}

// NewInstallRequest resolves and validates one install request before any
// operation touches the filesystem. home is required and both paths are made
// absolute. The client selection is copied and normalized: nil keeps the
// historical all-clients behavior, while an explicit empty or unknown
// selection fails here, before any preview or write. A Free-first preset
// without the OpenCode pack is rejected for the same reason.
func NewInstallRequest(repo, home string, apply bool, opts skillsync.Options) (InstallRequest, error) {
	if home == "" {
		return InstallRequest{}, errors.New("install request home is required")
	}
	if opts.OpenCodeFreeFirst && !opts.WithOpenCode {
		return InstallRequest{}, errors.New("--opencode-free-first requires --with-opencode")
	}
	ids, err := skillsync.NormalizeClientSelection(opts.ClientIDs)
	if err != nil {
		return InstallRequest{}, err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return InstallRequest{}, err
	}
	absRepo := ""
	if repo != "" {
		if absRepo, err = filepath.Abs(repo); err != nil {
			return InstallRequest{}, err
		}
	}
	opts.ClientIDs = ids
	return InstallRequest{repo: absRepo, home: absHome, apply: apply, opts: opts}, nil
}

// Repo returns the absolute repository path, or "" for the embedded source.
func (r InstallRequest) Repo() string { return r.repo }

// Home returns the absolute home directory the request plans against.
func (r InstallRequest) Home() string { return r.home }

// Apply reports whether the request asked for the write variant.
func (r InstallRequest) Apply() bool { return r.apply }

// Options returns a copy of the request's engine options. Changing the
// returned slice cannot change the request.
func (r InstallRequest) Options() skillsync.Options {
	opts := r.opts
	opts.ClientIDs = append([]string(nil), r.opts.ClientIDs...)
	return opts
}

// RestoreRequest is an immutable, validated request for one restore preview
// or apply. Construct it with NewRestoreRequest.
type RestoreRequest struct {
	repo  string
	home  string
	id    string
	apply bool
}

// NewRestoreRequest resolves and validates one restore request before any
// operation touches the filesystem. home is required and both paths are made
// absolute; id must have the exact backup-directory syntax.
func NewRestoreRequest(repo, home, id string, apply bool) (RestoreRequest, error) {
	if home == "" {
		return RestoreRequest{}, errors.New("restore request home is required")
	}
	if err := skillsync.ValidateBackupID(id); err != nil {
		return RestoreRequest{}, err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return RestoreRequest{}, err
	}
	absRepo := ""
	if repo != "" {
		if absRepo, err = filepath.Abs(repo); err != nil {
			return RestoreRequest{}, err
		}
	}
	return RestoreRequest{repo: absRepo, home: absHome, id: id, apply: apply}, nil
}

// Repo returns the absolute repository path, or "" for the embedded source.
func (r RestoreRequest) Repo() string { return r.repo }

// Home returns the absolute home directory the request plans against.
func (r RestoreRequest) Home() string { return r.home }

// BackupID returns the validated backup directory name to restore.
func (r RestoreRequest) BackupID() string { return r.id }

// Apply reports whether the request asked for the write variant.
func (r RestoreRequest) Apply() bool { return r.apply }

// OperationID identifies one requested operation and the generation of a
// result produced for it. Request separates independent operations;
// Generation increases each time the same request starts new work. A result
// may update screen state only while it is not stale for the current identity,
// so a slow earlier result cannot overwrite a newer one.
type OperationID struct {
	Request    uint64
	Generation uint64
}

// StaleFor reports whether id can no longer update state that current
// describes. A result from a different request, or an older generation of the
// same request, is stale; an equal identity is current.
func (id OperationID) StaleFor(current OperationID) bool {
	return id.Request != current.Request || id.Generation < current.Generation
}
