package skillsync

import (
	"sort"
	"strings"
)

// ClientSummary holds the plan counts for one client target (for example
// "claude" or "opencode-agents"). A zero field means the plan printed no
// line of that kind for the client.
type ClientSummary struct {
	Client   string
	Install  int
	Replace  int
	Restore  int
	UpToDate int
	Remove   int
}

// PlanSummary is the classified form of a preview or apply transcript
// written by SyncWithOptions or Restore.
//
// Clients is ordered deterministically: "agents", "codex", "claude", "agy",
// then "opencode-agents", then any other client name in sorted order.
// Unclassified holds every non-empty line the parser did not recognize; a
// non-empty Unclassified means the producer wording drifted.
type PlanSummary struct {
	Clients      []ClientSummary
	Install      int
	Replace      int
	Restore      int
	UpToDate     int
	Remove       int
	Unclassified []string
}

// TotalChanges returns how many skills or agent files the plan would write:
// installs, replacements, restores and removals. Up-to-date entries are not
// changes.
func (s PlanSummary) TotalChanges() int {
	return s.Install + s.Replace + s.Restore + s.Remove
}

// planVerb identifies the kind of item a plan line reports.
type planVerb int

const (
	planInstall planVerb = iota
	planReplace
	planRestore
	planUpToDate
	planRemove
)

// planItemPrefixes are the producer verb prefixes that introduce one skill
// or agent file, in the form "<prefix> <client>/<name>". They mirror the
// format strings in sync.go and restore.go; keeping them in one list makes a
// reworded producer line visible to the drift test.
var planItemPrefixes = []struct {
	prefix string
	verb   planVerb
}{
	{"replace + back up ", planReplace},
	{"remove + back up ", planRemove},
	{"up to date ", planUpToDate},
	{"install ", planInstall},
	{"restore ", planRestore},
}

// planNonItemPrefixes are the trailer and notice lines a plan transcript can
// contain. They carry no per-skill counts and are not classified.
var planNonItemPrefixes = []string{
	"Preview only.",
	"Note: ",
	"Legacy backup:",
	"All skills are up to date.",
	"Installed ",
	"Restored ",
}

// SummarizePlan classifies the lines of a preview or apply transcript into
// per-client counts and totals. It is a pure function: it performs no I/O
// and never changes what the producers print.
func SummarizePlan(transcript string) PlanSummary {
	var summary PlanSummary
	byClient := make(map[string]*ClientSummary)

	for _, raw := range strings.Split(transcript, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" {
			continue
		}
		verb, client, ok := classifyPlanLine(line)
		if !ok {
			if isPlanNonItemLine(line) {
				continue
			}
			summary.Unclassified = append(summary.Unclassified, line)
			continue
		}
		c, seen := byClient[client]
		if !seen {
			c = &ClientSummary{Client: client}
			byClient[client] = c
		}
		switch verb {
		case planInstall:
			c.Install++
			summary.Install++
		case planReplace:
			c.Replace++
			summary.Replace++
		case planRestore:
			c.Restore++
			summary.Restore++
		case planUpToDate:
			c.UpToDate++
			summary.UpToDate++
		case planRemove:
			c.Remove++
			summary.Remove++
		}
	}

	summary.Clients = orderPlanClients(byClient)
	return summary
}

// classifyPlanLine reports the verb and client of a per-item plan line. It
// requires a non-empty client and skill name after the verb prefix.
func classifyPlanLine(line string) (planVerb, string, bool) {
	for _, p := range planItemPrefixes {
		rest, found := strings.CutPrefix(line, p.prefix)
		if !found {
			continue
		}
		client, name, cut := strings.Cut(rest, "/")
		if !cut || client == "" || name == "" {
			return 0, "", false
		}
		return p.verb, client, true
	}
	return 0, "", false
}

// isPlanNonItemLine reports whether line is a known trailer or notice.
func isPlanNonItemLine(line string) bool {
	for _, prefix := range planNonItemPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// orderPlanClients returns the client summaries in the fixed order the
// clients are planned and installed in, followed by any other client name in
// sorted order.
func orderPlanClients(byClient map[string]*ClientSummary) []ClientSummary {
	if len(byClient) == 0 {
		return nil
	}
	fixed := []string{"agents", "codex", "claude", "agy", "opencode-agents"}
	clients := make([]ClientSummary, 0, len(byClient))
	placed := make(map[string]bool, len(byClient))
	for _, name := range fixed {
		if c, ok := byClient[name]; ok {
			clients = append(clients, *c)
			placed[name] = true
		}
	}
	var rest []string
	for name := range byClient {
		if !placed[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	for _, name := range rest {
		clients = append(clients, *byClient[name])
	}
	return clients
}
