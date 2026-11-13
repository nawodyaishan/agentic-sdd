package execution

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// TaskBriefRequest specifies the inputs for extracting and generating a task brief.
type TaskBriefRequest struct {
	RunID     string `json:"run_id,omitempty"`
	RunDir    string `json:"run_dir,omitempty"`
	RepoRoot  string `json:"repo_root,omitempty"`
	TasksPath string `json:"tasks_path,omitempty"`
	SpecPath  string `json:"spec_path,omitempty"`
	BatchID   string `json:"batch_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

// ExtractedTask holds the parsed task section and line bounds.
type ExtractedTask struct {
	TaskID      string
	BatchID     string
	Title       string
	LineStart   int
	LineEnd     int
	RawContent  string
	References  []string
	BatchHeader []string
}

// ResolvedReference represents a resolved requirement or acceptance criteria.
type ResolvedReference struct {
	ID      string
	Content string
}

// FileDigest returns the sha256 digest of data formatted as sha256:<hex>.
func FileDigest(data []byte) string {
	return DigestBytes(data)
}

// BatchSectionScopeDigest is an alias for NormalizedBatchScopeDigest.
func BatchSectionScopeDigest(tasksData []byte, batchID string) (string, error) {
	return NormalizedBatchScopeDigest(tasksData, batchID)
}

// TaskBrief extracts an authorized task section from tasks.md, resolves its FR/AC
// references in spec.md, renders a task brief, and writes briefs/<task>.md atomically.
func TaskBrief(req TaskBriefRequest) (*Brief, error) {
	// If RunDir is provided, attempt to load run.json if parameters are omitted.
	if strings.TrimSpace(req.RunDir) != "" {
		identity, err := LoadRunIdentity(req.RunDir)
		if err == nil {
			if identity.Scope.Kind == ScopeDirect {
				return nil, fmt.Errorf("%w: task brief is not supported for direct-scope runs", ErrUsage)
			}
			if strings.TrimSpace(req.RunID) == "" {
				req.RunID = identity.RunID
			}
			if strings.TrimSpace(req.BatchID) == "" {
				req.BatchID = identity.Scope.BatchID
			}
			if strings.TrimSpace(req.RepoRoot) == "" {
				req.RepoRoot = identity.Scope.RepoRoot
			}
			if strings.TrimSpace(req.TasksPath) == "" {
				req.TasksPath = identity.Scope.TasksPath
			}
			if strings.TrimSpace(req.SpecPath) == "" {
				req.SpecPath = identity.Scope.SpecPath
			}
		} else {
			if _, statErr := os.Stat(filepath.Join(req.RunDir, "run.json")); statErr == nil {
				return nil, err
			}
		}
	}

	if strings.TrimSpace(req.BatchID) == "" {
		return nil, fmt.Errorf("%w: batch ID is required", ErrUsage)
	}
	if strings.TrimSpace(req.TaskID) == "" {
		return nil, fmt.Errorf("%w: task ID is required", ErrUsage)
	}
	if strings.TrimSpace(req.TasksPath) == "" {
		return nil, fmt.Errorf("%w: tasks.md path is required", ErrUsage)
	}
	if strings.TrimSpace(req.SpecPath) == "" {
		return nil, fmt.Errorf("%w: spec.md path is required", ErrUsage)
	}

	tasksFullPath := req.TasksPath
	if !filepath.IsAbs(tasksFullPath) && strings.TrimSpace(req.RepoRoot) != "" {
		tasksFullPath = filepath.Join(req.RepoRoot, tasksFullPath)
	}
	specFullPath := req.SpecPath
	if !filepath.IsAbs(specFullPath) && strings.TrimSpace(req.RepoRoot) != "" {
		specFullPath = filepath.Join(req.RepoRoot, specFullPath)
	}

	tasksBytes, err := readBoundedRegular(tasksFullPath, MaxTasksFileBytes)
	if err != nil {
		return nil, err
	}
	specBytes, err := readBoundedRegular(specFullPath, MaxSpecFileBytes)
	if err != nil {
		return nil, err
	}

	extracted, err := ExtractTask(tasksBytes, req.BatchID, req.TaskID)
	if err != nil {
		return nil, err
	}

	resolvedRefs, err := ResolveReferences(specBytes, extracted.References)
	if err != nil {
		return nil, err
	}

	batchScopeDigest, err := NormalizedBatchScopeDigest(tasksBytes, req.BatchID)
	if err != nil {
		return nil, err
	}

	tasksDigest := DigestBytes(tasksBytes)
	specDigest := DigestBytes(specBytes)

	rendered := RenderTaskBrief(req, extracted, resolvedRefs, tasksDigest, batchScopeDigest, specDigest)
	briefBytes := []byte(rendered)
	briefDigest := DigestBytes(briefBytes)

	var briefPath string
	if strings.TrimSpace(req.RunDir) != "" {
		runDir := filepath.Clean(req.RunDir)
		info, err := os.Lstat(runDir)
		if err != nil {
			return nil, fmt.Errorf("%w: run directory is unreadable", ErrUnsafePath)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("%w: run directory is a symlink or not a directory", ErrUnsafePath)
		}
		if err := ensureArtifactDir(runDir, "briefs"); err != nil {
			return nil, err
		}
		briefPath = filepath.Join(runDir, "briefs", extracted.TaskID+".md")

		existingBytes, err := readBoundedRegular(briefPath, MaxSpecFileBytes)
		if err == nil {
			if DigestBytes(existingBytes) == briefDigest {
				// Idempotent no-op
				return &Brief{
					SchemaVersion: SchemaVersion,
					Kind:          "execution.task.brief",
					RunID:         req.RunID,
					TaskID:        extracted.TaskID,
					BriefPath:     briefPath,
					BriefDigest:   briefDigest,
					SourcePath:    req.TasksPath,
					LineStart:     extracted.LineStart,
					LineEnd:       extracted.LineEnd,
					References:    extracted.References,
					GeneratedAt:   time.Now().UTC(),
				}, nil
			}
			if !req.Force {
				return nil, fmt.Errorf("%w: brief %q already exists with differing digest (pass Force to overwrite)", ErrConflict, briefPath)
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}

		if err := writeFileAtomic(briefPath, briefBytes, 0600); err != nil {
			return nil, fmt.Errorf("%w: failed to write brief: %v", ErrUsage, err)
		}
	}

	return &Brief{
		SchemaVersion: SchemaVersion,
		Kind:          "execution.task.brief",
		RunID:         req.RunID,
		TaskID:        extracted.TaskID,
		BriefPath:     briefPath,
		BriefDigest:   briefDigest,
		SourcePath:    req.TasksPath,
		LineStart:     extracted.LineStart,
		LineEnd:       extracted.LineEnd,
		References:    extracted.References,
		GeneratedAt:   time.Now().UTC(),
	}, nil
}

type lineRecord struct {
	num     int
	raw     string
	inFence bool
}

// splitLinesWithFences tracks code fences and validates fence integrity.
func splitLinesWithFences(content []byte) ([]lineRecord, error) {
	rawText := string(content)
	// Split by newline while preserving lines
	rawLines := strings.Split(rawText, "\n")
	lines := make([]lineRecord, len(rawLines))

	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for i, l := range rawLines {
		raw := strings.TrimRight(l, "\r")
		trimmed := strings.TrimLeft(raw, " ")
		leadingSpaces := len(raw) - len(trimmed)

		currentInFence := inFence

		if !inFence {
			if leadingSpaces <= 3 && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
				ch := trimmed[0]
				count := 0
				for count < len(trimmed) && trimmed[count] == ch {
					count++
				}
				if count >= 3 {
					rest := trimmed[count:]
					if ch == '`' && strings.ContainsRune(rest, '`') {
						// Invalid backtick fence info string, ignore as fence
					} else {
						inFence = true
						fenceChar = ch
						fenceLen = count
					}
				}
			}
		} else {
			if leadingSpaces <= 3 && len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
				count := 0
				for count < len(trimmed) && trimmed[count] == fenceChar {
					count++
				}
				if count >= fenceLen {
					rest := strings.TrimSpace(trimmed[count:])
					if rest == "" {
						inFence = false
					}
				}
			}
		}

		lines[i] = lineRecord{
			num:     i + 1,
			raw:     raw,
			inFence: currentInFence,
		}
	}

	if inFence {
		return nil, fmt.Errorf("%w: malformed fenced code block (unclosed fence at EOF)", ErrUsage)
	}

	return lines, nil
}

type batchSection struct {
	id          string
	startLine   int
	endLine     int
	headingLine string
	headerLines []string
}

// parseBatches identifies all batch sections in tasks.md.
func parseBatches(lines []lineRecord) ([]batchSection, error) {
	var batches []batchSection
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if l.inFence {
			continue
		}
		if isBatchHeading(l.raw) {
			bID := extractBatchID(l.raw)
			startLine := l.num
			headingLine := l.raw

			// Find end of this batch: next level-2 or level-1 heading or EOF
			endLine := len(lines)
			for j := i + 1; j < len(lines); j++ {
				nextL := lines[j]
				if nextL.inFence {
					continue
				}
				if isBatchHeading(nextL.raw) || isLevel1Heading(nextL.raw) {
					endLine = nextL.num - 1
					break
				}
			}

			// Extract header lines before first task
			var headerLines []string
			for j := i + 1; j < len(lines) && lines[j].num <= endLine; j++ {
				hl := lines[j]
				if hl.inFence {
					continue
				}
				if isTaskHeading(hl.raw) || isTaskCheckbox(hl.raw) || isLevel2Heading(hl.raw) {
					break
				}
				trimmed := strings.TrimSpace(hl.raw)
				if startsWithIgnoreCase(trimmed, "**outcome:**") ||
					startsWithIgnoreCase(trimmed, "**verification:**") ||
					startsWithIgnoreCase(trimmed, "**state:**") {
					headerLines = append(headerLines, hl.raw)
				}
			}

			batches = append(batches, batchSection{
				id:          bID,
				startLine:   startLine,
				endLine:     endLine,
				headingLine: headingLine,
				headerLines: headerLines,
			})
		}
	}
	return batches, nil
}

func isLevel1Heading(raw string) bool {
	return (strings.HasPrefix(raw, "# ") || strings.HasPrefix(raw, "#\t")) && !strings.HasPrefix(raw, "##")
}

func isLevel2Heading(raw string) bool {
	return (strings.HasPrefix(raw, "## ") || strings.HasPrefix(raw, "##\t")) && !strings.HasPrefix(raw, "###")
}

func isBatchHeading(raw string) bool {
	return isLevel2Heading(raw)
}

func extractBatchID(raw string) string {
	after := strings.TrimSpace(raw[2:])
	if startsWithIgnoreCase(after, "batch ") || startsWithIgnoreCase(after, "batch\t") {
		after = strings.TrimSpace(after[5:])
	}
	// Take first token up to space, punctuation
	token := extractFirstToken(after)
	return token
}

func extractFirstToken(s string) string {
	separators := " \t—–:-("
	idx := strings.IndexAny(s, separators)
	if idx != -1 {
		s = s[:idx]
	}
	s = strings.TrimRight(s, ":-—–")
	return strings.TrimSpace(s)
}

func startsWithIgnoreCase(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return strings.EqualFold(s[:len(prefix)], prefix)
}

// parseTaskHeading checks if a level-3 heading is a task heading.
// Returns (taskID, isTask, isMalformed, err).
func parseTaskHeading(raw string) (string, bool, bool, error) {
	if !strings.HasPrefix(raw, "### ") && !strings.HasPrefix(raw, "###\t") {
		return "", false, false, nil
	}
	if strings.HasPrefix(raw, "####") {
		return "", false, false, nil
	}
	text := strings.TrimSpace(raw[3:])
	if text == "" {
		return "", false, false, nil
	}

	// Check if it starts with T or t followed by digits
	reT := regexp.MustCompile(`^(?i)T(\d+)(.*)$`)
	if match := reT.FindStringSubmatch(text); match != nil {
		digits := match[1]
		rest := match[2]
		if rest == "" || startsWithSeparator(rest) {
			return "T" + digits, true, false, nil
		}
		// Malformed: e.g. T01abc, T01?
		return "", false, true, fmt.Errorf("malformed task heading %q", raw)
	}

	// Check if it starts with Task (case-insensitive)
	reTask := regexp.MustCompile(`^(?i)Task\s*(\d*)(.*)$`)
	if match := reTask.FindStringSubmatch(text); match != nil {
		digits := match[1]
		rest := match[2]
		if digits == "" {
			return "", false, true, fmt.Errorf("malformed task heading missing task number: %q", raw)
		}
		if rest == "" || startsWithSeparator(rest) {
			return "T" + digits, true, false, nil
		}
		return "", false, true, fmt.Errorf("malformed task heading: %q", raw)
	}

	// Check if it starts with just T followed by non-digits
	reJustT := regexp.MustCompile(`^(?i)T\b(.*)$`)
	if match := reJustT.FindStringSubmatch(text); match != nil {
		rest := match[1]
		if rest == "" || startsWithSeparator(rest) {
			return "", false, true, fmt.Errorf("malformed task heading with missing number: %q", raw)
		}
	}

	// Check if it's a known non-task heading or malformed T-identifier
	firstWord := extractFirstToken(text)
	if strings.HasPrefix(strings.ToLower(firstWord), "t") {
		commonNonTaskWords := []string{
			"test", "testing", "types", "tools", "target", "technical",
			"timing", "table", "trace", "tui", "taskfile", "tasks",
		}
		lowerWord := strings.ToLower(firstWord)
		isCommon := false
		for _, w := range commonNonTaskWords {
			if lowerWord == w || strings.HasPrefix(lowerWord, w) {
				isCommon = true
				break
			}
		}
		if !isCommon && strings.ContainsAny(firstWord, "?!@#$%^&*") {
			return "", false, true, fmt.Errorf("malformed task heading: %q", raw)
		}
	}

	// Ordinary non-task level 3 heading
	return "", false, false, nil
}

func startsWithSeparator(s string) bool {
	if len(s) == 0 {
		return true
	}
	ch := s[0]
	if ch == ' ' || ch == '\t' || ch == ':' || ch == '-' || ch == '(' {
		return true
	}
	// Check UTF-8 dashes: em-dash (—) and en-dash (–)
	if strings.HasPrefix(s, "—") || strings.HasPrefix(s, "–") {
		return true
	}
	return false
}

// parseTaskCheckbox checks for bold checkbox items: - [ ] **T01** or - [x] **T01**
func parseTaskCheckbox(raw string) (string, bool) {
	re := regexp.MustCompile(`^\s*[-*]\s*\[[ xX]\]\s*\*\*([^*]+)\*\*(.*)$`)
	match := re.FindStringSubmatch(raw)
	if match == nil {
		return "", false
	}
	token := strings.TrimSpace(match[1])
	token = strings.TrimRight(token, ":")
	reT := regexp.MustCompile(`^(?i)T(\d+)$`)
	if matchT := reT.FindStringSubmatch(token); matchT != nil {
		return "T" + matchT[1], true
	}
	reTask := regexp.MustCompile(`^(?i)Task\s*(\d+)$`)
	if matchTask := reTask.FindStringSubmatch(token); matchTask != nil {
		return "T" + matchTask[1], true
	}
	return token, true
}

func isTaskHeading(raw string) bool {
	id, isTask, _, _ := parseTaskHeading(raw)
	return isTask && id != ""
}

func isTaskCheckbox(raw string) bool {
	_, ok := parseTaskCheckbox(raw)
	return ok
}

type taskLocation struct {
	taskID    string
	batchID   string
	lineStart int
	lineEnd   int
	lines     []string
}

// parseTasksInBatch scans the lines of one batch for task sections.
func parseTasksInBatch(lines []lineRecord, batch batchSection) ([]taskLocation, error) {
	var tasks []taskLocation
	var currentTaskID string
	var currentStartLine int
	var currentLines []string
	var seenCheckboxInCurrent bool

	finishCurrent := func(endLine int) {
		if currentTaskID == "" || len(currentLines) == 0 {
			return
		}
		// Trim trailing blank lines
		lastIdx := len(currentLines) - 1
		for lastIdx >= 0 && strings.TrimSpace(currentLines[lastIdx]) == "" {
			lastIdx--
		}
		trimmedLines := currentLines[:lastIdx+1]
		actualEnd := currentStartLine + len(trimmedLines) - 1
		if actualEnd < currentStartLine {
			actualEnd = currentStartLine
		}

		tasks = append(tasks, taskLocation{
			taskID:    currentTaskID,
			batchID:   batch.id,
			lineStart: currentStartLine,
			lineEnd:   actualEnd,
			lines:     trimmedLines,
		})
		currentTaskID = ""
		currentStartLine = 0
		currentLines = nil
		seenCheckboxInCurrent = false
	}

	for i := batch.startLine - 1; i < batch.endLine && i < len(lines); i++ {
		lr := lines[i]

		if lr.inFence {
			if currentTaskID != "" {
				currentLines = append(currentLines, lr.raw)
			}
			continue
		}

		// Check for level 2 or level 1 heading (end of batch or unexpected)
		if i > batch.startLine-1 && (isLevel2Heading(lr.raw) || isLevel1Heading(lr.raw)) {
			finishCurrent(lr.num - 1)
			break
		}

		// Check for level 3 heading
		taskID, isTask, isMalformed, err := parseTaskHeading(lr.raw)
		if isMalformed {
			return nil, fmt.Errorf("%w: %v", ErrUsage, err)
		}

		if isTask {
			// Ends previous task section and starts new task
			finishCurrent(lr.num - 1)
			currentTaskID = taskID
			currentStartLine = lr.num
			currentLines = []string{lr.raw}
			seenCheckboxInCurrent = false
			continue
		}

		// Check for non-task level 3 heading
		if (strings.HasPrefix(lr.raw, "### ") || strings.HasPrefix(lr.raw, "###\t")) && !strings.HasPrefix(lr.raw, "####") {
			// Non-task level 3 heading ends the current task section
			finishCurrent(lr.num - 1)
			continue
		}

		// Check for bold checkbox task
		cbID, isCb := parseTaskCheckbox(lr.raw)
		if isCb {
			if currentTaskID == "" {
				// Standalone checkbox starts a task
				currentTaskID = cbID
				currentStartLine = lr.num
				currentLines = []string{lr.raw}
				seenCheckboxInCurrent = true
				continue
			} else if strings.EqualFold(currentTaskID, cbID) {
				if seenCheckboxInCurrent {
					// Duplicate checkbox for same task in this section
					return nil, fmt.Errorf("%w: duplicate task %q checkbox in batch %q", ErrUsage, cbID, batch.id)
				}
				seenCheckboxInCurrent = true
				currentLines = append(currentLines, lr.raw)
				continue
			} else {
				// Different checkbox starts a new task in checkbox-only document
				finishCurrent(lr.num - 1)
				currentTaskID = cbID
				currentStartLine = lr.num
				currentLines = []string{lr.raw}
				seenCheckboxInCurrent = true
				continue
			}
		}

		// Nested heading (####) or content line
		if currentTaskID != "" {
			currentLines = append(currentLines, lr.raw)
		}
	}

	finishCurrent(batch.endLine)
	return tasks, nil
}

// ExtractTask parses tasks.md and extracts the requested task from the requested batch.
func ExtractTask(tasksBytes []byte, batchID, taskID string) (*ExtractedTask, error) {
	if len(tasksBytes) > MaxTasksFileBytes {
		return nil, fmt.Errorf("%w: tasks.md exceeds maximum size %d bytes", ErrUsage, MaxTasksFileBytes)
	}
	if !utf8.Valid(tasksBytes) {
		return nil, fmt.Errorf("%w: tasks.md contains invalid UTF-8", ErrUsage)
	}

	lines, err := splitLinesWithFences(tasksBytes)
	if err != nil {
		return nil, err
	}

	batches, err := parseBatches(lines)
	if err != nil {
		return nil, err
	}

	var matchedBatches []batchSection
	for _, b := range batches {
		if strings.EqualFold(b.id, batchID) {
			matchedBatches = append(matchedBatches, b)
		}
	}

	if len(matchedBatches) == 0 {
		return nil, fmt.Errorf("%w: batch %q not found in tasks.md", ErrNotFound, batchID)
	}
	if len(matchedBatches) > 1 {
		return nil, fmt.Errorf("%w: duplicate batch %q in tasks.md", ErrUsage, batchID)
	}

	selectedBatch := matchedBatches[0]

	tasksInBatch, err := parseTasksInBatch(lines, selectedBatch)
	if err != nil {
		return nil, err
	}

	var matchingTasks []taskLocation
	for _, t := range tasksInBatch {
		if strings.EqualFold(t.taskID, taskID) {
			matchingTasks = append(matchingTasks, t)
		}
	}

	if len(matchingTasks) > 1 {
		return nil, fmt.Errorf("%w: duplicate task %q in batch %q", ErrUsage, taskID, batchID)
	}

	if len(matchingTasks) == 0 {
		// Check if task exists in another batch
		for _, b := range batches {
			if strings.EqualFold(b.id, batchID) {
				continue
			}
			otherTasks, err := parseTasksInBatch(lines, b)
			if err != nil {
				continue
			}
			for _, ot := range otherTasks {
				if strings.EqualFold(ot.taskID, taskID) {
					return nil, fmt.Errorf("%w: task %q appears in batch %q only, not in selected batch %q", ErrUsage, taskID, b.id, batchID)
				}
			}
		}
		return nil, fmt.Errorf("%w: task %q not found in tasks.md", ErrNotFound, taskID)
	}

	match := matchingTasks[0]
	rawContent := strings.Join(match.lines, "\n")
	refs := ExtractReferences(rawContent)

	return &ExtractedTask{
		TaskID:      match.taskID,
		BatchID:     batchID,
		LineStart:   match.lineStart,
		LineEnd:     match.lineEnd,
		RawContent:  rawContent,
		References:  refs,
		BatchHeader: selectedBatch.headerLines,
	}, nil
}

// ExtractReferences scans text for FR-xx and AC-xx references, expanding lists and ranges.
func ExtractReferences(text string) []string {
	refSet := make(map[string]bool)

	// 1. Match ranges: (FR|AC)-(\d+)\s*(?:[–—\-]|to|through|\.\.)\s*(?:(FR|AC)-)?(\d+)
	reRange := regexp.MustCompile(`\b(FR|AC)-(\d+)\s*(?:[–—\-]|to|through|\.\.)\s*(?:(FR|AC)-)?(\d+)\b`)
	for _, match := range reRange.FindAllStringSubmatch(text, -1) {
		p1 := strings.ToUpper(match[1])
		n1Str := match[2]
		p2 := strings.ToUpper(match[3])
		n2Str := match[4]

		if p2 == "" {
			p2 = p1
		}
		if p1 == p2 {
			n1, err1 := strconv.Atoi(n1Str)
			n2, err2 := strconv.Atoi(n2Str)
			if err1 == nil && err2 == nil && n1 <= n2 {
				width := len(n1Str)
				for n := n1; n <= n2; n++ {
					refSet[fmt.Sprintf("%s-%0*d", p1, width, n)] = true
				}
			}
		}
	}

	// 2. Match lists: (FR|AC)-(\d+)((?:[/,]\s*(?:(?:FR|AC)-)?\d+)+)
	reList := regexp.MustCompile(`\b(FR|AC)-(\d+)((?:\s*[/,]\s*(?:(?:FR|AC)-)?\d+)+)\b`)
	for _, match := range reList.FindAllStringSubmatch(text, -1) {
		prefix := strings.ToUpper(match[1])
		initialNum := match[2]
		width := len(initialNum)
		refSet[fmt.Sprintf("%s-%s", prefix, initialNum)] = true

		rest := match[3]
		items := strings.FieldsFunc(rest, func(r rune) bool {
			return r == '/' || r == ',' || r == ' ' || r == '\t'
		})
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if strings.HasPrefix(strings.ToUpper(item), "FR-") || strings.HasPrefix(strings.ToUpper(item), "AC-") {
				refSet[strings.ToUpper(item)] = true
			} else {
				if n, err := strconv.Atoi(item); err == nil {
					refSet[fmt.Sprintf("%s-%0*d", prefix, width, n)] = true
				}
			}
		}
	}

	// 3. Match individual (FR|AC)-(\d+)
	reSingle := regexp.MustCompile(`\b(FR|AC)-(\d+)\b`)
	for _, match := range reSingle.FindAllStringSubmatch(text, -1) {
		prefix := strings.ToUpper(match[1])
		numStr := match[2]
		refSet[fmt.Sprintf("%s-%s", prefix, numStr)] = true
	}

	var result []string
	for r := range refSet {
		result = append(result, r)
	}
	sort.Strings(result)
	return result
}

// ResolveReferences resolves every referenced ID uniquely in spec.md.
func ResolveReferences(specBytes []byte, refIDs []string) ([]ResolvedReference, error) {
	if len(refIDs) == 0 {
		return nil, nil
	}
	if len(specBytes) > MaxTasksFileBytes {
		return nil, fmt.Errorf("%w: spec.md exceeds maximum size %d bytes", ErrUsage, MaxTasksFileBytes)
	}
	if !utf8.Valid(specBytes) {
		return nil, fmt.Errorf("%w: spec.md contains invalid UTF-8", ErrUsage)
	}

	lines, err := splitLinesWithFences(specBytes)
	if err != nil {
		return nil, err
	}

	var resolved []ResolvedReference

	for _, id := range refIDs {
		definitions := findReferenceDefinitions(lines, id)
		if len(definitions) == 0 {
			return nil, fmt.Errorf("%w: required reference %q not found in spec.md", ErrUsage, id)
		}
		if len(definitions) > 1 {
			return nil, fmt.Errorf("%w: duplicate definition for reference %q in spec.md (%d definitions found)", ErrUsage, id, len(definitions))
		}
		resolved = append(resolved, ResolvedReference{
			ID:      id,
			Content: definitions[0],
		})
	}

	return resolved, nil
}

func findReferenceDefinitions(lines []lineRecord, id string) []string {
	var definitions []string

	// Patterns:
	// A. Table row: | **AC-01** | ... | or | AC-01 | ... |
	tableRe := regexp.MustCompile(`(?i)^\s*\|\s*(\*\*)?` + regexp.QuoteMeta(id) + `(\*\*)?\s*\|(.*)$`)

	// B. Heading: ### FR-01 — ... or ### **FR-01** ...
	headingRe := regexp.MustCompile(`(?i)^#{2,3}\s+(\*\*)?` + regexp.QuoteMeta(id) + `(\*\*)?(\s*[-—–:].*|\s+.*)?$`)

	// C. List item: - **AC-01** ... or - AC-01: ...
	listRe := regexp.MustCompile(`(?i)^\s*[-*]\s+(\*\*)?` + regexp.QuoteMeta(id) + `(\*\*)?[:\s](.*)$`)

	for i := 0; i < len(lines); i++ {
		lr := lines[i]
		if lr.inFence {
			continue
		}

		if tableRe.MatchString(lr.raw) {
			definitions = append(definitions, lr.raw)
			continue
		}

		if headingRe.MatchString(lr.raw) {
			// Collect heading and content until next heading of equal or higher level
			var sectionLines []string
			sectionLines = append(sectionLines, lr.raw)
			headingLevel := 2
			if strings.HasPrefix(lr.raw, "###") {
				headingLevel = 3
			}

			for j := i + 1; j < len(lines); j++ {
				nextL := lines[j]
				if nextL.inFence {
					sectionLines = append(sectionLines, nextL.raw)
					continue
				}
				if isLevelHeading(nextL.raw, headingLevel) {
					break
				}
				sectionLines = append(sectionLines, nextL.raw)
			}
			definitions = append(definitions, strings.Join(sectionLines, "\n"))
			continue
		}

		if listRe.MatchString(lr.raw) {
			// Collect list item and indented lines
			var itemLines []string
			itemLines = append(itemLines, lr.raw)
			for j := i + 1; j < len(lines); j++ {
				nextL := lines[j]
				if nextL.inFence {
					itemLines = append(itemLines, nextL.raw)
					continue
				}
				trimmed := strings.TrimSpace(nextL.raw)
				if trimmed == "" {
					itemLines = append(itemLines, nextL.raw)
					continue
				}
				if strings.HasPrefix(nextL.raw, "  ") || strings.HasPrefix(nextL.raw, "\t") {
					itemLines = append(itemLines, nextL.raw)
				} else {
					break
				}
			}
			definitions = append(definitions, strings.Join(itemLines, "\n"))
			continue
		}
	}

	return definitions
}

func isLevelHeading(raw string, maxLevel int) bool {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}
	count := 0
	for count < len(trimmed) && trimmed[count] == '#' {
		count++
	}
	if count > 0 && count <= maxLevel && len(trimmed) > count && (trimmed[count] == ' ' || trimmed[count] == '\t') {
		return true
	}
	return false
}

// RenderTaskBrief builds the task brief markdown document.
func RenderTaskBrief(
	req TaskBriefRequest,
	task *ExtractedTask,
	resolvedRefs []ResolvedReference,
	tasksDigest string,
	batchScopeDigest string,
	specDigest string,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Task Brief: %s\n\n", task.TaskID))

	sb.WriteString("## Identity\n")
	if req.RunID != "" {
		sb.WriteString(fmt.Sprintf("- **Run ID:** %s\n", req.RunID))
	}
	sb.WriteString(fmt.Sprintf("- **Task ID:** %s\n", task.TaskID))
	sb.WriteString(fmt.Sprintf("- **Batch ID:** %s\n", task.BatchID))
	sb.WriteString(fmt.Sprintf("- **Tasks Source:** %s (lines %d–%d)\n", req.TasksPath, task.LineStart, task.LineEnd))
	sb.WriteString(fmt.Sprintf("- **Tasks Full Digest:** %s\n", tasksDigest))
	sb.WriteString(fmt.Sprintf("- **Batch Scope Digest:** %s\n", batchScopeDigest))
	sb.WriteString(fmt.Sprintf("- **Spec Source:** %s\n", req.SpecPath))
	sb.WriteString(fmt.Sprintf("- **Spec Digest:** %s\n\n", specDigest))

	if len(task.BatchHeader) > 0 {
		sb.WriteString("## Batch Context\n")
		for _, hl := range task.BatchHeader {
			sb.WriteString(hl + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Task Definition\n")
	sb.WriteString(task.RawContent)
	sb.WriteString("\n\n")

	if len(resolvedRefs) > 0 {
		sb.WriteString("## Scoped Requirements and Acceptance Criteria\n")
		// Group table rows together if applicable
		inTable := false
		for _, ref := range resolvedRefs {
			content := strings.TrimSpace(ref.Content)
			if strings.HasPrefix(content, "|") {
				if !inTable {
					sb.WriteString("| ID | Requirement / Condition |\n|---|---|\n")
					inTable = true
				}
				sb.WriteString(content + "\n")
			} else {
				if inTable {
					sb.WriteString("\n")
					inTable = false
				}
				sb.WriteString(content + "\n\n")
			}
		}
		if inTable {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## Authority and Stop Conditions\n")
	sb.WriteString("- This task brief is an execution observation and scoped assignment, not independent approval.\n")
	sb.WriteString(fmt.Sprintf("- Work only within the assigned batch (%s) and task (%s).\n", task.BatchID, task.TaskID))
	sb.WriteString("- Assignment wording grants no permission to commit, push, deploy, migrate, rotate credentials, run paid inference beyond what was authorized, spawn further children, start another batch, or answer its own questions about human intent.\n")
	sb.WriteString("- Stop conditions:\n")
	sb.WriteString("  1. Any path or file outside the authorized scope.\n")
	sb.WriteString("  2. Pre-existing check or test failures that cannot be resolved within scope.\n")
	sb.WriteString("  3. Missing approvals or ambiguous requirements.\n")
	sb.WriteString("  4. Interrupted writes or uncertain state.\n")

	return sb.String()
}
