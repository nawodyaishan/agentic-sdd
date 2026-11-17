package appops

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// The release check is deliberately fixed and read-only. It has one endpoint,
// no override, no authentication, no environment lookup, no disk cache and no
// shell action. It sends no local inventory, and it never downloads or applies
// a release. Presentation text stays at the CLI or TUI boundary.
const (
	// UpdateReleaseAPIURL is the only endpoint this package contacts.
	UpdateReleaseAPIURL = "https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest"
	// UpdateUserAgent identifies the bounded check to the fixed endpoint.
	UpdateUserAgent = "agentic-sdd-update-check"
	// UpdateRequestAccept asks for the documented GitHub JSON media type.
	UpdateRequestAccept = "application/vnd.github+json"
	// UpdateRequestTimeout is the bounded total budget a caller should give one
	// check: DNS, connection, redirects, headers and body reads. Callers may
	// shorten it, for example to bound a first-render background attempt.
	UpdateRequestTimeout = 5 * time.Second
	// MaxUpdateBodyBytes caps the response body; one extra byte is read so an
	// oversized payload is detected instead of silently truncated.
	MaxUpdateBodyBytes = 64 << 10
)

// UpdateState classifies one bounded release check. The zero value is a
// failure, so a zero or uninitialised result is never mistaken for a version
// comparison or an update claim.
type UpdateState uint8

const (
	// UpdateFailed means no release information was confirmed: transport,
	// status, body-size, metadata or context problems all land here. Callers
	// should present it as a non-fatal "could not check" state.
	UpdateFailed UpdateState = iota
	// UpdateUnavailable means a valid published release was read but the
	// installed version is not a comparable release version, such as a
	// development build. Latest may carry the validated published version;
	// Current is always empty.
	UpdateUnavailable
	// UpdateEqual means the installed version matches the published release.
	UpdateEqual
	// UpdateAhead means the installed version is newer than the release.
	UpdateAhead
	// UpdateNewer means a newer published release exists. No upgrade is
	// performed or implied.
	UpdateNewer
)

// UpdateResult carries only validated, normalized versions. Current and Latest
// stay empty unless the corresponding text passed strict release-version
// parsing, and a failed result never carries either. A newer, equal or ahead
// result always has both set.
//
// There is deliberately no error or raw response field: transport and server
// text is untrusted, and callers need a classification rather than an echo.
type UpdateResult struct {
	State   UpdateState
	Current string
	Latest  string
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

// redirectRefused is the policy forced on every check. It is a function
// value so each returned client owns an immutable copy.
func redirectRefused(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// boundedClient returns the caller's client with redirects always refused.
// A caller-supplied policy can tighten this further but can never widen it, so
// a redirected request cannot be used to move the check to another host.
// Copying http.Client is safe: it holds no locks, and the transport and jar
// are shared by pointer, as they are meant to be.
func boundedClient(client *http.Client) *http.Client {
	guarded := *client
	guarded.CheckRedirect = redirectRefused
	return &guarded
}

// CheckLatestRelease performs one explicit read-only request for the fixed
// release endpoint and classifies the result. Cancelling ctx aborts an
// in-flight request; the caller owns the timeout and any staleness generation.
//
// A nil client or nil context yields UpdateFailed rather than a panic, and a
// cancelled context yields UpdateFailed because no release information was
// confirmed. The caller distinguishes its own cancellation from a network
// failure from its own context state, not from untrusted response text.
func CheckLatestRelease(ctx context.Context, client *http.Client, current string) UpdateResult {
	if client == nil || ctx == nil {
		return UpdateResult{State: UpdateFailed}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UpdateReleaseAPIURL, nil)
	if err != nil {
		return UpdateResult{State: UpdateFailed}
	}
	req.Header.Set("User-Agent", UpdateUserAgent)
	req.Header.Set("Accept", UpdateRequestAccept)

	resp, err := boundedClient(client).Do(req)
	if err != nil {
		return UpdateResult{State: UpdateFailed}
	}
	if resp == nil || resp.Body == nil {
		return UpdateResult{State: UpdateFailed}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UpdateResult{State: UpdateFailed}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxUpdateBodyBytes+1))
	if err != nil || len(body) > MaxUpdateBodyBytes {
		return UpdateResult{State: UpdateFailed}
	}

	var release struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return UpdateResult{State: UpdateFailed}
	}
	latest, ok := parseVersion(release.TagName)
	if !ok || release.Draft || release.Prerelease {
		return UpdateResult{State: UpdateFailed}
	}

	// The published release is validated first, so an invalid installed
	// version reports unavailable rather than a fabricated comparison.
	installed, ok := parseVersion(current)
	if !ok {
		return UpdateResult{State: UpdateUnavailable, Latest: latest.String()}
	}

	state := UpdateEqual
	switch compareVersions(installed, latest) {
	case -1:
		state = UpdateNewer
	case 1:
		state = UpdateAhead
	}
	return UpdateResult{State: state, Current: installed.String(), Latest: latest.String()}
}

// parseVersion accepts an optional lowercase "v" and exactly three decimal
// components. Leading-zero multi-digit components, suffixes, whitespace,
// controls and non-ASCII digits are rejected.
func parseVersion(s string) (updateVersion, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return updateVersion{}, false
	}
	for _, part := range parts {
		if !validVersionComponent(part) {
			return updateVersion{}, false
		}
	}
	return updateVersion{major: parts[0], minor: parts[1], patch: parts[2]}, true
}

func validVersionComponent(part string) bool {
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

// compareVersions returns -1, 0 or 1. Numeric components are compared by
// digit count first and then lexically, so no machine integer can overflow.
func compareVersions(a, b updateVersion) int {
	if cmp := compareVersionComponent(a.major, b.major); cmp != 0 {
		return cmp
	}
	if cmp := compareVersionComponent(a.minor, b.minor); cmp != 0 {
		return cmp
	}
	return compareVersionComponent(a.patch, b.patch)
}

func compareVersionComponent(a, b string) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}
