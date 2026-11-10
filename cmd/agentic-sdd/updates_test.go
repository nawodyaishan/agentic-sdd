package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"agentic-sdd/internal/appops"
)

// updateTransportFunc lets each test answer one request without networking.
type updateTransportFunc func(*http.Request) (*http.Response, error)

func (f updateTransportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// updateTestResponse builds a minimal response; production code must close it.
func updateTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// updateTrackingBody records Close so body-close behaviour is observable.
type updateTrackingBody struct {
	*strings.Reader
	closed bool
}

func (b *updateTrackingBody) Close() error {
	b.closed = true
	return nil
}

func updateTestCheck(t *testing.T, transport http.RoundTripper, current string) updateResult {
	t.Helper()
	return checkLatestRelease(&http.Client{Transport: transport, Timeout: 5 * time.Second}, current)
}

func TestUpdatesClientConfig(t *testing.T) {
	client := newUpdateClient()
	if client.Timeout != 5*time.Second {
		t.Fatalf("client timeout = %s, want %s", client.Timeout, 5*time.Second)
	}
	if client.CheckRedirect == nil {
		t.Fatal("client has no redirect policy")
	}
	req, err := http.NewRequest(http.MethodGet, updateReleaseAPIURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(req, nil); err != http.ErrUseLastResponse {
		t.Fatalf("redirect policy error = %v, want http.ErrUseLastResponse", err)
	}
}

func TestUpdatesRequestContract(t *testing.T) {
	var got *http.Request
	calls := 0
	result := updateTestCheck(t, updateTransportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		got = req
		return updateTestResponse(http.StatusOK, `{"tag_name":"v1.2.3"}`), nil
	}), "1.2.3")

	if calls != 1 {
		t.Fatalf("requests = %d, want 1", calls)
	}
	if got == nil {
		t.Fatal("no request captured")
	}
	if got.Method != http.MethodGet {
		t.Errorf("method = %q, want GET", got.Method)
	}
	if got.URL.String() != updateReleaseAPIURL {
		t.Errorf("URL = %q, want fixed %q", got.URL.String(), updateReleaseAPIURL)
	}
	if got.URL.Scheme != "https" {
		t.Errorf("scheme = %q, want https", got.URL.Scheme)
	}
	if got.URL.Host != "api.github.com" {
		t.Errorf("host = %q, want api.github.com", got.URL.Host)
	}
	if got.URL.Path != "/repos/nawodyaishan/agentic-sdd/releases/latest" {
		t.Errorf("path = %q", got.URL.Path)
	}
	if got.URL.RawQuery != "" {
		t.Errorf("query = %q, want empty", got.URL.RawQuery)
	}
	if ua := got.Header.Get("User-Agent"); ua != updateUserAgent {
		t.Errorf("User-Agent = %q, want %q", ua, updateUserAgent)
	}
	if accept := got.Header.Get("Accept"); accept != "application/vnd.github+json" {
		t.Errorf("Accept = %q, want application/vnd.github+json", accept)
	}
	for _, header := range []string{"Authorization", "Cookie", "X-Api-Key"} {
		if value := got.Header.Get(header); value != "" {
			t.Errorf("%s = %q, want empty", header, value)
		}
	}
	if got.Body != nil {
		t.Errorf("request body = %v, want none", got.Body)
	}
	if result.state != updateEqual || result.current != "1.2.3" || result.latest != "1.2.3" {
		t.Errorf("result = %+v, want equal 1.2.3", result)
	}
}

func TestUpdatesResultStates(t *testing.T) {
	tests := []struct {
		name    string
		current string
		tag     string
		want    updateState
	}{
		{"newer", "0.9.0", "v0.10.0", updateNewer},
		{"equal with optional v", "v1.2.3", "1.2.3", updateEqual},
		{"ahead", "2.0.0", "v1.99.99", updateAhead},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(http.StatusOK, `{"tag_name":"`+tc.tag+`"}`), nil
			}), tc.current)
			if result.state != tc.want {
				t.Fatalf("state = %d, want %d", result.state, tc.want)
			}
			wantCurrent := strings.TrimPrefix(tc.current, "v")
			wantLatest := strings.TrimPrefix(tc.tag, "v")
			if result.current != wantCurrent || result.latest != wantLatest {
				t.Errorf("versions = %q -> %q, want %q -> %q", result.current, result.latest, wantCurrent, wantLatest)
			}
		})
	}
}

func TestUpdatesComparator(t *testing.T) {
	hugeNines := strings.Repeat("9", 100)
	hugeOneAndZeros := "1" + strings.Repeat("0", 99)
	longNines := strings.Repeat("9", 300)

	tests := []struct {
		name    string
		current string
		tag     string
		want    updateState
	}{
		{"numeric not lexical", "0.9.0", "v0.10.0", updateNewer},
		{"equal", "v1.2.3", "1.2.3", updateEqual},
		{"ahead", "2.0.0", "v1.99.99", updateAhead},
		{"major newer", "1.9.9", "v2.0.0", updateNewer},
		{"patch numeric", "1.0.9", "v1.0.10", updateNewer},
		{"zero components", "0.0.0", "v0.0.0", updateEqual},
		{"huge current ahead", "1." + hugeNines + ".0", "v1." + hugeOneAndZeros + ".0", updateAhead},
		{"huge latest newer", "1." + hugeOneAndZeros + ".0", "v1." + hugeNines + ".0", updateNewer},
		{"huge equal", "1." + longNines + ".0", "v1." + longNines + ".0", updateEqual},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(http.StatusOK, `{"tag_name":"`+tc.tag+`"}`), nil
			}), tc.current)
			if result.state != tc.want {
				t.Fatalf("state = %d, want %d", result.state, tc.want)
			}
			if result.current != strings.TrimPrefix(tc.current, "v") {
				t.Errorf("current = %q", result.current)
			}
			if result.latest != strings.TrimPrefix(tc.tag, "v") {
				t.Errorf("latest = %q", result.latest)
			}
		})
	}
}

func TestUpdatesInvalidCurrentUnavailable(t *testing.T) {
	currents := []string{
		"dev",
		"",
		"1.0.0-rc.1",
		"1.0",
		"v1.0.0\n",
		"vv1.0.0",
		" 1.0.0",
		"1.0.01",
		"1.0.0.0",
		"V1.0.0",
		"1.0.0+build",
		"1.0.0\x00",
		"١.0.0",
	}
	for i, current := range currents {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			calls := 0
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return updateTestResponse(http.StatusOK, `{"tag_name":"v2.0.0"}`), nil
			}), current)
			if calls != 1 {
				t.Fatalf("requests = %d, want 1", calls)
			}
			if result.state != updateUnavailable {
				t.Fatalf("state = %d, want unavailable", result.state)
			}
			if result.current != "" {
				t.Errorf("current = %q, want empty for invalid input", result.current)
			}
			if result.latest != "2.0.0" {
				t.Errorf("latest = %q, want validated 2.0.0", result.latest)
			}
			var out bytes.Buffer
			printUpdateResult(&out, result)
			if strings.Contains(out.String(), "Update available") {
				t.Errorf("unavailable result claims an update:\n%s", out.String())
			}
			if !strings.Contains(out.String(), updateReleasesURL) {
				t.Errorf("unavailable result lacks fixed releases URL:\n%s", out.String())
			}
		})
	}
}

func TestUpdatesFailurePrecedesInvalidCurrent(t *testing.T) {
	result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
		return updateTestResponse(http.StatusOK, `{"tag_name":`), nil
	}), "dev")
	if result.state != updateFailed {
		t.Fatalf("state = %d, want failed", result.state)
	}
	if result.current != "" || result.latest != "" {
		t.Fatalf("versions = %q -> %q, want empty on failure", result.current, result.latest)
	}
}

func TestUpdatesMaliciousFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"control in tag", `{"tag_name":"v1.0.0\u001b[31m"}`},
		{"newline in tag", `{"tag_name":"v1.0.0\n"}`},
		{"uppercase v", `{"tag_name":"V1.0.0"}`},
		{"leading zero", `{"tag_name":"v01.0.0"}`},
		{"leading zero patch", `{"tag_name":"v1.0.01"}`},
		{"missing patch", `{"tag_name":"v1.0"}`},
		{"extra component", `{"tag_name":"v1.0.0.0"}`},
		{"prerelease suffix", `{"tag_name":"v1.0.0-rc.1"}`},
		{"build suffix", `{"tag_name":"v1.0.0+build"}`},
		{"whitespace suffix", `{"tag_name":"v1.0.0 "}`},
		{"missing tag", `{}`},
		{"wrong tag type", `{"tag_name":1}`},
		{"draft release", `{"tag_name":"v1.0.0","draft":true}`},
		{"prerelease release", `{"tag_name":"v1.0.0","prerelease":true}`},
		{"malformed json", `{"tag_name":`},
		{"json array", `[]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(http.StatusOK, tc.body), nil
			}), "1.0.0")
			if result.state != updateFailed {
				t.Fatalf("state = %d, want failed", result.state)
			}
			if result.current != "" || result.latest != "" {
				t.Fatalf("versions = %q -> %q, want empty on failure", result.current, result.latest)
			}
			var out bytes.Buffer
			printUpdateResult(&out, result)
			if strings.ContainsRune(out.String(), '\x1b') {
				t.Errorf("renderer emitted ESC for malformed input: %q", out.String())
			}
			if !strings.Contains(out.String(), "Could not check for updates.") || !strings.Contains(out.String(), updateReleasesURL) {
				t.Errorf("failure output missing generic text or fixed URL:\n%s", out.String())
			}
		})
	}
}

func TestUpdatesIgnoresUntrustedReleaseFields(t *testing.T) {
	body := `{"tag_name":"v1.0.0","body":"\u001b[31m evil https://evil.example/secret","assets":[{"name":"evil"}]}`
	result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
		return updateTestResponse(http.StatusOK, body), nil
	}), "0.9.0")
	if result.state != updateNewer {
		t.Fatalf("state = %d, want newer", result.state)
	}
	if result.current != "0.9.0" || result.latest != "1.0.0" {
		t.Fatalf("versions = %q -> %q", result.current, result.latest)
	}
	var out bytes.Buffer
	printUpdateResult(&out, result)
	if strings.Contains(out.String(), "evil") || strings.ContainsRune(out.String(), '\x1b') {
		t.Errorf("untrusted release text leaked into output:\n%s", out.String())
	}
}

func TestUpdatesNon200(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(status, `server error \u001b[31m https://evil.example/secret`), nil
			}), "1.0.0")
			if result.state != updateFailed {
				t.Fatalf("state = %d, want failed", result.state)
			}
			var out bytes.Buffer
			printUpdateResult(&out, result)
			if strings.Contains(out.String(), "evil") || strings.ContainsRune(out.String(), '\x1b') {
				t.Errorf("server text leaked into output:\n%s", out.String())
			}
		})
	}
}

func TestUpdatesBodyLimit(t *testing.T) {
	base := `{"tag_name":"v1.0.0"}`
	if len(base) >= maxUpdateBodyBytes {
		t.Fatal("test response base exceeds the body cap")
	}
	exact := base + strings.Repeat(" ", maxUpdateBodyBytes-len(base))
	over := base + strings.Repeat(" ", maxUpdateBodyBytes-len(base)+1)

	t.Run("exact limit accepted", func(t *testing.T) {
		result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
			return updateTestResponse(http.StatusOK, exact), nil
		}), "1.0.0")
		if result.state != updateEqual {
			t.Fatalf("state = %d, want equal", result.state)
		}
	})
	t.Run("one byte over rejected", func(t *testing.T) {
		result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
			return updateTestResponse(http.StatusOK, over), nil
		}), "1.0.0")
		if result.state != updateFailed {
			t.Fatalf("state = %d, want failed", result.state)
		}
	})
}

func TestUpdatesClosesBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
		want updateState
	}{
		{"success", `{"tag_name":"v1.0.0"}`, http.StatusOK, updateEqual},
		{"non-200", "server error", http.StatusInternalServerError, updateFailed},
		{"malformed", `{"tag_name":`, http.StatusOK, updateFailed},
		{"oversized", strings.Repeat(" ", maxUpdateBodyBytes+1), http.StatusOK, updateFailed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracked := &updateTrackingBody{Reader: strings.NewReader(tc.body)}
			resp := updateTestResponse(tc.code, "")
			resp.Body = tracked
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return resp, nil
			}), "1.0.0")
			if result.state != tc.want {
				t.Fatalf("state = %d, want %d", result.state, tc.want)
			}
			if !tracked.closed {
				t.Fatal("response body was not closed")
			}
		})
	}
}

func TestUpdatesTransportFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"timeout", context.DeadlineExceeded},
		{"network", errors.New("dial failed")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return nil, tc.err
			}), "1.0.0")
			if result.state != updateFailed {
				t.Fatalf("state = %d, want failed", result.state)
			}
			if result.current != "" || result.latest != "" {
				t.Fatalf("versions = %q -> %q, want empty", result.current, result.latest)
			}
		})
	}
}

func TestUpdatesRejectsRedirect(t *testing.T) {
	calls := 0
	tracked := &updateTrackingBody{Reader: strings.NewReader("redirect body")}
	client := newUpdateClient()
	client.Transport = updateTransportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		resp := updateTestResponse(http.StatusFound, "")
		resp.Header.Set("Location", "https://evil.example/steal")
		resp.Body = tracked
		return resp, nil
	})
	result := checkLatestRelease(client, "1.0.0")
	if result.state != updateFailed {
		t.Fatalf("state = %d, want failed", result.state)
	}
	if calls != 1 {
		t.Fatalf("requests = %d, want 1 (redirect must not be followed)", calls)
	}
	if !tracked.closed {
		t.Fatal("redirect response body was not closed")
	}
}

func TestUpdatesPrintNewer(t *testing.T) {
	var out bytes.Buffer
	printUpdateResult(&out, updateResult{state: updateNewer, current: "0.9.0", latest: "0.10.0"})
	text := out.String()
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	hasLine := func(want string) bool {
		for _, line := range lines {
			if line == want {
				return true
			}
		}
		return false
	}

	if len(lines) == 0 || lines[0] != "Update available: 0.9.0 -> 0.10.0" {
		t.Fatalf("first line = %q", lines[0])
	}
	if !hasLine("For Homebrew installations — copy and run manually") {
		t.Errorf("missing exact Homebrew-only label:\n%s", text)
	}
	for _, command := range []string{updateBrewCommand, updateVerifyCommand, updatePreviewCommand, updateApplyCommand} {
		if !hasLine(command) {
			t.Errorf("missing exact unstyled command line %q:\n%s", command, text)
		}
	}
	for _, want := range []string{
		"Homebrew",
		"metadata",
		"dependencies",
		"lag",
		updateReadmeURL,
		updateReleasesURL,
		"does not update installed skill files",
		updatePreviewCommand,
		updateApplyCommand,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output missing %q:\n%s", want, text)
		}
	}
	for _, line := range lines {
		if strings.Contains(line, updatePreviewCommand) && strings.Contains(line, updateApplyCommand) {
			t.Errorf("preview and apply must stay separate: %q", line)
		}
	}
	for _, forbidden := range []string{"[Copy]", "clipboard", "curl ", "| sh", "sudo ", "successfully", "self-update"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Errorf("output contains forbidden text %q:\n%s", forbidden, text)
		}
	}
	if strings.ContainsRune(text, '\x1b') {
		t.Errorf("output contains ESC:\n%q", text)
	}
	for _, r := range text {
		if r < 0x20 && r != '\n' {
			t.Errorf("output contains control rune %q", r)
		}
	}
}

func TestUpdatesPrintStates(t *testing.T) {
	tests := []struct {
		name      string
		result    updateResult
		want      []string
		forbidden []string
	}{
		{
			name:      "equal",
			result:    updateResult{state: updateEqual},
			want:      []string{"You are up to date."},
			forbidden: []string{"Update available", updateReleasesURL},
		},
		{
			name:      "ahead",
			result:    updateResult{state: updateAhead},
			want:      []string{"Installed version is newer than the latest published release."},
			forbidden: []string{"Update available", updateReleasesURL},
		},
		{
			name:      "unavailable",
			result:    updateResult{state: updateUnavailable, latest: "2.0.0"},
			want:      []string{"unavailable", "GitHub release metadata received.", "Latest published release: 2.0.0", updateReleasesURL},
			forbidden: []string{"Update available", updateBrewCommand},
		},
		{
			name:      "failed",
			result:    updateResult{state: updateFailed},
			want:      []string{"Could not check for updates.", updateReleasesURL},
			forbidden: []string{"Update available", updateBrewCommand, "\x1b"},
		},
		{
			name:      "zero value",
			result:    updateResult{},
			want:      []string{"Could not check for updates.", updateReleasesURL},
			forbidden: []string{"Update available", updateBrewCommand},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			printUpdateResult(&out, tc.result)
			text := out.String()
			lower := strings.ToLower(text)
			for _, want := range tc.want {
				needle := strings.ToLower(want)
				if !strings.Contains(lower, needle) {
					t.Errorf("output missing %q:\n%s", want, text)
				}
			}
			for _, forbidden := range tc.forbidden {
				needle := strings.ToLower(forbidden)
				if strings.Contains(lower, needle) {
					t.Errorf("output contains %q:\n%s", forbidden, text)
				}
			}
			if !strings.HasSuffix(text, "\n") {
				t.Errorf("output does not end with a newline: %q", text)
			}
		})
	}
}

// TestUpdatesSharedCheckerEquivalence locks the CLI wrapper to the shared
// appops checker. Both paths must classify the same response identically and
// render identical bytes, including strict version rejection and every
// failure class.
func TestUpdatesSharedCheckerEquivalence(t *testing.T) {
	hugeNines := strings.Repeat("9", 100)
	hugeOneAndZeros := "1" + strings.Repeat("0", 99)

	tests := []struct {
		name    string
		status  int
		body    string
		current string
	}{
		{"newer", http.StatusOK, `{"tag_name":"v0.10.0"}`, "0.9.0"},
		{"equal with optional v", http.StatusOK, `{"tag_name":"1.2.3"}`, "v1.2.3"},
		{"ahead", http.StatusOK, `{"tag_name":"v1.99.99"}`, "2.0.0"},
		{"huge numeric", http.StatusOK, `{"tag_name":"v1.` + hugeNines + `.0"}`, "1." + hugeOneAndZeros + ".0"},
		{"dev unavailable", http.StatusOK, `{"tag_name":"v2.0.0"}`, "dev"},
		{"empty current", http.StatusOK, `{"tag_name":"v2.0.0"}`, ""},
		{"leading zero tag", http.StatusOK, `{"tag_name":"v01.0.0"}`, "1.0.0"},
		{"uppercase v tag", http.StatusOK, `{"tag_name":"V1.0.0"}`, "1.0.0"},
		{"prerelease tag", http.StatusOK, `{"tag_name":"v1.0.0-rc.1"}`, "1.0.0"},
		{"draft release", http.StatusOK, `{"tag_name":"v1.0.0","draft":true}`, "1.0.0"},
		{"malformed json", http.StatusOK, `{"tag_name":`, "1.0.0"},
		{"json array", http.StatusOK, `[]`, "1.0.0"},
		{"rate limited", http.StatusTooManyRequests, `{"tag_name":"v9.9.9"}`, "1.0.0"},
		{"not found", http.StatusNotFound, `{"tag_name":"v9.9.9"}`, "1.0.0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cliTransport := updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(tc.status, tc.body), nil
			})
			sharedTransport := updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(tc.status, tc.body), nil
			})

			cliResult := checkLatestRelease(&http.Client{Transport: cliTransport, Timeout: 5 * time.Second}, tc.current)
			shared := appops.CheckLatestRelease(context.Background(), &http.Client{Transport: sharedTransport, Timeout: 5 * time.Second}, tc.current)

			if want := updateStateFor(shared.State); cliResult.state != want {
				t.Fatalf("state = %d, want %d from shared %d", cliResult.state, want, shared.State)
			}
			if cliResult.current != shared.Current || cliResult.latest != shared.Latest {
				t.Fatalf("versions = %q -> %q, want %q -> %q", cliResult.current, cliResult.latest, shared.Current, shared.Latest)
			}

			var cliOut, sharedOut bytes.Buffer
			printUpdateResult(&cliOut, cliResult)
			printUpdateResult(&sharedOut, updateResult{
				state:   updateStateFor(shared.State),
				current: shared.Current,
				latest:  shared.Latest,
			})
			if cliOut.String() != sharedOut.String() {
				t.Fatalf("rendered bytes differ:\nCLI:\n%s\nshared:\n%s", cliOut.String(), sharedOut.String())
			}
		})
	}
}

// TestUpdatesStateMapping proves every shared classification has an explicit
// CLI equivalent and that an unknown value can never claim an update.
func TestUpdatesStateMapping(t *testing.T) {
	tests := []struct {
		name string
		in   appops.UpdateState
		want updateState
	}{
		{"failed", appops.UpdateFailed, updateFailed},
		{"unavailable", appops.UpdateUnavailable, updateUnavailable},
		{"equal", appops.UpdateEqual, updateEqual},
		{"ahead", appops.UpdateAhead, updateAhead},
		{"newer", appops.UpdateNewer, updateNewer},
		{"unknown", appops.UpdateState(200), updateFailed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := updateStateFor(tc.in); got != tc.want {
				t.Fatalf("updateStateFor(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestUpdatesSharedRequestParity checks the CLI still sends exactly the
// shared fixed request: same method, URL, headers, empty body and a live
// background context.
func TestUpdatesSharedRequestParity(t *testing.T) {
	capture := func(seen **http.Request, liveErr *error) *http.Client {
		return &http.Client{Transport: updateTransportFunc(func(req *http.Request) (*http.Response, error) {
			*seen = req
			*liveErr = req.Context().Err()
			return updateTestResponse(http.StatusOK, `{"tag_name":"v1.2.3"}`), nil
		})}
	}

	var cliReq, sharedReq *http.Request
	var cliErr, sharedErr error
	cliResult := checkLatestRelease(capture(&cliReq, &cliErr), "1.2.3")
	sharedResult := appops.CheckLatestRelease(context.Background(), capture(&sharedReq, &sharedErr), "1.2.3")

	if cliReq == nil || sharedReq == nil {
		t.Fatal("missing captured request")
	}
	if cliReq.Method != sharedReq.Method {
		t.Errorf("method = %q, want %q", cliReq.Method, sharedReq.Method)
	}
	if cliReq.URL.String() != sharedReq.URL.String() {
		t.Errorf("URL = %q, want %q", cliReq.URL.String(), sharedReq.URL.String())
	}
	if cliReq.URL.String() != updateReleaseAPIURL {
		t.Errorf("URL = %q, want the fixed CLI constant", cliReq.URL.String())
	}
	for _, header := range []string{"User-Agent", "Accept"} {
		if got, want := cliReq.Header.Get(header), sharedReq.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if cliReq.Body != nil || sharedReq.Body != nil {
		t.Error("requests must not carry a body")
	}
	if cliErr != nil || sharedErr != nil {
		t.Errorf("in-flight contexts = %v / %v, want live", cliErr, sharedErr)
	}
	if cliResult != (updateResult{state: updateEqual, current: "1.2.3", latest: "1.2.3"}) {
		t.Errorf("CLI result = %+v", cliResult)
	}
	if sharedResult.State != appops.UpdateEqual {
		t.Errorf("shared state = %d, want equal", sharedResult.State)
	}
}

// TestUpdatesSharedPolicyConstants pins the request policy values the CLI
// exposes. The constants are aliases of the shared ones, so this locks the
// combined value: a careless edit to either place fails here.
func TestUpdatesSharedPolicyConstants(t *testing.T) {
	const (
		wantEndpoint = "https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest"
		wantAgent    = "agentic-sdd-update-check"
		wantAccept   = "application/vnd.github+json"
		wantCap      = 64 << 10
	)
	if updateReleaseAPIURL != wantEndpoint {
		t.Errorf("endpoint = %q, want %q", updateReleaseAPIURL, wantEndpoint)
	}
	if updateUserAgent != wantAgent {
		t.Errorf("User-Agent = %q, want %q", updateUserAgent, wantAgent)
	}
	if updateRequestAccept != wantAccept {
		t.Errorf("Accept = %q, want %q", updateRequestAccept, wantAccept)
	}
	if updateRequestTimeout != 5*time.Second {
		t.Errorf("timeout = %s, want 5s", updateRequestTimeout)
	}
	if maxUpdateBodyBytes != wantCap {
		t.Errorf("body cap = %d, want %d", maxUpdateBodyBytes, wantCap)
	}
}
