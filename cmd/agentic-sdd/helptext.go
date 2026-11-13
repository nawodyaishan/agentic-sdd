package main

import (
	"fmt"
	"io"
)

const (
	canonicalCLIName = "asdx-core"
	canonicalTUIName = "asdx"
	legacyCLIName    = "agentic-sdd"
	legacyTUIName    = "agentic-sdd-tui"
)

// topicHelp holds the per-topic help blocks printed by "asdx-core help
// <topic>" before the unchanged global usage block. Each block states what
// the command does, whether it writes files, one example and one related
// command. Help is never styled with ANSI color.
var topicHelp = map[string]string{
	"preview": `preview — show what apply would change (read-only)

Plans every skill for the agent client directories and prints one line per
item: install, replace, or up to date. Nothing is written and no backup is
created.
Migration: legacy "agentic-sdd" wraps "asdx-core" (CLI) and "agentic-sdd-tui"
wraps "asdx" (TUI); ~/.agentic-sdd/ and AGENTIC_SDD_* stay unchanged.
Writes files: no (read-only)
Example: asdx-core preview --repo ./my-skills --home ~
Related: asdx-core help apply

`,
	"apply": `apply — back up changed skills, then install

Installs skills into the agent client directories. Existing skills are
replaced only after they are copied into a backup, and every run writes a log
under ~/.agentic-sdd/logs.
Writes files: yes (backs up replaced items first)
Example: asdx-core apply --repo ./my-skills --home ~
Related: asdx-core restore

`,
	"backups": `backups — list available backups, newest first

Lists each backup with its id, creation time, operation, tool version and
summary line. This is the same listing as "asdx-core restore list".
Writes files: no (read-only)
Example: asdx-core backups --home ~
Related: asdx-core restore

`,
	"logs": `logs — show the latest apply or restore log

Prints the most recent log file from ~/.agentic-sdd/logs. Logs are written by
apply and restore only, so run one of those first.
Writes files: no (read-only)
Example: asdx-core logs --home ~
Related: asdx-core apply

`,
	"restore": `restore — preview or apply a backup by id or number

With an id it previews that restore; add --apply (or use "restore apply ID")
to write it. Replaced skills are backed up before they are overwritten.
Writes files: only with --apply (backs up replaced items first)
Example: asdx-core restore 20260101T000000.000000000Z --apply
Related: asdx-core backups

`,
	"guide": `guide — interactive step-by-step wizard

Leads through installing, restoring, or reading the last log with numbered
prompts. A preview is always shown before anything is written. For the full
terminal UI with Guide, run "asdx" (legacy "agentic-sdd-tui").
Writes files: only if you confirm (backs up replaced items first)
Example: asdx-core guide --home ~
Related: asdx-core help preview

`,
	"version": `version — print version, commit, and build date

Shows the asdx-core version, the Git commit and build date it was built
from, and the Go version used to compile it. Legacy "agentic-sdd" and
"agentic-sdd-tui" identify their compatibility wrapper names on --version.
Writes files: no (read-only)
Example: asdx-core version
Related: asdx-core help

`,
	"tui": `tui — open the terminal UI in a real terminal (compatibility launcher)

Runs the packaged terminal UI (asdx) from the same installation. Prefer
running "asdx" directly in an interactive terminal; legacy "agentic-sdd-tui"
wraps "asdx", and legacy "agentic-sdd" wraps "asdx-core". It needs a real
input and output terminal; a pipe, a script, or
AGENTIC_SDD_FORCE_INTERACTIVE=1 keeps the standard CLI and never opens the
alternate screen. A missing or mismatched companion is an error, never a
silent fallback.
Writes files: only after an explicit confirmation inside the UI
Example: asdx-core tui --home ~
Related: asdx-core help preview

`,
	"run": `run — private execution workspace and check evidence

Creates and inspects one private run under .agentic-sdd/runs/<run-id>/:
init, status, check, record and clean. It never approves work, calls a model
or acts as a sandbox; a passing check or record is an observation, never
acceptance or human approval.
Writes files: yes (its own run scratch only; clean without --yes is a preview)
Example: asdx-core run init --feature specs/016-agent-execution-context-evidence-recovery --batch B3
Related: asdx-core help task

`,
	"task": `task — extract one authorized task into a run

Runs "asdx-core task brief --run ID --task T01 [--force]" to copy one task
section, its acceptance references and source digests into briefs/<task>.md.
Requires a valid owning run under the per-run writer lock (foreign
checkpoints exit 2) and refuses direct-scope runs created with --direct
(exit 2). Creation
registers the brief path and digest in checkpoint.json while preserving
existing history; re-issuing registers a byte-identical unregistered brief
after source validation, while differing bytes require --force (--force never
bypasses a changed source scope). Fails closed on missing, duplicate or
ambiguous task IDs; a brief, check or record is an observation, never approval.
Writes files: yes (briefs/<task>.md and checkpoint registration inside the run)
Example: asdx-core task brief --run run-20261113T114500Z-a1b2c3 --task T14
Related: asdx-core help run

`,
	"review": `review — package actual Git changes for a bounded review

Writes a manifest and patch covering committed, staged, unstaged and untracked
changes, with binary, sensitive and oversized content handled explicitly. An
incomplete package exits 3 and names every omission; it never claims a review
or approval.
Writes files: yes (review/<package-id>/ inside the run)
Example: asdx-core review package --run run-20261113T114500Z-a1b2c3 --base a8f1c1c
Related: asdx-core help run

`,
}

// printTopicHelp writes the topic-specific block for topic and reports
// whether printTopicHelp owns that topic. An unknown topic writes nothing.
func printTopicHelp(out io.Writer, topic string) bool {
	text, ok := topicHelp[topic]
	if !ok {
		return false
	}
	fmt.Fprint(out, text)
	return true
}
