package main

import (
	"bytes"
	"strings"
	"testing"
)

// Pinned from the pre-011 binary with real terminal stdin and piped stdout.
// New terminal branding must not change this legacy, script-readable branch.
func TestGuidePlainBrandingCompatibility(t *testing.T) {
	repo, home := guideFixture(t)
	var stdout, stderr bytes.Buffer
	code := run([]string{"guide", "--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader("4\nq\n"), true)
	const menu = `Agentic SDD — engineering skills for your coding agents
Nothing is written until you confirm.

What would you like to do?
  1) Install or update skills   (default)
  2) Restore an earlier backup
  3) Show the last install log
  4) What is this?
  q) Quit
` + "Choose [1]: "
	const about = `
Agentic SDD installs reusable engineering skills for coding agents.
It writes into four client directories, backs up anything it replaces,
and can undo an install with "asdx-core restore".
Every action is also a plain command: preview, apply, backups,
restore, logs. Nothing is written until you confirm.

`
	want := menu + about + menu + "Cancelled.\n"
	if code != 0 || stderr.Len() != 0 || stdout.String() != want {
		t.Fatalf("legacy guide changed: exit=%d stderr=%q\ngot %q\nwant %q", code, stderr.String(), stdout.String(), want)
	}
	assertNoWrites(t, home)
}
