//go:build !linux && !darwin

package harnesscheck

import (
	"context"
	"fmt"
	"io"
)

const processSupported = false

func process(context.Context, []string, []string, string, io.Writer) (string, *int, string, bool) {
	return "blocked", nil, "owned process-group cancellation unsupported on this host", false
}

func executable(string, string, []string) (string, error) {
	return "", fmt.Errorf("unsupported execution host")
}
