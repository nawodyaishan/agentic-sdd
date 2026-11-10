package tui_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"agentic-sdd/experiments/tui"
	"github.com/charmbracelet/x/ansi"
)

// These tests cover the trusted browser opener and terminal-hyperlink helper
// only. Every launch and every path resolution is injected, so the suite never
// starts a browser, reads the real environment or touches a real home.

// openerEnv builds an injected environment lookup. Missing keys are empty, so a
// desktop test only names the variables it needs.
func openerEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// recordingLaunch records one argument vector and reports the configured error.
type recordingLaunch struct {
	calls   int
	program string
	args    []string
	err     error
}

func (r *recordingLaunch) launch(program string, args []string) error {
	r.calls++
	r.program = program
	r.args = append([]string(nil), args...)
	return r.err
}

// TestOpenerFixedHTTPSLinks pins the only destinations the TUI may launch. The
// values are Feature 013 plan §3's trusted presentation constants; the author
// spelling is the GitHub handle.
func TestOpenerFixedHTTPSLinks(t *testing.T) {
	if got, want := tui.ReadmeURL, "https://github.com/nawodyaishan/agentic-sdd#readme"; got != want {
		t.Errorf("ReadmeURL = %q, want %q", got, want)
	}
	if got, want := tui.AuthorURL, "https://github.com/nawodyaishan"; got != want {
		t.Errorf("AuthorURL = %q, want %q", got, want)
	}
	if got, want := tui.LinkReadme.URL(), tui.ReadmeURL; got != want {
		t.Errorf("LinkReadme.URL() = %q, want %q", got, want)
	}
	if got, want := tui.LinkAuthor.URL(), tui.AuthorURL; got != want {
		t.Errorf("LinkAuthor.URL() = %q, want %q", got, want)
	}
	if !tui.LinkReadme.Trusted() || !tui.LinkAuthor.Trusted() {
		t.Fatal("the fixed README and author links must be trusted")
	}
}

// TestOpenerRejectsArbitraryDestinations proves a source-provided or mistyped
// URL can never reach the opener, even when the caller asks for it directly.
func TestOpenerRejectsArbitraryDestinations(t *testing.T) {
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:     "linux",
		Getenv:   openerEnv(map[string]string{"DISPLAY": ":0"}),
		LookPath: func(string) (string, error) { return "/usr/bin/xdg-open", nil },
		Launch:   launch.launch,
	}

	untrusted := tui.Link("https://example.invalid/not-fixed")
	if untrusted.Trusted() {
		t.Fatal("an arbitrary destination must not be trusted")
	}
	if got := untrusted.URL(); got != "" {
		t.Fatalf("untrusted URL() = %q, want empty", got)
	}

	result := opener.Open(untrusted)
	if result.Opened {
		t.Fatal("Open() opened an untrusted destination")
	}
	if launch.calls != 0 {
		t.Fatalf("launcher calls = %d, want 0 for an untrusted destination", launch.calls)
	}
	if !strings.Contains(result.Explanation(), "untrusted") {
		t.Errorf("Explanation() = %q, want it to explain the untrusted destination", result.Explanation())
	}
}

// TestOpenerLinuxResolvesOnceAndLaunchesArgumentVector proves Linux uses one
// LookPath resolution to an absolute path and then launches that exact program
// with the URL as a single argument vector element. No shell is involved.
func TestOpenerLinuxResolvesOnceAndLaunchesArgumentVector(t *testing.T) {
	lookCalls := 0
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:   "linux",
		Getenv: openerEnv(map[string]string{"DISPLAY": ":0"}),
		LookPath: func(name string) (string, error) {
			lookCalls++
			if name != "xdg-open" {
				t.Fatalf("LookPath(%q), want xdg-open", name)
			}
			return "/usr/bin/xdg-open", nil
		},
		Launch: launch.launch,
	}

	first := opener.Resolve(tui.LinkReadme)
	second := opener.Resolve(tui.LinkReadme)
	if lookCalls != 1 {
		t.Fatalf("LookPath calls = %d, want exactly 1 cached resolution", lookCalls)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Resolve() changed across calls: first %#v second %#v", first, second)
	}
	if !first.Resolved {
		t.Fatalf("Resolve() = %#v, want a resolved plan", first)
	}
	if first.Program != "/usr/bin/xdg-open" {
		t.Errorf("Program = %q, want the absolute xdg-open path", first.Program)
	}
	if len(first.Args) != 1 || first.Args[0] != tui.ReadmeURL {
		t.Errorf("Args = %#v, want exactly the fixed URL", first.Args)
	}
	if launch.calls != 0 {
		t.Fatal("Resolve() launched a browser; resolution must have no side effect")
	}

	result := opener.Open(tui.LinkReadme)
	if !result.Opened {
		t.Fatalf("Open() did not open: %s", result.Explanation())
	}
	if launch.calls != 1 {
		t.Fatalf("launcher calls = %d, want 1", launch.calls)
	}
	if launch.program != "/usr/bin/xdg-open" {
		t.Errorf("launched program = %q, want /usr/bin/xdg-open", launch.program)
	}
	if len(launch.args) != 1 || launch.args[0] != tui.ReadmeURL {
		t.Errorf("launched args = %#v, want exactly [%q]", launch.args, tui.ReadmeURL)
	}
	for _, forbidden := range []string{"/bin/sh", "/bin/bash", "sh", "bash", "cmd"} {
		if launch.program == forbidden {
			t.Errorf("launched program = %q, want no shell", launch.program)
		}
	}
	if lookCalls != 1 {
		t.Fatalf("Open() re-resolved the opener; LookPath calls = %d, want 1", lookCalls)
	}
}

// TestOpenerDarwinUsesFixedOpen proves macOS never searches PATH: it uses the
// fixed /usr/bin/open program and the same argument-vector shape.
func TestOpenerDarwinUsesFixedOpen(t *testing.T) {
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:   "darwin",
		Getenv: openerEnv(nil),
		LookPath: func(string) (string, error) {
			t.Fatal("macOS must not resolve xdg-open or search PATH")
			return "", nil
		},
		Launch: launch.launch,
	}

	plan := opener.Resolve(tui.LinkAuthor)
	if !plan.Resolved {
		t.Fatalf("Resolve() = %#v, want a resolved plan", plan)
	}
	if plan.Program != "/usr/bin/open" {
		t.Errorf("Program = %q, want /usr/bin/open", plan.Program)
	}
	if len(plan.Args) != 1 || plan.Args[0] != tui.AuthorURL {
		t.Errorf("Args = %#v, want exactly the author URL", plan.Args)
	}
	if result := opener.Open(tui.LinkAuthor); !result.Opened {
		t.Fatalf("Open() did not open: %s", result.Explanation())
	}
	if launch.program != "/usr/bin/open" || len(launch.args) != 1 || launch.args[0] != tui.AuthorURL {
		t.Errorf("launched %q %#v, want /usr/bin/open [%q]", launch.program, launch.args, tui.AuthorURL)
	}
}

// TestOpenerMissingProgramRetainsURL proves an absent xdg-open is a non-fatal
// fallback that keeps the fixed URL visible and explains manual opening.
func TestOpenerMissingProgramRetainsURL(t *testing.T) {
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:   "linux",
		Getenv: openerEnv(map[string]string{"DISPLAY": ":0"}),
		LookPath: func(string) (string, error) {
			return "", errors.New("executable file not found in $PATH")
		},
		Launch: launch.launch,
	}

	result := opener.Open(tui.LinkReadme)
	if result.Opened {
		t.Fatal("Open() claimed success without an opener")
	}
	if launch.calls != 0 {
		t.Fatalf("launcher calls = %d, want 0", launch.calls)
	}
	explanation := result.Explanation()
	if !strings.Contains(explanation, tui.ReadmeURL) {
		t.Errorf("Explanation() = %q, want the visible URL", explanation)
	}
	if !strings.Contains(strings.ToLower(explanation), "manually") {
		t.Errorf("Explanation() = %q, want local opening guidance", explanation)
	}
}

// TestOpenerRelativePathIsRefused proves a PATH entry resolving to a relative
// path is rejected rather than becoming a working-directory-relative fallback.
func TestOpenerRelativePathIsRefused(t *testing.T) {
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:   "linux",
		Getenv: openerEnv(map[string]string{"DISPLAY": ":0"}),
		LookPath: func(string) (string, error) {
			return "bin/xdg-open", nil
		},
		Launch: launch.launch,
	}

	result := opener.Open(tui.LinkReadme)
	if result.Opened {
		t.Fatal("Open() accepted a relative opener path")
	}
	if launch.calls != 0 {
		t.Fatalf("launcher calls = %d, want 0", launch.calls)
	}
	if !strings.Contains(strings.ToLower(result.Explanation()), "relative") {
		t.Errorf("Explanation() = %q, want the relative-path refusal", result.Explanation())
	}
}

// TestOpenerSSHSessionDoesNotLaunch proves a remote session cannot claim to
// open the user's laptop browser; the URL and a local explanation remain.
func TestOpenerSSHSessionDoesNotLaunch(t *testing.T) {
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:   "linux",
		Getenv: openerEnv(map[string]string{"SSH_CONNECTION": "10.0.0.1 1234 10.0.0.2 22", "DISPLAY": ":0"}),
		LookPath: func(string) (string, error) {
			t.Fatal("an SSH session must not resolve or launch an opener")
			return "", nil
		},
		Launch: launch.launch,
	}

	result := opener.Open(tui.LinkReadme)
	if result.Opened {
		t.Fatal("Open() launched during an SSH session")
	}
	if launch.calls != 0 {
		t.Fatalf("launcher calls = %d, want 0", launch.calls)
	}
	explanation := strings.ToLower(result.Explanation())
	if !strings.Contains(explanation, "ssh") {
		t.Errorf("Explanation() = %q, want the SSH limitation", result.Explanation())
	}
	if !strings.Contains(result.Explanation(), tui.ReadmeURL) {
		t.Errorf("Explanation() = %q, want the visible URL", result.Explanation())
	}
}

// TestOpenerHeadlessLinuxDoesNotLaunch proves a missing desktop session keeps
// the URL visible instead of attempting an opener that cannot work.
func TestOpenerHeadlessLinuxDoesNotLaunch(t *testing.T) {
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:   "linux",
		Getenv: openerEnv(nil),
		LookPath: func(string) (string, error) {
			t.Fatal("headless Linux must not resolve or launch an opener")
			return "", nil
		},
		Launch: launch.launch,
	}

	result := opener.Open(tui.LinkAuthor)
	if result.Opened {
		t.Fatal("Open() launched without a desktop session")
	}
	if launch.calls != 0 {
		t.Fatalf("launcher calls = %d, want 0", launch.calls)
	}
	explanation := result.Explanation()
	if !strings.Contains(explanation, tui.AuthorURL) {
		t.Errorf("Explanation() = %q, want the visible URL", explanation)
	}
	if !strings.Contains(strings.ToLower(explanation), "desktop") {
		t.Errorf("Explanation() = %q, want the headless explanation", explanation)
	}
}

// TestOpenerLaunchFailureRetainsURL proves an opener that exists but fails is
// non-fatal and still tells the user how to open the URL locally.
func TestOpenerLaunchFailureRetainsURL(t *testing.T) {
	launch := &recordingLaunch{err: errors.New("no browser available")}
	opener := &tui.Opener{
		GOOS:     "linux",
		Getenv:   openerEnv(map[string]string{"DISPLAY": ":0"}),
		LookPath: func(string) (string, error) { return "/usr/bin/xdg-open", nil },
		Launch:   launch.launch,
	}

	result := opener.Open(tui.LinkReadme)
	if result.Opened {
		t.Fatal("Open() claimed success after a launcher error")
	}
	if result.Err == nil {
		t.Fatal("OpenResult.Err = nil, want the launcher error")
	}
	explanation := result.Explanation()
	if !strings.Contains(explanation, tui.ReadmeURL) {
		t.Errorf("Explanation() = %q, want the visible URL", explanation)
	}
	if !strings.Contains(strings.ToLower(explanation), "manually") {
		t.Errorf("Explanation() = %q, want local opening guidance", explanation)
	}
}

// TestOpenerUnsupportedPlatformRetainsURL covers a platform without a
// documented opener; the URL stays available for manual use.
func TestOpenerUnsupportedPlatformRetainsURL(t *testing.T) {
	launch := &recordingLaunch{}
	opener := &tui.Opener{
		GOOS:   "plan9",
		Getenv: openerEnv(nil),
		LookPath: func(string) (string, error) {
			t.Fatal("an unsupported platform must not resolve an opener")
			return "", nil
		},
		Launch: launch.launch,
	}

	result := opener.Open(tui.LinkReadme)
	if result.Opened || launch.calls != 0 {
		t.Fatal("Open() attempted a launch on an unsupported platform")
	}
	if !strings.Contains(result.Explanation(), tui.ReadmeURL) {
		t.Errorf("Explanation() = %q, want the visible URL", result.Explanation())
	}
}

// TestHyperlinkPlainAndOSC8 pins the terminal-hyperlink contract: disabled or
// untrusted input stays plain text, enabled trusted input carries OSC 8, and
// the visible text (the copyable URL) is always present.
func TestHyperlinkPlainAndOSC8(t *testing.T) {
	if got := tui.Hyperlink("guide", tui.LinkReadme, false); got != "guide" {
		t.Errorf("disabled Hyperlink() = %q, want plain text", got)
	}
	if got := tui.Hyperlink("guide", tui.Link("https://example.invalid"), true); got != "guide" {
		t.Errorf("untrusted Hyperlink() = %q, want plain text", got)
	}

	got := tui.Hyperlink(tui.ReadmeURL, tui.LinkReadme, true)
	if !strings.HasPrefix(got, "\x1b]8;;"+tui.ReadmeURL) {
		t.Errorf("Hyperlink() = %q, want an OSC 8 prefix with the fixed URL", got)
	}
	if !strings.HasSuffix(got, "\x1b]8;;\x1b\\") {
		t.Errorf("Hyperlink() = %q, want an OSC 8 terminator", got)
	}
	if !strings.Contains(got, tui.ReadmeURL) {
		t.Errorf("Hyperlink() = %q, want the visible URL", got)
	}
	if width := ansi.StringWidth(got); width != ansi.StringWidth(tui.ReadmeURL) {
		t.Errorf("Hyperlink() width = %d, want the visible width %d (control sequences are zero-width)", width, ansi.StringWidth(tui.ReadmeURL))
	}
}
