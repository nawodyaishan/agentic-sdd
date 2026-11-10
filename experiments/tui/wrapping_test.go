package tui

import (
	"reflect"
	"testing"
)

func TestWrapPathPreservesGraphemeClusters(t *testing.T) {
	got := wrapPath("a👩‍💻b", 3)
	want := []string{"a👩‍💻", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrapPath split or overflowed grapheme: got %q want %q", got, want)
	}
}
