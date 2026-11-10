package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Trusted presentation destinations. These are the only URLs the TUI may
// hyperlink or hand to a platform opener; they are constants rather than data
// derived from catalog content or a source selection. The author value matches
// BrandAuthor's exact GitHub handle.
const (
	// ReadmeURL is the online guide / README.
	ReadmeURL = "https://github.com/nawodyaishan/agentic-sdd#readme"
	// AuthorURL is the personal attribution profile.
	AuthorURL = "https://github.com/nawodyaishan"
)

// Link is a trusted, fixed HTTPS destination. Its zero value and any string
// that is not one of the constants above are untrusted: URL returns empty and
// the opener refuses them, so a caller cannot pass an arbitrary source-provided
// URL through the adapter.
type Link string

const (
	// LinkReadme is the online guide / README destination.
	LinkReadme Link = ReadmeURL
	// LinkAuthor is the nawodyaishan profile destination.
	LinkAuthor Link = AuthorURL
)

// Trusted reports whether the link is one of the fixed destinations.
func (l Link) Trusted() bool {
	switch l {
	case LinkReadme, LinkAuthor:
		return true
	default:
		return false
	}
}

// URL returns the fixed HTTPS destination, or an empty string for an untrusted
// link. Never use the raw string conversion instead of this method.
func (l Link) URL() string {
	if !l.Trusted() {
		return ""
	}
	return string(l)
}

// Label is the human-readable name of a trusted link, or empty otherwise.
func (l Link) Label() string {
	switch l {
	case LinkReadme:
		return "Online guide / README"
	case LinkAuthor:
		return "Author profile"
	default:
		return ""
	}
}

// Hyperlink wraps text in an OSC 8 terminal hyperlink when enabled and the
// destination is trusted. It is best-effort: the caller must always show the
// visible/copyable URL next to the linked text, and plain/ASCII sessions pass
// enabled=false so no hyperlink control sequence is emitted. The text is not
// escaped, so callers pass their own display text.
func Hyperlink(text string, link Link, enabled bool) string {
	url := link.URL()
	if !enabled || url == "" {
		return text
	}
	// OSC 8 ; params ; URI ST text OSC 8 ; ; ST. The string terminator (ST) form
	// is used instead of BEL so the sequence is unambiguous in captured output.
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// LaunchPlan is one resolved browser invocation. It is data only: constructing
// or inspecting a plan never starts a process, and tests assert the exact
// argument vector without launching a browser.
type LaunchPlan struct {
	Link     Link
	URL      string
	Program  string
	Args     []string
	Resolved bool
	Reason   string
}

// OpenResult is the outcome of an explicit Open call. Opened=false with an
// empty Err means the environment cannot open a browser (missing opener,
// headless, SSH or unsupported platform); Explanation always keeps the fixed
// URL available and tells the user how to open it locally.
type OpenResult struct {
	Plan   LaunchPlan
	Opened bool
	Err    error
}

// Explanation is a plain-language status that always includes the fixed URL
// when the destination was trusted. It never includes environment inventory,
// credentials or a source-provided URL.
func (r OpenResult) Explanation() string {
	url := r.Plan.URL
	if url == "" {
		return "This link destination is untrusted, so nothing was opened."
	}
	if r.Opened {
		return "Opened " + url + " in your browser."
	}

	reason := r.Plan.Reason
	if r.Err != nil {
		reason = "The browser opener failed: " + r.Err.Error()
	}
	if reason == "" {
		reason = "A browser could not be opened automatically"
	}
	return reason + ". Open this URL manually on a machine with a browser: " + url
}

// Opener is the injected platform-opener adapter. Every dependency is a field
// so tests and the host can supply an absolute resolution, a fixed environment
// and a recording launcher; the zero value is inert.
//
// Opener trust differs from companion trust: macOS uses the fixed /usr/bin/open
// path; Linux resolves the user-installed xdg-open once through LookPath and
// explicitly treats the user's PATH/program as trusted desktop configuration,
// not a security boundary. It never uses a shell, a source-configured
// executable or a working-directory-relative fallback.
type Opener struct {
	GOOS     string
	Getenv   func(string) string
	LookPath func(string) (string, error)
	Launch   func(program string, args []string) error

	once         sync.Once
	linuxProgram string
	linuxErr     error
}

// NewOpener returns the real platform opener. It performs no resolution and
// starts nothing until Open is called explicitly.
func NewOpener() *Opener {
	return &Opener{
		GOOS:     runtime.GOOS,
		Getenv:   os.Getenv,
		LookPath: exec.LookPath,
		Launch:   launchBrowser,
	}
}

// launchBrowser starts the resolved program with an argument vector and
// releases the child so the TUI does not wait for the browser to exit. There is
// deliberately no shell: program and args are passed to exec.Command as-is.
func launchBrowser(program string, args []string) error {
	cmd := exec.Command(program, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		return cmd.Process.Release()
	}
	return nil
}

// Resolve returns the launch plan for one trusted link without starting
// anything. It is safe to call from a render path: only Open launches.
func (o *Opener) Resolve(link Link) LaunchPlan {
	url := link.URL()
	if url == "" {
		return LaunchPlan{Link: link, Reason: "untrusted link destination"}
	}

	plan := LaunchPlan{Link: link, URL: url}

	if reason := o.environmentReason(); reason != "" {
		plan.Reason = reason
		return plan
	}

	switch o.goos() {
	case "darwin":
		// Fixed system path, never a PATH search.
		plan.Program = "/usr/bin/open"
		plan.Args = []string{url}
		plan.Resolved = true
	case "linux":
		program, err := o.resolveLinux()
		if err != nil {
			plan.Reason = err.Error()
			return plan
		}
		plan.Program = program
		plan.Args = []string{url}
		plan.Resolved = true
	default:
		plan.Reason = fmt.Sprintf("automatic browser opening is not supported on %s", o.goos())
	}
	return plan
}

// Open resolves and launches one trusted link. It is the only method that
// starts a process, and callers invoke it only after an explicit user action.
func (o *Opener) Open(link Link) OpenResult {
	plan := o.Resolve(link)
	result := OpenResult{Plan: plan}
	if !plan.Resolved {
		return result
	}

	launch := o.Launch
	if launch == nil {
		result.Err = errors.New("no browser launcher configured")
		return result
	}
	if err := launch(plan.Program, plan.Args); err != nil {
		result.Err = err
		return result
	}
	result.Opened = true
	return result
}

func (o *Opener) goos() string {
	if o.GOOS != "" {
		return o.GOOS
	}
	return runtime.GOOS
}

func (o *Opener) getenv() func(string) string {
	if o.Getenv != nil {
		return o.Getenv
	}
	return func(string) string { return "" }
}

// environmentReason refuses environments where an opener cannot reach the
// user's local browser. A remote SSH opener would run on the remote host, and
// headless Linux has no desktop session; both keep the URL copyable instead of
// claiming a launch that did not happen.
func (o *Opener) environmentReason() string {
	getenv := o.getenv()
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		if strings.TrimSpace(getenv(key)) != "" {
			return "this is a remote SSH session, so an opener would run on the remote host rather than your local computer"
		}
	}
	if o.goos() == "linux" &&
		strings.TrimSpace(getenv("DISPLAY")) == "" &&
		strings.TrimSpace(getenv("WAYLAND_DISPLAY")) == "" {
		return "no desktop session was detected (DISPLAY and WAYLAND_DISPLAY are unset)"
	}
	return ""
}

// resolveLinux performs the one permitted xdg-open lookup and caches it. A
// relative result is refused so a working-directory-relative program can never
// be executed.
func (o *Opener) resolveLinux() (string, error) {
	o.once.Do(func() {
		if o.LookPath == nil {
			o.linuxErr = errors.New("no opener resolver is configured; open the URL manually")
			return
		}
		program, err := o.LookPath("xdg-open")
		if err != nil {
			o.linuxErr = fmt.Errorf("xdg-open was not found on PATH; open the URL manually: %w", err)
			return
		}
		if !filepath.IsAbs(program) {
			o.linuxErr = fmt.Errorf("xdg-open resolved to the relative path %q; refusing a working-directory-relative opener", program)
			return
		}
		o.linuxProgram = program
	})
	return o.linuxProgram, o.linuxErr
}
