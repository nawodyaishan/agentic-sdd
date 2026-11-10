package appops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests exercise the shared release checker only through its exported
// API. Every request is answered by an injected transport: no test resolves or
// contacts GitHub, and none reads a real home directory.

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

// updateBlockingBody blocks every read until the request context is cancelled.
// It proves cancellation reaches an in-flight body read, not only the dial.
type updateBlockingBody struct {
	ctx     context.Context
	started chan struct{}
	once    sync.Once
	closed  bool
}

func (b *updateBlockingBody) Read([]byte) (int, error) {
	if b.started != nil {
		b.once.Do(func() { close(b.started) })
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *updateBlockingBody) Close() error {
	b.closed = true
	return nil
}

// updateTestCheck runs the checker with one injected transport and a bounded
// background context, mirroring the CLI's client shape.
func updateTestCheck(t *testing.T, transport http.RoundTripper, current string) UpdateResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), UpdateRequestTimeout)
	defer cancel()
	return CheckLatestRelease(ctx, &http.Client{Transport: transport, Timeout: UpdateRequestTimeout}, current)
}

func TestUpdateStateZeroValueIsFailure(t *testing.T) {
	var result UpdateResult
	if result.State != UpdateFailed {
		t.Fatalf("zero State = %d, want UpdateFailed", result.State)
	}
	if result.Current != "" || result.Latest != "" {
		t.Fatalf("zero result carries versions: %+v", result)
	}
}

func TestCheckLatestReleaseRequestContract(t *testing.T) {
	var got *http.Request
	var liveErr error
	calls := 0
	result := updateTestCheck(t, updateTransportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		got = req
		// The client derives and later cancels its own request context, so
		// liveness is observed while the request is in flight.
		liveErr = req.Context().Err()
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
	if got.URL.String() != UpdateReleaseAPIURL {
		t.Errorf("URL = %q, want fixed %q", got.URL.String(), UpdateReleaseAPIURL)
	}
	if got.URL.Scheme != "https" || got.URL.Host != "api.github.com" {
		t.Errorf("URL = %q, want https://api.github.com", got.URL.String())
	}
	if got.URL.Path != "/repos/nawodyaishan/agentic-sdd/releases/latest" {
		t.Errorf("path = %q", got.URL.Path)
	}
	if got.URL.RawQuery != "" || got.URL.Fragment != "" || got.URL.User != nil {
		t.Errorf("URL must stay fixed: %q", got.URL.String())
	}
	if ua := got.Header.Get("User-Agent"); ua != UpdateUserAgent {
		t.Errorf("User-Agent = %q, want %q", ua, UpdateUserAgent)
	}
	if accept := got.Header.Get("Accept"); accept != UpdateRequestAccept {
		t.Errorf("Accept = %q, want %q", accept, UpdateRequestAccept)
	}
	for _, header := range []string{"Authorization", "Cookie", "X-Api-Key", "If-None-Match"} {
		if value := got.Header.Get(header); value != "" {
			t.Errorf("%s = %q, want empty", header, value)
		}
	}
	if got.Body != nil {
		t.Errorf("request body = %v, want none", got.Body)
	}
	if got.Context() == nil {
		t.Fatal("request carried no context")
	}
	if liveErr != nil {
		t.Errorf("in-flight request context = %v, want a live caller context", liveErr)
	}
	if result.State != UpdateEqual || result.Current != "1.2.3" || result.Latest != "1.2.3" {
		t.Errorf("result = %+v, want equal 1.2.3", result)
	}
}

func TestCheckLatestReleaseStates(t *testing.T) {
	tests := []struct {
		name    string
		current string
		tag     string
		want    UpdateState
	}{
		{"newer", "0.9.0", "v0.10.0", UpdateNewer},
		{"equal with optional v", "v1.2.3", "1.2.3", UpdateEqual},
		{"ahead", "2.0.0", "v1.99.99", UpdateAhead},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(http.StatusOK, `{"tag_name":"`+tc.tag+`"}`), nil
			}), tc.current)
			if result.State != tc.want {
				t.Fatalf("State = %d, want %d", result.State, tc.want)
			}
			wantCurrent := strings.TrimPrefix(tc.current, "v")
			wantLatest := strings.TrimPrefix(tc.tag, "v")
			if result.Current != wantCurrent || result.Latest != wantLatest {
				t.Errorf("versions = %q -> %q, want %q -> %q", result.Current, result.Latest, wantCurrent, wantLatest)
			}
		})
	}
}

func TestCheckLatestReleaseComparator(t *testing.T) {
	hugeNines := strings.Repeat("9", 100)
	hugeOneAndZeros := "1" + strings.Repeat("0", 99)
	longNines := strings.Repeat("9", 300)

	tests := []struct {
		name    string
		current string
		tag     string
		want    UpdateState
	}{
		{"numeric not lexical", "0.9.0", "v0.10.0", UpdateNewer},
		{"equal", "v1.2.3", "1.2.3", UpdateEqual},
		{"ahead", "2.0.0", "v1.99.99", UpdateAhead},
		{"major newer", "1.9.9", "v2.0.0", UpdateNewer},
		{"patch numeric", "1.0.9", "v1.0.10", UpdateNewer},
		{"zero components", "0.0.0", "v0.0.0", UpdateEqual},
		{"huge current ahead", "1." + hugeNines + ".0", "v1." + hugeOneAndZeros + ".0", UpdateAhead},
		{"huge latest newer", "1." + hugeOneAndZeros + ".0", "v1." + hugeNines + ".0", UpdateNewer},
		{"huge equal", "1." + longNines + ".0", "v1." + longNines + ".0", UpdateEqual},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(http.StatusOK, `{"tag_name":"`+tc.tag+`"}`), nil
			}), tc.current)
			if result.State != tc.want {
				t.Fatalf("State = %d, want %d", result.State, tc.want)
			}
			if result.Current != strings.TrimPrefix(tc.current, "v") {
				t.Errorf("Current = %q", result.Current)
			}
			if result.Latest != strings.TrimPrefix(tc.tag, "v") {
				t.Errorf("Latest = %q", result.Latest)
			}
		})
	}
}

func TestCheckLatestReleaseInvalidCurrentUnavailable(t *testing.T) {
	currents := []string{
		"dev",
		"none",
		"unknown",
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
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(http.StatusOK, `{"tag_name":"v2.0.0"}`), nil
			}), current)
			if result.State != UpdateUnavailable {
				t.Fatalf("State = %d, want unavailable", result.State)
			}
			if result.Current != "" {
				t.Errorf("Current = %q, want empty for invalid input", result.Current)
			}
			if result.Latest != "2.0.0" {
				t.Errorf("Latest = %q, want validated 2.0.0", result.Latest)
			}
		})
	}
}

func TestCheckLatestReleaseFailurePrecedesInvalidCurrent(t *testing.T) {
	result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
		return updateTestResponse(http.StatusOK, `{"tag_name":`), nil
	}), "dev")
	if result.State != UpdateFailed {
		t.Fatalf("State = %d, want failed", result.State)
	}
	if result.Current != "" || result.Latest != "" {
		t.Fatalf("versions = %q -> %q, want empty on failure", result.Current, result.Latest)
	}
}

func TestCheckLatestReleaseUntrustedMetadata(t *testing.T) {
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
		{"json null", `null`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(http.StatusOK, tc.body), nil
			}), "1.0.0")
			if result.State != UpdateFailed {
				t.Fatalf("State = %d, want failed", result.State)
			}
			if result.Current != "" || result.Latest != "" {
				t.Fatalf("versions = %q -> %q, want empty on failure", result.Current, result.Latest)
			}
		})
	}
}

func TestCheckLatestReleaseIgnoresUntrustedReleaseFields(t *testing.T) {
	body := `{"tag_name":"v1.0.0","body":"\u001b[31m evil https://evil.example/secret","assets":[{"name":"evil"}]}`
	result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
		return updateTestResponse(http.StatusOK, body), nil
	}), "0.9.0")
	if result.State != UpdateNewer {
		t.Fatalf("State = %d, want newer", result.State)
	}
	if result.Current != "0.9.0" || result.Latest != "1.0.0" {
		t.Fatalf("versions = %q -> %q", result.Current, result.Latest)
	}
}

func TestCheckLatestReleaseNon200(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return updateTestResponse(status, `{"tag_name":"v1.0.0"}`), nil
			}), "1.0.0")
			if result.State != UpdateFailed {
				t.Fatalf("State = %d, want failed", result.State)
			}
			if result.Current != "" || result.Latest != "" {
				t.Fatalf("versions = %q -> %q, want empty", result.Current, result.Latest)
			}
		})
	}
}

func TestCheckLatestReleaseBodyLimit(t *testing.T) {
	base := `{"tag_name":"v1.0.0"}`
	if len(base) >= MaxUpdateBodyBytes {
		t.Fatal("test response base exceeds the body cap")
	}
	exact := base + strings.Repeat(" ", MaxUpdateBodyBytes-len(base))
	over := base + strings.Repeat(" ", MaxUpdateBodyBytes-len(base)+1)

	t.Run("exact limit accepted", func(t *testing.T) {
		result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
			return updateTestResponse(http.StatusOK, exact), nil
		}), "1.0.0")
		if result.State != UpdateEqual {
			t.Fatalf("State = %d, want equal", result.State)
		}
	})
	t.Run("one byte over rejected", func(t *testing.T) {
		result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
			return updateTestResponse(http.StatusOK, over), nil
		}), "1.0.0")
		if result.State != UpdateFailed {
			t.Fatalf("State = %d, want failed", result.State)
		}
	})
	t.Run("unbounded stream rejected without buffering it all", func(t *testing.T) {
		result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
			return updateTestResponse(http.StatusOK, `{"tag_name":"v1.0.0","padding":"`+
				strings.Repeat("a", 4*MaxUpdateBodyBytes)+`"}`), nil
		}), "1.0.0")
		if result.State != UpdateFailed {
			t.Fatalf("State = %d, want failed", result.State)
		}
	})
}

func TestCheckLatestReleaseClosesBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
		want UpdateState
	}{
		{"success", `{"tag_name":"v1.0.0"}`, http.StatusOK, UpdateEqual},
		{"non-200", "server error", http.StatusInternalServerError, UpdateFailed},
		{"malformed", `{"tag_name":`, http.StatusOK, UpdateFailed},
		{"oversized", strings.Repeat(" ", MaxUpdateBodyBytes+1), http.StatusOK, UpdateFailed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracked := &updateTrackingBody{Reader: strings.NewReader(tc.body)}
			resp := updateTestResponse(tc.code, "")
			resp.Body = tracked
			result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
				return resp, nil
			}), "1.0.0")
			if result.State != tc.want {
				t.Fatalf("State = %d, want %d", result.State, tc.want)
			}
			if !tracked.closed {
				t.Fatal("response body was not closed")
			}
		})
	}
}

func TestCheckLatestReleaseTransportFailure(t *testing.T) {
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
			if result.State != UpdateFailed {
				t.Fatalf("State = %d, want failed", result.State)
			}
			if result.Current != "" || result.Latest != "" {
				t.Fatalf("versions = %q -> %q, want empty", result.Current, result.Latest)
			}
		})
	}
}

func TestCheckLatestReleaseNilResponse(t *testing.T) {
	result := updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	}), "1.0.0")
	if result.State != UpdateFailed {
		t.Fatalf("State = %d, want failed", result.State)
	}

	result = updateTestCheck(t, updateTransportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	}), "1.0.0")
	if result.State != UpdateFailed {
		t.Fatalf("State with nil body = %d, want failed", result.State)
	}
}

func TestCheckLatestReleaseNilInputs(t *testing.T) {
	result := CheckLatestRelease(context.Background(), nil, "1.0.0")
	if result.State != UpdateFailed || result.Current != "" || result.Latest != "" {
		t.Fatalf("nil client result = %+v, want failed", result)
	}

	calls := 0
	client := &http.Client{Transport: updateTransportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return updateTestResponse(http.StatusOK, `{"tag_name":"v1.0.0"}`), nil
	})}
	//nolint:staticcheck // A nil context must degrade to failure, not panic.
	result = CheckLatestRelease(nil, client, "1.0.0")
	if result.State != UpdateFailed {
		t.Fatalf("nil context State = %d, want failed", result.State)
	}
	if calls != 0 {
		t.Fatalf("requests = %d with a nil context, want 0", calls)
	}
}

func TestCheckLatestReleaseRefusesRedirect(t *testing.T) {
	permissive := func(*http.Request, []*http.Request) error { return nil }

	t.Run("refusing client", func(t *testing.T) {
		calls := 0
		tracked := &updateTrackingBody{Reader: strings.NewReader("redirect body")}
		client := &http.Client{
			Transport: updateTransportFunc(func(*http.Request) (*http.Response, error) {
				calls++
				resp := updateTestResponse(http.StatusFound, "")
				resp.Header.Set("Location", "https://evil.example/steal")
				resp.Body = tracked
				return resp, nil
			}),
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
		result := CheckLatestRelease(context.Background(), client, "1.0.0")
		if result.State != UpdateFailed {
			t.Fatalf("State = %d, want failed", result.State)
		}
		if calls != 1 {
			t.Fatalf("requests = %d, want 1 (redirect must not be followed)", calls)
		}
		if !tracked.closed {
			t.Fatal("redirect response body was not closed")
		}
	})

	t.Run("caller cannot widen the policy", func(t *testing.T) {
		calls := 0
		client := &http.Client{
			Transport: updateTransportFunc(func(*http.Request) (*http.Response, error) {
				calls++
				resp := updateTestResponse(http.StatusFound, "")
				resp.Header.Set("Location", "https://evil.example/steal")
				return resp, nil
			}),
			CheckRedirect: permissive,
		}
		result := CheckLatestRelease(context.Background(), client, "1.0.0")
		if result.State != UpdateFailed {
			t.Fatalf("State = %d, want failed", result.State)
		}
		if calls != 1 {
			t.Fatalf("requests = %d, want 1 even with a permissive caller policy", calls)
		}
		if client.CheckRedirect == nil {
			t.Fatal("caller client was mutated")
		}
		if err := client.CheckRedirect(nil, nil); err != nil {
			t.Fatalf("caller redirect policy was mutated: %v", err)
		}
	})
}

// TestCheckLatestReleaseContextCancellation proves cancellation reaches the
// transport and an in-flight body read, rather than merely being observed by
// a caller that discards a late result.
func TestCheckLatestReleaseContextCancellation(t *testing.T) {
	t.Run("transport observes ctx.Done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		entered := make(chan struct{})
		observed := make(chan error, 1)
		client := &http.Client{Transport: updateTransportFunc(func(req *http.Request) (*http.Response, error) {
			close(entered)
			<-req.Context().Done()
			observed <- req.Context().Err()
			return nil, req.Context().Err()
		})}

		results := make(chan UpdateResult, 1)
		go func() {
			results <- CheckLatestRelease(ctx, client, "1.0.0")
		}()

		<-entered
		cancel()

		select {
		case err := <-observed:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("transport saw %v, want context.Canceled", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("transport never observed cancellation")
		}

		select {
		case result := <-results:
			if result.State != UpdateFailed {
				t.Fatalf("State = %d, want failed after cancellation", result.State)
			}
			if result.Current != "" || result.Latest != "" {
				t.Fatalf("cancelled result carries versions: %+v", result)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("check did not return after cancellation")
		}
	})

	t.Run("body read observes ctx.Done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		body := &updateBlockingBody{ctx: ctx, started: make(chan struct{})}
		entered := make(chan struct{})
		client := &http.Client{Transport: updateTransportFunc(func(*http.Request) (*http.Response, error) {
			close(entered)
			resp := updateTestResponse(http.StatusOK, "")
			resp.Body = body
			return resp, nil
		})}

		results := make(chan UpdateResult, 1)
		go func() {
			results <- CheckLatestRelease(ctx, client, "1.0.0")
		}()

		<-entered
		// Wait for the body read to block, then cancel it.
		<-body.started
		cancel()

		select {
		case result := <-results:
			if result.State != UpdateFailed {
				t.Fatalf("State = %d, want failed after body cancellation", result.State)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("blocked body read was not cancelled")
		}
		if !body.closed {
			t.Fatal("response body was not closed after cancellation")
		}
	})

	t.Run("already cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		client := &http.Client{Transport: updateTransportFunc(func(req *http.Request) (*http.Response, error) {
			if err := req.Context().Err(); err != nil {
				return nil, err
			}
			t.Error("transport ran with a live context")
			return updateTestResponse(http.StatusOK, `{"tag_name":"v9.9.9"}`), nil
		})}
		result := CheckLatestRelease(ctx, client, "1.0.0")
		if result.State != UpdateFailed {
			t.Fatalf("State = %d, want failed", result.State)
		}
	})
}
