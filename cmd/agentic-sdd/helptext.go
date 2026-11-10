package main

import (
	"fmt"
	"io"
)

// topicHelp holds the per-topic help blocks printed by "agentic-sdd help
// <topic>" before the unchanged global usage block. Each block states what
// the command does, whether it writes files, one example and one related
// command. Help is never styled with ANSI color.
var topicHelp = map[string]string{
	"preview": `preview — show what apply would change (read-only)

Plans every skill for the agent client directories and prints one line per
item: install, replace, or up to date. Nothing is written and no backup is
created.
Writes files: no (read-only)
Example: agentic-sdd preview --repo ./my-skills --home ~
Related: agentic-sdd help apply

`,
	"apply": `apply — back up changed skills, then install

Installs skills into the agent client directories. Existing skills are
replaced only after they are copied into a backup, and every run writes a log
under ~/.agentic-sdd/logs.
Writes files: yes (backs up replaced items first)
Example: agentic-sdd apply --repo ./my-skills --home ~
Related: agentic-sdd restore

`,
	"backups": `backups — list available backups, newest first

Lists each backup with its id, creation time, operation, tool version and
summary line. This is the same listing as "agentic-sdd restore list".
Writes files: no (read-only)
Example: agentic-sdd backups --home ~
Related: agentic-sdd restore

`,
	"logs": `logs — show the latest apply or restore log

Prints the most recent log file from ~/.agentic-sdd/logs. Logs are written by
apply and restore only, so run one of those first.
Writes files: no (read-only)
Example: agentic-sdd logs --home ~
Related: agentic-sdd apply

`,
	"restore": `restore — preview or apply a backup by id or number

With an id it previews that restore; add --apply (or use "restore apply ID")
to write it. Replaced skills are backed up before they are overwritten.
Writes files: only with --apply (backs up replaced items first)
Example: agentic-sdd restore 20260101T000000.000000000Z --apply
Related: agentic-sdd backups

`,
	"guide": `guide — interactive step-by-step wizard

Leads through installing, restoring, or reading the last log with numbered
prompts. A preview is always shown before anything is written.
Writes files: only if you confirm (backs up replaced items first)
Example: agentic-sdd guide --home ~
Related: agentic-sdd help preview

`,
	"version": `version — print version, commit, and build date

Shows the agentic-sdd version, the Git commit and build date it was built
from, and the Go version used to compile it.
Writes files: no (read-only)
Example: agentic-sdd version
Related: agentic-sdd help

`,
	"tui": `tui — open the terminal UI in a real terminal

Runs the packaged terminal UI (agentic-sdd-tui) from the same installation.
It needs a real input and output terminal; a pipe, a script, or
AGENTIC_SDD_FORCE_INTERACTIVE=1 keeps the standard CLI and never opens the
alternate screen. A missing or mismatched companion is an error, never a
silent fallback.
Writes files: only after an explicit confirmation inside the UI
Example: agentic-sdd tui --home ~
Related: agentic-sdd help preview

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
