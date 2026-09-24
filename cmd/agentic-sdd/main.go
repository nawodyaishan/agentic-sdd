package main

import (
	"flag"
	"fmt"
	"os"

	"agentic-sdd/internal/skillsync"
)

func main() {
	var repo, home string
	var apply bool
	flag.StringVar(&repo, "repo", ".", "repository containing skills-files")
	flag.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	flag.BoolVar(&apply, "apply", false, "back up and install skills; default is preview")
	flag.Parse()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			fatal(err)
		}
	}
	if err := skillsync.Sync(repo, home, apply, os.Stdout); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
