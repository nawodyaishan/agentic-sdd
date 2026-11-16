package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// collectTopLevelTestNames parses the Go test files matched by patterns
// (relative to the package directory) and returns the set of top-level
// func Test… names.
func collectTopLevelTestNames(t *testing.T, patterns []string) map[string]bool {
	t.Helper()
	names := make(map[string]bool)
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil {
			t.Fatalf("glob %q: %v", pat, err)
		}
		for _, m := range matches {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, m, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", m, err)
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				if strings.HasPrefix(fn.Name.Name, "Test") {
					names[fn.Name.Name] = true
				}
			}
		}
	}
	return names
}

func splitMarkdownTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

var backtickedTestNameRe = regexp.MustCompile("`Test[A-Za-z0-9_]+`")

func TestExecutionContractTraceabilityMatrix(t *testing.T) {
	// 1. Collect top-level Test… names from the three authorized glob roots.
	testNames := collectTopLevelTestNames(t, []string{
		"*_test.go",
		"../../internal/execution/*_test.go",
		"../../internal/skillsync/*_test.go",
	})
	if len(testNames) == 0 {
		t.Fatal("no top-level Test functions collected")
	}

	// 2. Read the matrix (read-only).
	matrixPath := "../../specs/021-execution-cli-verification/test-case-matrix.md"
	data, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatalf("read matrix %s: %v", matrixPath, err)
	}
	text := string(data)

	// 3. Every exact backticked Test<name> token must name an existing top-level test.
	for _, tok := range backtickedTestNameRe.FindAllString(text, -1) {
		name := strings.Trim(tok, "`")
		if !testNames[name] {
			t.Errorf("matrix references missing top-level test %s", name)
		}
	}

	// 4. Expected clause set.
	expectedClauses := make([]string, 0, 33)
	for i := 1; i <= 12; i++ {
		expectedClauses = append(expectedClauses, fmt.Sprintf("FR-%02d", i))
	}
	for i := 1; i <= 13; i++ {
		expectedClauses = append(expectedClauses, fmt.Sprintf("AC-%02d", i))
	}
	for i := 1; i <= 5; i++ {
		expectedClauses = append(expectedClauses, fmt.Sprintf("017 AC-%02d", i))
	}
	for _, d := range []string{"03", "04", "10"} {
		expectedClauses = append(expectedClauses, "019 D-"+d)
	}
	expectedSet := make(map[string]bool, len(expectedClauses))
	for _, c := range expectedClauses {
		expectedSet[c] = true
	}

	// 5. Parse clause rows from markdown tables.
	clauseCounts := make(map[string]int)
	type clauseRow struct {
		clause   string
		evidence string
		origin   string
		status   string
	}
	var rows []clauseRow

	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitMarkdownTableRow(line)
		if len(cells) < 4 {
			continue
		}
		first := cells[0]
		if first == "" || first == "Clause" {
			continue
		}
		if strings.Trim(first, "-: ") == "" {
			continue
		}
		parts := strings.Fields(first)
		var clause string
		if len(parts) >= 2 && (parts[0] == "017" || parts[0] == "019") {
			clause = parts[0] + " " + parts[1]
		} else if len(parts) >= 1 {
			clause = parts[0]
		}
		if !expectedSet[clause] {
			continue
		}
		clauseCounts[clause]++
		rows = append(rows, clauseRow{
			clause:   clause,
			evidence: cells[1],
			origin:   cells[2],
			status:   cells[3],
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan matrix: %v", err)
	}

	// 6. Every expected clause appears exactly once as a row.
	for _, c := range expectedClauses {
		if clauseCounts[c] != 1 {
			t.Errorf("clause %s appears %d times as a row, want 1", c, clauseCounts[c])
		}
	}

	// 7. Origin/status vocabulary validation.
	allowedOrigins := map[string]bool{
		"existing-service":    true,
		"existing-cli":        true,
		"existing-cli (stub)": true,
		"new-cli":             true,
		"static":              true,
		"static-check":        true,
	}
	allowedStatuses := map[string]bool{
		"covered":        true,
		"covered-static": true,
		"not_run":        true,
	}
	for _, r := range rows {
		for _, tok := range strings.Split(r.origin, "/") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			if !allowedOrigins[tok] {
				t.Errorf("row %s has invalid origin token %q", r.clause, tok)
			}
		}
		if !allowedStatuses[r.status] {
			t.Errorf("row %s has invalid runtime status %q", r.clause, r.status)
		}
	}

	// 8. G13 must not be a dependency of a covered row.
	for _, r := range rows {
		if r.status == "covered" && strings.Contains(r.evidence, "G13") {
			t.Errorf("covered row %s depends on G13", r.clause)
		}
	}
}
