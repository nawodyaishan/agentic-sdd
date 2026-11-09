package main

import (
	"bytes"
	"strings"
	"testing"
)

// guideUpdateSpy records explicit update-check calls without networking.
type guideUpdateSpy struct {
	calls    int
	versions []string
	result   updateResult
}

func (s *guideUpdateSpy) check(current string) updateResult {
	s.calls++
	s.versions = append(s.versions, current)
	return s.result
}

func guideRuntimeWithSpy(installed string, spy *guideUpdateSpy) guideRuntime {
	return guideRuntime{installed: installed, check: spy.check}
}

// TestGuideUpdatesOnlyExplicitTerminalChoice proves no guide path checks
// automatically: only accepted terminal-stdout choice 5 makes one call.
func TestGuideUpdatesOnlyExplicitTerminalChoice(t *testing.T) {
	terminal := func() *Env {
		return &Env{Interactive: true, StdoutTTY: true, Width: 80}
	}

	t.Run("startup quit", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{result: updateResult{state: updateEqual}}
		code, stdout, stderr := runBrandedGuide(t, repo, home, "q\n", terminal(), guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || stderr != "" {
			t.Fatalf("exit %d stderr %q", code, stderr)
		}
		if spy.calls != 0 {
			t.Fatalf("update check calls = %d, want 0", spy.calls)
		}
		if strings.Contains(stdout, "You are up to date.") {
			t.Fatalf("update result printed without choice 5:\n%s", stdout)
		}
		assertNoWrites(t, home)
	})

	t.Run("startup eof", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{}
		code, _, _ := runBrandedGuide(t, repo, home, "", terminal(), guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 0 {
			t.Fatalf("exit %d calls %d, want 0/0", code, spy.calls)
		}
		assertNoWrites(t, home)
	})

	t.Run("about", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{}
		code, _, _ := runBrandedGuide(t, repo, home, "4\nq\n", terminal(), guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 0 {
			t.Fatalf("exit %d calls %d, want 0/0", code, spy.calls)
		}
		assertNoWrites(t, home)
	})

	t.Run("logs", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{}
		code, stdout, _ := runBrandedGuide(t, repo, home, "3\nq\n", terminal(), guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 0 {
			t.Fatalf("exit %d calls %d, want 0/0", code, spy.calls)
		}
		if !strings.Contains(stdout, "No install or restore logs found yet.") {
			t.Fatalf("logs path did not run:\n%s", stdout)
		}
		assertNoWrites(t, home)
	})

	t.Run("install decline", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{}
		code, stdout, _ := runBrandedGuide(t, repo, home, "1\nn\nn\n", terminal(), guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 0 {
			t.Fatalf("exit %d calls %d, want 0/0", code, spy.calls)
		}
		if !strings.Contains(stdout, "Cancelled.") {
			t.Fatalf("decline did not cancel:\n%s", stdout)
		}
		assertNoWrites(t, home)
	})

	t.Run("install confirm", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{}
		code, _, stderr := runBrandedGuide(t, repo, home, "1\nn\ny\n", terminal(), guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 0 {
			t.Fatalf("exit %d calls %d, want 0/0: %s", code, spy.calls, stderr)
		}
	})

	t.Run("plain terminal stdout rejects five", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{}
		env := &Env{Interactive: true, Width: 80}
		code, stdout, _ := runBrandedGuide(t, repo, home, "5\nq\n", env, guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 0 {
			t.Fatalf("exit %d calls %d, want 0/0", code, spy.calls)
		}
		if !strings.Contains(stdout, "Please enter 1, 2, 3, 4, or q.") {
			t.Fatalf("plain branch did not reject choice 5:\n%s", stdout)
		}
		if strings.Contains(stdout, "5) Check for CLI updates") {
			t.Fatalf("plain branch offered the terminal-only update choice:\n%s", stdout)
		}
		if strings.Contains(stdout, brandTagline) {
			t.Fatalf("plain branch rendered branding:\n%s", stdout)
		}
		assertNoWrites(t, home)
	})

	t.Run("version non-terminal", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := runNonInteractive([]string{"version"}, &stdout, &stderr)
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("version exit %d stderr %q", code, stderr.String())
		}
		if strings.Contains(stdout.String(), "You are up to date.") || strings.Contains(stdout.String(), "Could not check for updates.") {
			t.Fatalf("version performed or displayed an update check:\n%s", stdout.String())
		}
	})
}

// TestGuideUpdatesChoiceFiveAndRepeat covers one call per explicit choice,
// repeated explicit choices, dev builds, and narrow no-color terminals.
func TestGuideUpdatesChoiceFiveAndRepeat(t *testing.T) {
	t.Run("one explicit choice", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{result: updateResult{state: updateEqual}}
		env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
		code, stdout, stderr := runBrandedGuide(t, repo, home, "5\nq\n", env, guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || stderr != "" {
			t.Fatalf("exit %d stderr %q", code, stderr)
		}
		if spy.calls != 1 || len(spy.versions) != 1 || spy.versions[0] != "1.0.0" {
			t.Fatalf("calls=%d versions=%v, want one call with 1.0.0", spy.calls, spy.versions)
		}
		if n := strings.Count(stdout, "You are up to date."); n != 1 {
			t.Fatalf("result shown %d times, want once:\n%s", n, stdout)
		}
		if n := strings.Count(stdout, "Main menu"); n != 2 {
			t.Fatalf("menu shown %d times, want 2:\n%s", n, stdout)
		}
		if n := strings.Count(stdout, brandTagline); n != 1 {
			t.Fatalf("intro replayed after update check: tagline shown %d times:\n%s", n, stdout)
		}
		assertNoWrites(t, home)
	})

	t.Run("two explicit choices", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{result: updateResult{state: updateEqual}}
		env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
		code, stdout, _ := runBrandedGuide(t, repo, home, "5\n5\nq\n", env, guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 2 {
			t.Fatalf("exit %d calls %d, want 0/2", code, spy.calls)
		}
		if n := strings.Count(stdout, "You are up to date."); n != 2 {
			t.Fatalf("result shown %d times, want 2:\n%s", n, stdout)
		}
	})

	t.Run("dev build still requests", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{result: updateResult{state: updateUnavailable, latest: "2.0.0"}}
		env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
		code, stdout, _ := runBrandedGuide(t, repo, home, "5\nq\n", env, guideRuntimeWithSpy("dev", spy))
		if code != 0 || spy.calls != 1 {
			t.Fatalf("exit %d calls %d, want 0/1", code, spy.calls)
		}
		if len(spy.versions) != 1 || spy.versions[0] != "dev" {
			t.Fatalf("checker received %v, want dev", spy.versions)
		}
		if !strings.Contains(stdout, "version comparison unavailable") || strings.Contains(stdout, "Update available") {
			t.Fatalf("dev result claimed a comparison:\n%s", stdout)
		}
	})

	t.Run("narrow no-color terminal accepts five", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{result: updateResult{state: updateEqual}}
		env := &Env{Interactive: true, StdoutTTY: true, Width: 40}
		code, stdout, _ := runBrandedGuide(t, repo, home, "5\nq\n", env, guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 1 {
			t.Fatalf("exit %d calls %d, want 0/1", code, spy.calls)
		}
		if !strings.Contains(stdout, "You are up to date.") {
			t.Fatalf("narrow update result missing:\n%s", stdout)
		}
	})
}

// TestGuideUpdatesStates renders each injected state through the guide and
// checks the fixed update script text, plain output and no ANSI.
func TestGuideUpdatesStates(t *testing.T) {
	tests := []struct {
		name   string
		result updateResult
		want   []string
	}{
		{
			name:   "newer",
			result: updateResult{state: updateNewer, current: "0.9.0", latest: "0.10.0"},
			want: []string{
				"Update available: 0.9.0 -> 0.10.0",
				"For Homebrew installations — copy and run manually",
				updateBrewCommand,
				updateVerifyCommand,
				updateReleasesURL,
				updatePreviewCommand,
				updateApplyCommand,
			},
		},
		{"equal", updateResult{state: updateEqual}, []string{"You are up to date."}},
		{"ahead", updateResult{state: updateAhead}, []string{"Installed version is newer than the latest published release."}},
		{"unavailable", updateResult{state: updateUnavailable}, []string{"version comparison unavailable", updateReleasesURL}},
		{"failed", updateResult{state: updateFailed}, []string{"Could not check for updates.", updateReleasesURL}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, home := guideFixture(t)
			spy := &guideUpdateSpy{result: tc.result}
			env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
			code, stdout, stderr := runBrandedGuide(t, repo, home, "5\nq\n", env, guideRuntimeWithSpy("0.9.0", spy))
			if code != 0 || stderr != "" {
				t.Fatalf("exit %d stderr %q", code, stderr)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout, want) {
					t.Fatalf("state %s output missing %q:\n%s", tc.name, want, stdout)
				}
			}
			if strings.ContainsRune(stdout, 0x1b) {
				t.Fatalf("color-disabled update result emitted ANSI:\n%q", stdout)
			}
			if n := strings.Count(stdout, "Main menu"); n != 2 {
				t.Fatalf("menu shown %d times after update result, want 2:\n%s", n, stdout)
			}
			assertNoWrites(t, home)
		})
	}
}

// TestGuideUpdatesTracePrivacyAndLegacyBytes checks the additive result screen
// with no update payload in the trace, and that plain stdout never gains the
// update option or a network call.
func TestGuideUpdatesTracePrivacyAndLegacyBytes(t *testing.T) {
	t.Run("trace has only the result screen", func(t *testing.T) {
		repo, home := guideFixture(t)
		var traceBuf bytes.Buffer
		tracer := newTracer(&traceBuf, nil, true)
		spy := &guideUpdateSpy{result: updateResult{state: updateNewer, current: "77.1.2", latest: "77.1.3"}}
		env := &Env{Interactive: true, StdoutTTY: true, Width: 80, Trace: tracer}
		code, _, stderr := runBrandedGuide(t, repo, home, "5\nq\n", env, guideRuntimeWithSpy("77.1.2", spy))
		if code != 0 {
			t.Fatalf("exit %d: %s", code, stderr)
		}
		events := decodeTrace(t, traceBuf.String())
		choiceIndex, screenIndex := -1, -1
		for i, event := range events {
			if event["event"] == "op.result" {
				t.Fatalf("update check emitted op.result: %v", event)
			}
			if event["event"] == "prompt.answered" && event["prompt"] == "menu" && event["choice"] == "5" {
				choiceIndex = i
			}
			if event["event"] == "screen.shown" && event["screen"] == "updates.result" {
				screenIndex = i
				if len(event) != 5 {
					t.Fatalf("updates.result event has unexpected fields: %v", event)
				}
				for _, key := range []string{"version", "latest", "current", "url", "error", "result"} {
					if _, ok := event[key]; ok {
						t.Fatalf("updates.result event leaked %q: %v", key, event)
					}
				}
			}
		}
		if choiceIndex < 0 || screenIndex < 0 {
			t.Fatalf("missing choice 5 or updates.result event:\n%s", traceBuf.String())
		}
		if screenIndex < choiceIndex {
			t.Fatalf("updates.result shown before the explicit choice: %v", events)
		}
		raw := traceBuf.String()
		for _, forbidden := range []string{"77.1.3", "77.1.2", updateReleasesURL, updateReleaseAPIURL, "Update available", "brew", "rate limit"} {
			if strings.Contains(raw, forbidden) {
				t.Fatalf("trace leaked %q:\n%s", forbidden, raw)
			}
		}
		assertNoWrites(t, home)
	})

	t.Run("plain branch rejects five without checking", func(t *testing.T) {
		repo, home := guideFixture(t)
		spy := &guideUpdateSpy{result: updateResult{state: updateNewer, current: "1.0.0", latest: "2.0.0"}}
		env := &Env{Interactive: true, Width: 80}
		code, stdout, _ := runBrandedGuide(t, repo, home, "5\nq\n", env, guideRuntimeWithSpy("1.0.0", spy))
		if code != 0 || spy.calls != 0 {
			t.Fatalf("exit %d calls %d, want 0/0", code, spy.calls)
		}
		if strings.Contains(stdout, "Update available") || strings.Contains(stdout, updateBrewCommand) {
			t.Fatalf("plain branch executed or rendered an update:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Please enter 1, 2, 3, 4, or q.") {
			t.Fatalf("plain branch did not use the legacy invalid message:\n%s", stdout)
		}
		assertNoWrites(t, home)
	})
}
