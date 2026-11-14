package tui_test

// Feature 020 B6/T11 — fixed documentation links and their visible fallback.
//
// The registry helpers (DocsLinks, DocsFallbackLine) and the seven LinkDocs*
// constants arrive with the integration merge, so these tests must keep
// compiling in a pre-integration worktree while still pinning the frozen
// contract. They do that two ways:
//
//   - the ordered, trusted destinations are observed through the Guide screen,
//     which is the registry's only consumer, and through the injected opener;
//   - the registry source is checked statically for the frozen constant values,
//     the DocsLinks order and DocsFallbackLine's refusal rule.
//
// When the registry is absent every contract test fails with an explicit
// "integration-pending:" message. Nothing here is skipped.

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"agentic-sdd/experiments/tui"
	"github.com/charmbracelet/x/ansi"
)

const docsPending = "integration-pending: the Feature 020 documentation link registry is absent in this worktree (expected docs_links.go plus the seven LinkDocs* constants, Trusted() entries and Labels in opener.go)"

const docsBlobBase = "https://github.com/nawodyaishan/agentic-sdd/blob/main/website/src/content/docs/"

// docsDestination is one frozen registry entry: its display label and the
// canonical page source relative to docsBlobBase.
type docsDestination struct {
	Label string
	Path  string
}

var docsDestinations = []docsDestination{
	{Label: "Getting started", Path: "getting-started/first-five-minutes.md"},
	{Label: "CLI reference", Path: "reference/cli.md"},
	{Label: "TUI reference", Path: "reference/tui.md"},
	{Label: "Skills", Path: "how-to/skills.md"},
	{Label: "Updating and recovery", Path: "how-to/updates-and-recovery.md"},
	{Label: "Troubleshooting", Path: "operations/troubleshooting.md"},
	{Label: "Contributing / docs changes", Path: "contributing/docs-and-changes.md"},
}

var docsLinkConstNames = []string{
	"LinkDocsGettingStarted",
	"LinkDocsCLI",
	"LinkDocsTUI",
	"LinkDocsSkills",
	"LinkDocsUpdates",
	"LinkDocsTroubleshooting",
	"LinkDocsContributing",
}

// docsLaunchRecorder records every invented launch. It never starts a process.
type docsLaunchRecorder struct {
	calls int
	urls  []string
	err   error
}

func (r *docsLaunchRecorder) launch(program string, args []string) error {
	r.calls++
	if len(args) > 0 {
		r.urls = append(r.urls, args[0])
	}
	return r.err
}

// docsDesktopOpener resolves the fixed absolute xdg-open path in a desktop
// session; every dependency is injected.
func docsDesktopOpener(rec *docsLaunchRecorder) *tui.Opener {
	return &tui.Opener{
		GOOS: "linux",
		Getenv: func(key string) string {
			if key == "DISPLAY" {
				return ":0"
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "/usr/bin/xdg-open", nil },
		Launch:   rec.launch,
	}
}

func docsHeadlessOpener(rec *docsLaunchRecorder) *tui.Opener {
	return &tui.Opener{
		GOOS:     "linux",
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "/usr/bin/xdg-open", nil },
		Launch:   rec.launch,
	}
}

func docsSSHOpener(rec *docsLaunchRecorder) *tui.Opener {
	return &tui.Opener{
		GOOS: "linux",
		Getenv: func(key string) string {
			switch key {
			case "DISPLAY":
				return ":0"
			case "SSH_CONNECTION":
				return "10.0.0.1 1234 10.0.0.2 22"
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "/usr/bin/xdg-open", nil },
		Launch:   rec.launch,
	}
}

func docsUnsupportedOpener(rec *docsLaunchRecorder) *tui.Opener {
	return &tui.Opener{
		GOOS:     "plan9",
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "", errors.New("no opener") },
		Launch:   rec.launch,
	}
}

func docsNilLaunchOpener(rec *docsLaunchRecorder) *tui.Opener {
	opener := docsDesktopOpener(rec)
	opener.Launch = nil
	return opener
}

func docsFailingOpener(rec *docsLaunchRecorder) *tui.Opener {
	opener := docsDesktopOpener(rec)
	rec.err = errors.New("no browser available")
	return opener
}

func docsURLs() []string {
	urls := make([]string, len(docsDestinations))
	for i, dest := range docsDestinations {
		urls[i] = docsBlobBase + dest.Path
	}
	return urls
}

// docsSeen is one documentation destination as the Guide actually showed it.
type docsSeen struct {
	Label string
	URL   string
}

// docsCompact removes wrapping and selection indentation so a URL remains
// searchable when Guide wraps it in a narrow or plain terminal.
func docsCompact(text string) string {
	return strings.Join(strings.Fields(ansi.Strip(text)), "")
}

// docsVisible picks the one expected destination whose literal URL is on
// screen. Guide renders the fallback line only for the highlighted row, so at
// most one destination can match.
func docsVisible(view string) (string, bool) {
	compact := docsCompact(view)
	for _, url := range docsURLs() {
		if strings.Contains(compact, url) {
			return url, true
		}
	}
	return "", false
}

// docsWalk highlights each documentation row in turn, top to bottom, and
// returns the label and literal URL Guide displayed for it. It never assumes
// how many guided-action rows precede the documentation section.
func docsWalk(t *testing.T) []docsSeen {
	t.Helper()
	model := tui.NewGuideModel(tui.TUIConfig{Width: 200, Height: 48, Interactive: false})
	var seen []docsSeen
	for step := 0; step < 64 && len(seen) < len(docsDestinations); step++ {
		view := model.View().Content
		if url, ok := docsVisible(view); ok {
			label := model.SelectedEntry().Title
			if len(seen) == 0 || seen[len(seen)-1].URL != url {
				seen = append(seen, docsSeen{Label: label, URL: url})
			}
		}
		next, _ := model.Update(runes("j"))
		model = next
	}
	return seen
}

// TestDocsRegistryOrderLabelsAndTrust walks the Guide documentation section and
// pins the frozen registry: seven destinations, in order, every one a trusted
// HTTPS GitHub blob URL with the frozen label.
func TestDocsRegistryOrderLabelsAndTrust(t *testing.T) {
	seen := docsWalk(t)
	if len(seen) != len(docsDestinations) {
		t.Fatalf("%s: Guide exposed %d documentation destinations, want %d", docsPending, len(seen), len(docsDestinations))
	}

	want := docsURLs()
	for i, dest := range docsDestinations {
		if got := seen[i].Label; got != dest.Label {
			t.Errorf("documentation row %d label = %q, want %q", i, got, dest.Label)
		}
		if got := seen[i].URL; got != want[i] {
			t.Errorf("documentation row %d URL = %q, want %q", i, got, want[i])
		}
		if !strings.HasPrefix(seen[i].URL, "https://") {
			t.Errorf("documentation row %d URL = %q, want HTTPS", i, seen[i].URL)
		}
		if strings.Contains(seen[i].URL, "github.io") {
			t.Errorf("documentation row %d URL = %q, want the GitHub blob fallback, not a site URL", i, seen[i].URL)
		}
		link := tui.Link(seen[i].URL)
		if !link.Trusted() {
			t.Errorf("documentation row %d URL %q is not a trusted Link", i, seen[i].URL)
		}
		if got := link.URL(); got != seen[i].URL {
			t.Errorf("Link(%q).URL() = %q, want the same literal URL", seen[i].URL, got)
		}
		if got := link.Label(); got != dest.Label {
			t.Errorf("Link(%q).Label() = %q, want %q", seen[i].URL, got, dest.Label)
		}
	}
}

// TestDocsRegistrySourceContract pins the parts of the frozen interface that
// the Guide screen cannot show: the literal constant values, the registry order
// in DocsLinks, and DocsFallbackLine's rule that an untrusted or unlabeled link
// renders nothing.
func TestDocsRegistrySourceContract(t *testing.T) {
	values := docsLinkConstants(t)
	for i, name := range docsLinkConstNames {
		value, ok := values[name]
		if !ok {
			t.Fatalf("%s: opener.go has no %s constant", docsPending, name)
		}
		want := docsURLs()[i]
		if value != want {
			t.Errorf("%s = %q, want %q", name, value, want)
		}
		if !strings.HasPrefix(value, "https://github.com/nawodyaishan/agentic-sdd/") {
			t.Errorf("%s = %q, want an immutable HTTPS GitHub destination", name, value)
		}
	}

	order := docsLinksOrder(t)
	if strings.Join(order, ",") != strings.Join(docsLinkConstNames, ",") {
		t.Fatalf("DocsLinks() order = %v, want %v", order, docsLinkConstNames)
	}

	docsFallbackLineContract(t)
}

// docsLinkConstants reads the seven frozen Link constants from opener.go. The
// file exists before integration, so a missing constant is a precise failure
// rather than a build break.
func docsLinkConstants(t *testing.T) map[string]string {
	t.Helper()
	file := docsParse(t, "opener.go")
	values := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || value.Type == nil {
				continue
			}
			typeName, ok := value.Type.(*ast.Ident)
			if !ok || typeName.Name != "Link" {
				continue
			}
			for i, name := range value.Names {
				if i >= len(value.Values) {
					continue
				}
				lit, ok := value.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				unquoted, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", name.Name, err)
				}
				values[name.Name] = unquoted
			}
		}
	}
	return values
}

// docsLinksOrder reads the returned constant identifiers from DocsLinks, which
// is what fixes the on-screen order.
func docsLinksOrder(t *testing.T) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "docs_links.go", nil, 0)
	if err != nil {
		t.Fatalf("%s: docs_links.go is not present or does not parse: %v", docsPending, err)
	}
	fn := docsFunc(file, "DocsLinks")
	if fn == nil {
		t.Fatalf("%s: docs_links.go defines no DocsLinks function", docsPending)
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		lit, ok := ret.Results[0].(*ast.CompositeLit)
		if !ok {
			continue
		}
		order := make([]string, 0, len(lit.Elts))
		for _, elt := range lit.Elts {
			ident, ok := elt.(*ast.Ident)
			if !ok {
				t.Fatalf("DocsLinks returns a non-identifier element %T; the registry must be fixed constants", elt)
			}
			order = append(order, ident.Name)
		}
		return order
	}
	t.Fatalf("%s: DocsLinks has no fixed composite-literal return", docsPending)
	return nil
}

// docsFallbackLineContract checks the fallback helper's signature and refusal
// path without calling it: a direct call would not compile in a worktree where
// the helper is not integrated yet.
func docsFallbackLineContract(t *testing.T) {
	t.Helper()
	file := docsParse(t, "docs_links.go")
	fn := docsFunc(file, "DocsFallbackLine")
	if fn == nil {
		t.Fatalf("%s: docs_links.go defines no DocsFallbackLine function", docsPending)
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) != 1 {
		t.Fatalf("DocsFallbackLine must take exactly one Link argument")
	}
	paramType, ok := fn.Type.Params.List[0].Type.(*ast.Ident)
	if !ok || paramType.Name != "Link" {
		t.Fatalf("DocsFallbackLine parameter type = %T, want Link", fn.Type.Params.List[0].Type)
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		t.Fatalf("DocsFallbackLine must return exactly one string")
	}
	resultType, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	if !ok || resultType.Name != "string" {
		t.Fatalf("DocsFallbackLine result type = %T, want string", fn.Type.Results.List[0].Type)
	}

	var selectors []string
	var emptyReturn bool
	var stringConversion bool
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			selectors = append(selectors, typed.Sel.Name)
		case *ast.ReturnStmt:
			for _, result := range typed.Results {
				if lit, ok := result.(*ast.BasicLit); ok && lit.Kind == token.STRING && lit.Value == `""` {
					emptyReturn = true
				}
			}
		case *ast.CallExpr:
			if ident, ok := typed.Fun.(*ast.Ident); ok && ident.Name == "string" {
				for _, arg := range typed.Args {
					if argIdent, ok := arg.(*ast.Ident); ok && argIdent.Name == "link" {
						stringConversion = true
					}
				}
			}
		}
		return true
	})
	if stringConversion {
		t.Fatal("DocsFallbackLine converts the raw Link with string(link); it must use URL() and Label() only")
	}
	if !docsContains(selectors, "URL") || !docsContains(selectors, "Label") {
		t.Fatalf("DocsFallbackLine must render from URL() and Label(); selectors = %v", selectors)
	}
	if !emptyReturn {
		t.Fatal("DocsFallbackLine must return an empty string for an untrusted or unlabeled link")
	}

	// No documentation URL may be composed in the registry file: the fixed
	// values live in opener.go and a caller must never build one at run time.
	ast.Inspect(file, func(node ast.Node) bool {
		if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING && strings.Contains(strings.ToLower(lit.Value), "http") {
			t.Errorf("docs_links.go contains the URL literal %s; destinations must stay in opener.go", lit.Value)
		}
		return true
	})
}

func docsParse(t *testing.T, name string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
	if err != nil {
		t.Fatalf("%s: %s is not present or does not parse: %v", docsPending, name, err)
	}
	return file
}

func docsFunc(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

func docsContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// TestDocsUntrustedDestinationsRefused proves no arbitrary string can become a
// documentation URL: every near-miss is untrusted, URL() stays empty, the
// opener refuses it without launching, and the hyperlink helper stays plain.
func TestDocsUntrustedDestinationsRefused(t *testing.T) {
	trusted := docsURLs()[0]
	cases := []tui.Link{
		"",
		"https://example.invalid/docs",
		tui.Link("http://github.com/nawodyaishan/agentic-sdd/blob/main/website/src/content/docs/reference/cli.md"),
		tui.Link(docsBlobBase + "../secrets.md"),
		tui.Link(docsBlobBase + "reference/cli.md#fragment"),
		tui.Link(" " + trusted),
		tui.Link(trusted + " "),
		tui.Link(strings.ToUpper(trusted)),
		tui.Link(strings.Replace(trusted, "https://", "http://", 1)),
	}
	for _, link := range cases {
		if link.Trusted() {
			t.Errorf("Link(%q) is trusted, want an untrusted arbitrary string", string(link))
		}
		if got := link.URL(); got != "" {
			t.Errorf("Link(%q).URL() = %q, want empty", string(link), got)
		}
		if got := link.Label(); got != "" {
			t.Errorf("Link(%q).Label() = %q, want empty", string(link), got)
		}
		if got := tui.Hyperlink("docs", link, true); got != "docs" {
			t.Errorf("Hyperlink with untrusted %q = %q, want plain text", string(link), got)
		}

		rec := &docsLaunchRecorder{}
		result := docsDesktopOpener(rec).Open(link)
		if result.Opened {
			t.Errorf("Open(%q) reported success for an untrusted destination", string(link))
		}
		if rec.calls != 0 {
			t.Errorf("Open(%q) launched the opener %d times", string(link), rec.calls)
		}
		if !strings.Contains(result.Explanation(), "untrusted") {
			t.Errorf("Explanation() = %q, want the untrusted refusal", result.Explanation())
		}
	}
}

// TestDocsFallbackRetainedOnEveryNonLaunchPath proves that no opener
// environment claims success: a nil launcher, a launcher failure, headless
// Linux, an SSH session and an unsupported platform all keep the literal
// registry URL in the explanation for manual opening.
func TestDocsFallbackRetainedOnEveryNonLaunchPath(t *testing.T) {
	seen := docsWalk(t)
	if len(seen) != len(docsDestinations) {
		t.Fatalf("%s: Guide exposed %d documentation destinations, want %d", docsPending, len(seen), len(docsDestinations))
	}

	type environment struct {
		name      string
		build     func(rec *docsLaunchRecorder) *tui.Opener
		wantCalls int
	}
	environments := []environment{
		{"headless", docsHeadlessOpener, 0},
		{"ssh", docsSSHOpener, 0},
		{"unsupported platform", docsUnsupportedOpener, 0},
		{"no launcher", docsNilLaunchOpener, 0},
		{"launcher failure", docsFailingOpener, 1},
	}

	for _, url := range docsURLs() {
		link := tui.Link(url)
		if !link.Trusted() {
			t.Fatalf("%q is not a trusted registry destination", url)
		}

		// The nil-opener path builds this result directly; without an injected
		// opener there is no process seam, and the URL must stay available.
		nilResult := tui.OpenResult{Plan: tui.LaunchPlan{
			Link:   link,
			URL:    url,
			Reason: "no browser opener is configured in this session",
		}}
		if nilResult.Opened {
			t.Fatal("a nil opener must not report success")
		}
		docsAssertRetainsURL(t, "nil opener", nilResult, url)

		for _, env := range environments {
			rec := &docsLaunchRecorder{}
			result := env.build(rec).Open(link)
			if result.Opened {
				t.Errorf("%s: Open(%q) claimed success", env.name, url)
			}
			if rec.calls != env.wantCalls {
				t.Errorf("%s: launcher calls = %d, want %d", env.name, rec.calls, env.wantCalls)
			}
			docsAssertRetainsURL(t, env.name, result, url)
		}
	}
}

func docsAssertRetainsURL(t *testing.T, context string, result tui.OpenResult, url string) {
	t.Helper()
	if result.Plan.URL != url {
		t.Errorf("%s: Plan.URL = %q, want %q", context, result.Plan.URL, url)
	}
	explanation := result.Explanation()
	if !strings.Contains(explanation, url) {
		t.Errorf("%s: Explanation() = %q, want the literal URL", context, explanation)
	}
	if strings.HasPrefix(explanation, "Opened ") {
		t.Errorf("%s: Explanation() = %q, must not claim the page opened", context, explanation)
	}
	if !strings.Contains(strings.ToLower(explanation), "manually") {
		t.Errorf("%s: Explanation() = %q, want manual-opening guidance", context, explanation)
	}
}
