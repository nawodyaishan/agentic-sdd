package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// The update check is deliberately fixed and read-only. It has no endpoint
// override, authentication, environment lookup, disk cache or shell action.
const (
	updateReleaseAPIURL  = "https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest"
	updateReleasesURL    = "https://github.com/nawodyaishan/agentic-sdd/releases/latest"
	updateReadmeURL      = "https://github.com/nawodyaishan/agentic-sdd#readme"
	updateUserAgent      = "agentic-sdd-update-check"
	updateRequestAccept  = "application/vnd.github+json"
	updateRequestTimeout = 5 * time.Second
	maxUpdateBodyBytes   = 64 << 10 // 64 KiB; one extra byte detects overflow
	updateBrewCommand    = "brew update && brew upgrade nawodyaishan/tap/agentic-sdd"
	updateVerifyCommand  = "agentic-sdd version"
	updatePreviewCommand = "agentic-sdd preview"
	updateApplyCommand   = "agentic-sdd apply"
)

// updateState classifies one bounded update check. The zero value is a
// failure, so callers never mistake an uninitialised result for a comparison.
type updateState uint8

const (
	updateFailed updateState = iota
	updateUnavailable
	updateEqual
	updateAhead
	updateNewer
)

// updateResult carries only validated, normalized versions. current and
// latest stay empty unless the corresponding value passed stable parsing, and
// failed results never carry either. A newer result always has both set.
type updateResult struct {
	state   updateState
	current string
	latest  string
}

// updateVersion is a stable major.minor.patch version whose components keep
// their decimal text, so comparison works for arbitrarily long numbers.
type updateVersion struct {
	major string
	minor string
	patch string
}

func (v updateVersion) String() string {
	return v.major + "." + v.minor + "." + v.patch
}

// newUpdateClient builds the production client. A total timeout bounds DNS,
// connection, redirects, headers and body reads; redirects are never followed.
func newUpdateClient() *http.Client {
	return &http.Client{
		Timeout: updateRequestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// checkLatestRelease performs one explicit read-only request. All transport,
// status, body and release metadata problems become updateFailed. A valid
// upstream release is validated before the installed version is considered,
// so an invalid current version yields updateUnavailable rather than a
// comparison or an upgrade claim.
func checkLatestRelease(client *http.Client, current string) updateResult {
	if client == nil {
		return updateResult{state: updateFailed}
	}

	req, err := http.NewRequest(http.MethodGet, updateReleaseAPIURL, nil)
	if err != nil {
		return updateResult{state: updateFailed}
	}
	req.Header.Set("User-Agent", updateUserAgent)
	req.Header.Set("Accept", updateRequestAccept)

	resp, err := client.Do(req)
	if err != nil {
		return updateResult{state: updateFailed}
	}
	if resp == nil || resp.Body == nil {
		return updateResult{state: updateFailed}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return updateResult{state: updateFailed}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxUpdateBodyBytes+1))
	if err != nil || len(body) > maxUpdateBodyBytes {
		return updateResult{state: updateFailed}
	}

	var release struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return updateResult{state: updateFailed}
	}
	latest, ok := parseUpdateVersion(release.TagName)
	if !ok || release.Draft || release.Prerelease {
		return updateResult{state: updateFailed}
	}

	installed, ok := parseUpdateVersion(current)
	if !ok {
		return updateResult{state: updateUnavailable, latest: latest.String()}
	}

	state := updateEqual
	switch compareUpdateVersions(installed, latest) {
	case -1:
		state = updateNewer
	case 1:
		state = updateAhead
	}
	return updateResult{state: state, current: installed.String(), latest: latest.String()}
}

// parseUpdateVersion accepts an optional lowercase "v" and exactly three
// decimal components. Leading-zero multi-digit components, suffixes,
// whitespace, controls and non-ASCII digits are rejected.
func parseUpdateVersion(s string) (updateVersion, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return updateVersion{}, false
	}
	for _, part := range parts {
		if !validUpdateComponent(part) {
			return updateVersion{}, false
		}
	}
	return updateVersion{major: parts[0], minor: parts[1], patch: parts[2]}, true
}

func validUpdateComponent(part string) bool {
	if part == "" {
		return false
	}
	if len(part) > 1 && part[0] == '0' {
		return false
	}
	for i := 0; i < len(part); i++ {
		if part[i] < '0' || part[i] > '9' {
			return false
		}
	}
	return true
}

// compareUpdateVersions returns -1, 0 or 1. Numeric components are compared
// by digit count first and then lexically, so no machine integer can overflow.
func compareUpdateVersions(a, b updateVersion) int {
	if cmp := compareUpdateComponent(a.major, b.major); cmp != 0 {
		return cmp
	}
	if cmp := compareUpdateComponent(a.minor, b.minor); cmp != 0 {
		return cmp
	}
	return compareUpdateComponent(a.patch, b.patch)
}

func compareUpdateComponent(a, b string) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// printUpdateResult renders one plain-text result. Only fixed text and
// normalized numeric versions reach the terminal; no raw response body,
// upstream URL or control sequence is echoed. Script lines are intentionally
// unstyled and unindented so they stay copyable.
func printUpdateResult(out io.Writer, result updateResult) {
	if out == nil {
		return
	}
	switch result.state {
	case updateNewer:
		fmt.Fprintf(out, "Update available: %s -> %s\n", result.current, result.latest)
		fmt.Fprintln(out, updateReleasesURL)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "For Homebrew installations — copy and run manually")
		fmt.Fprintln(out, updateBrewCommand)
		fmt.Fprintln(out, updateVerifyCommand)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Homebrew refreshes package metadata, may update dependencies, and")
		fmt.Fprintln(out, "its formula can lag a GitHub release. Run the version command")
		fmt.Fprintln(out, "afterwards to verify what is installed.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "For source or manual installations, use the README or release instructions:")
		fmt.Fprintln(out, updateReadmeURL)
		fmt.Fprintln(out, updateReleasesURL)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Upgrading the CLI does not update installed skill files.")
		fmt.Fprintln(out, "Run this next and review it:")
		fmt.Fprintln(out, updatePreviewCommand)
		fmt.Fprintln(out, "Apply only after that separate review:")
		fmt.Fprintln(out, updateApplyCommand)
		fmt.Fprintln(out)
	case updateEqual:
		fmt.Fprintln(out, "You are up to date.")
		fmt.Fprintln(out)
	case updateAhead:
		fmt.Fprintln(out, "Installed version is newer than the latest published release.")
		fmt.Fprintln(out)
	case updateUnavailable:
		fmt.Fprintln(out, "GitHub release metadata received.")
		fmt.Fprintln(out, "Development or non-release build; version comparison unavailable.")
		if latest, ok := parseUpdateVersion(result.latest); ok {
			fmt.Fprintln(out, "Latest published release: "+latest.String())
		}
		fmt.Fprintln(out, "See the released versions at:")
		fmt.Fprintln(out, updateReleasesURL)
	default: // updateFailed, including the zero value
		fmt.Fprintln(out, "Could not check for updates.")
		fmt.Fprintln(out, "No release information confirmed. Check your connection or try later.")
		fmt.Fprintln(out, "See the released versions at:")
		fmt.Fprintln(out, updateReleasesURL)
	}
}
