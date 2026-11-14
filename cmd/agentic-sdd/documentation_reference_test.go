package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// cliSourceFacts holds the AST-extracted CLI dispatch, help-topic, FlagSet,
// and usage-text facts from the currently selected supported-core source files.
type cliSourceFacts struct {
	SelectedGoFiles   []string
	IgnoredGoFiles    []string
	TopLevelCommands  []string
	HelpTopics        []string
	TopicHelpKeys     []string
	HelpEarlyTopics   []string
	TraceCommands     []string
	FamilySubcommands map[string][]string
	FlagSets          map[string][]string
	UsageTexts        map[string]string
}

type flagSetHelperInfo struct {
	paramName    string
	nameTemplate ast.Expr
	baseFlags    []string
}

var (
	flagVarMethods = map[string]bool{
		"StringVar":   true,
		"BoolVar":     true,
		"IntVar":      true,
		"Int64Var":    true,
		"UintVar":     true,
		"Uint64Var":   true,
		"Float64Var":  true,
		"DurationVar": true,
		"Var":         true,
	}
	flagValueMethods = map[string]bool{
		"String":   true,
		"Bool":     true,
		"Int":      true,
		"Int64":    true,
		"Uint":     true,
		"Uint64":   true,
		"Float64":  true,
		"Duration": true,
		"Func":     true,
		"BoolFunc": true,
	}
	flagRuntimeMethods = map[string]bool{
		"SetOutput":     true,
		"Parse":         true,
		"NArg":          true,
		"Arg":           true,
		"Args":          true,
		"Visit":         true,
		"Lookup":        true,
		"PrintDefaults": true,
		"Usage":         true,
		"Name":          true,
	}
	allowedFlagSetParseHelpers = map[string]bool{
		"parseMCPFlags":       true,
		"parseMCPWriteArgs":   true,
		"parseExecutionFlags": true,
	}
)

func extractCLISourceFactsFromDir(dir string) (*cliSourceFacts, error) {
	ctxt := build.Default
	ctxt.BuildTags = nil // Default supported-core profile: excludes agentic_sdd_experimental.
	pkg, err := ctxt.ImportDir(dir, 0)
	if err != nil {
		return nil, fmt.Errorf("import dir %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	var parsed []*ast.File
	for _, name := range pkg.GoFiles {
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		parsed = append(parsed, file)
	}
	consts, err := collectStringConsts(fset, parsed)
	if err != nil {
		return nil, err
	}
	// Resolve internal/buildinfo string constants from Go source without running anything external.
	buildinfoDir := filepath.Join(dir, "..", "..", "internal", "buildinfo")
	info, statErr := os.Stat(buildinfoDir)
	if statErr != nil || !info.IsDir() {
		return nil, fmt.Errorf("missing internal/buildinfo directory at %s: %v", buildinfoDir, statErr)
	}
	biPkg, err := ctxt.ImportDir(buildinfoDir, 0)
	if err != nil {
		return nil, fmt.Errorf("import buildinfo dir %s: %w", buildinfoDir, err)
	}
	var biFiles []*ast.File
	for _, name := range biPkg.GoFiles {
		file, err := parser.ParseFile(fset, filepath.Join(buildinfoDir, name), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse buildinfo/%s: %w", name, err)
		}
		biFiles = append(biFiles, file)
	}
	biConsts, err := collectStringConsts(fset, biFiles)
	if err != nil {
		return nil, err
	}
	for k, v := range biConsts {
		consts["buildinfo."+k] = v
	}

	facts, err := extractCLISourceFactsFromAST(fset, parsed, consts)
	if err != nil {
		return nil, err
	}
	facts.SelectedGoFiles = append([]string(nil), pkg.GoFiles...)
	facts.IgnoredGoFiles = append([]string(nil), pkg.IgnoredGoFiles...)
	sort.Strings(facts.SelectedGoFiles)
	sort.Strings(facts.IgnoredGoFiles)
	return facts, nil
}

func extractCLISourceFactsFromSource(filename, src string) (*cliSourceFacts, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	parsed := []*ast.File{file}
	consts, err := collectStringConsts(fset, parsed)
	if err != nil {
		return nil, err
	}
	facts, err := extractCLISourceFactsFromAST(fset, parsed, consts)
	if err != nil {
		return nil, err
	}
	facts.SelectedGoFiles = []string{filename}
	return facts, nil
}

func collectStringConsts(fset *token.FileSet, files []*ast.File) (map[string]string, error) {
	consts := make(map[string]string)
	// Two passes to resolve simple constant-to-constant references within the package.
	for pass := 0; pass < 2; pass++ {
		for _, file := range files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.CONST {
					continue
				}
				for _, spec := range gen.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range vs.Names {
						if i >= len(vs.Values) {
							continue
						}
						if val, err := evalStringExpr(vs.Values[i], consts, nil); err == nil {
							consts[name.Name] = val
						}
					}
				}
			}
		}
	}
	return consts, nil
}

func evalStringExpr(expr ast.Expr, consts map[string]string, locals map[string]string) (string, error) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", fmt.Errorf("non-string literal %s", e.Value)
		}
		return strconv.Unquote(e.Value)
	case *ast.ParenExpr:
		return evalStringExpr(e.X, consts, locals)
	case *ast.Ident:
		if locals != nil {
			if val, ok := locals[e.Name]; ok {
				return val, nil
			}
		}
		if val, ok := consts[e.Name]; ok {
			return val, nil
		}
		return "", fmt.Errorf("unresolved identifier %q", e.Name)
	case *ast.SelectorExpr:
		pkgIdent, ok := e.X.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("unsupported selector expression %T", e.X)
		}
		key := pkgIdent.Name + "." + e.Sel.Name
		if val, ok := consts[key]; ok {
			return val, nil
		}
		return "", fmt.Errorf("unresolved selector %q", key)
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", fmt.Errorf("unsupported binary op %s", e.Op)
		}
		left, err := evalStringExpr(e.X, consts, locals)
		if err != nil {
			return "", err
		}
		right, err := evalStringExpr(e.Y, consts, locals)
		if err != nil {
			return "", err
		}
		return left + right, nil
	default:
		return "", fmt.Errorf("non-constant string expression %T", expr)
	}
}

func extractCLISourceFactsFromAST(fset *token.FileSet, files []*ast.File, consts map[string]string) (*cliSourceFacts, error) {
	facts := &cliSourceFacts{
		FamilySubcommands: make(map[string][]string),
		FlagSets:          make(map[string][]string),
		UsageTexts:        make(map[string]string),
	}

	// Collect all top-level function declarations and map FlagSet-factory helpers first.
	var funcs []*ast.FuncDecl
	helpers := make(map[string]flagSetHelperInfo)

	for _, file := range files {
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
				for _, spec := range gen.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range vs.Names {
						if name.Name == "topicHelp" && i < len(vs.Values) {
							keys, err := extractMapLiteralKeys(fset, vs.Values[i], consts)
							if err != nil {
								return nil, err
							}
							facts.TopicHelpKeys = keys
						}
					}
				}
			}
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			funcs = append(funcs, fn)
			if returnsFlagSet(fn.Type) {
				info, err := inspectFlagSetHelper(fset, fn, consts)
				if err != nil {
					return nil, err
				}
				helpers[fn.Name.Name] = info
			}
		}
	}

	// Verify helper parsers (parseMCPFlags, etc.) do not register flags on the FlagSet they receive.
	for _, fn := range funcs {
		if allowedFlagSetParseHelpers[fn.Name.Name] {
			if err := verifyNoFlagRegistrationsInParserHelper(fset, fn); err != nil {
				return nil, err
			}
		}
	}

	// Scan every function for FlagSets, global flag usage, dispatch switches, and print*Usage texts.
	for _, fn := range funcs {
		if _, isHelper := helpers[fn.Name.Name]; !isHelper {
			if err := extractFlagSetsInFunc(fset, fn, consts, helpers, facts.FlagSets); err != nil {
				return nil, err
			}
		}

		if strings.HasPrefix(fn.Name.Name, "print") && strings.HasSuffix(fn.Name.Name, "Usage") {
			text, err := extractUsageText(fset, fn, consts)
			if err != nil {
				return nil, err
			}
			facts.UsageTexts[fn.Name.Name] = text
		}

		switch fn.Name.Name {
		case "runEnv":
			cmds, err := extractSwitchCases(fset, fn, "args[0]", consts)
			if err != nil {
				return nil, err
			}
			facts.TopLevelCommands = cmds
			early, err := extractRunEnvHelpEarlyTopics(fset, fn, consts)
			if err != nil {
				return nil, err
			}
			facts.HelpEarlyTopics = early
		case "isHelpTopic":
			topics, err := extractSwitchCases(fset, fn, "s", consts)
			if err != nil {
				return nil, err
			}
			facts.HelpTopics = topics
		case "commandName":
			cmds, err := extractSwitchCases(fset, fn, "arg", consts)
			if err != nil {
				return nil, err
			}
			facts.TraceCommands = cmds
		case "runRestore":
			subs, err := extractSwitchCases(fset, fn, "args[0]", consts)
			if err != nil {
				return nil, err
			}
			facts.FamilySubcommands[fn.Name.Name] = subs
		case "runFleet", "runMCP", "runMCPRestore":
			subs, err := extractSwitchCases(fset, fn, "sub", consts)
			if err != nil {
				return nil, err
			}
			facts.FamilySubcommands[fn.Name.Name] = subs
		case "runRun", "runTaskCommand", "runReview":
			subs, err := extractSwitchCases(fset, fn, "args[0]", consts)
			if err != nil {
				return nil, err
			}
			facts.FamilySubcommands[fn.Name.Name] = subs
		}
	}

	return facts, nil
}

func returnsFlagSet(ft *ast.FuncType) bool {
	if ft == nil || ft.Results == nil {
		return false
	}
	for _, field := range ft.Results.List {
		star, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		pkg, ok := sel.X.(*ast.Ident)
		if ok && pkg.Name == "flag" && sel.Sel.Name == "FlagSet" {
			return true
		}
	}
	return false
}

func firstStringParamName(ft *ast.FuncType) (string, bool) {
	if ft == nil || ft.Params == nil {
		return "", false
	}
	for _, field := range ft.Params.List {
		ident, ok := field.Type.(*ast.Ident)
		if ok && ident.Name == "string" && len(field.Names) > 0 {
			return field.Names[0].Name, true
		}
	}
	return "", false
}

func inspectFlagSetHelper(fset *token.FileSet, fn *ast.FuncDecl, consts map[string]string) (flagSetHelperInfo, error) {
	paramName, ok := firstStringParamName(fn.Type)
	if !ok {
		return flagSetHelperInfo{}, fmt.Errorf("%s: FlagSet helper %s must take a string name parameter", fset.Position(fn.Pos()), fn.Name.Name)
	}

	var info flagSetHelperInfo
	info.paramName = paramName
	var fsVar string
	var flags []string
	var inspectErr error

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		switch node := n.(type) {
		case *ast.AssignStmt:
			if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
				if call, ok := node.Rhs[0].(*ast.CallExpr); ok && isFlagPkgCall(call, "NewFlagSet") {
					lhs, ok := node.Lhs[0].(*ast.Ident)
					if !ok || len(call.Args) < 1 {
						inspectErr = fmt.Errorf("%s: unrecognized flag.NewFlagSet assignment in %s", fset.Position(node.Pos()), fn.Name.Name)
						return false
					}
					if fsVar != "" {
						inspectErr = fmt.Errorf("%s: multiple FlagSets in helper %s", fset.Position(node.Pos()), fn.Name.Name)
						return false
					}
					if _, err := evalStringExpr(call.Args[0], consts, map[string]string{paramName: "<param>"}); err != nil {
						inspectErr = fmt.Errorf("%s: non-constant FlagSet name in helper %s: %w", fset.Position(call.Pos()), fn.Name.Name, err)
						return false
					}
					fsVar = lhs.Name
					info.nameTemplate = call.Args[0]
					return true
				}
			}
		case *ast.CallExpr:
			if pkgMethod, ok := flagPkgCallName(node); ok && pkgMethod != "NewFlagSet" {
				inspectErr = fmt.Errorf("%s: disallowed package-level flag.%s call in %s", fset.Position(node.Pos()), pkgMethod, fn.Name.Name)
				return false
			}
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if recv, ok := sel.X.(*ast.Ident); ok && fsVar != "" && recv.Name == fsVar {
					name, registered, err := classifyFlagSetMethodCall(fset, fn.Name.Name, node, sel.Sel.Name, consts)
					if err != nil {
						inspectErr = err
						return false
					}
					if registered {
						flags = append(flags, name)
					}
				}
			}
		}
		return true
	})

	if inspectErr != nil {
		return flagSetHelperInfo{}, inspectErr
	}
	if fsVar == "" || info.nameTemplate == nil {
		return flagSetHelperInfo{}, fmt.Errorf("%s: FlagSet helper %s does not construct flag.NewFlagSet", fset.Position(fn.Pos()), fn.Name.Name)
	}
	info.baseFlags = uniqueSorted(flags)
	return info, nil
}

func verifyNoFlagRegistrationsInParserHelper(fset *token.FileSet, fn *ast.FuncDecl) error {
	var checkErr error
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if checkErr != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if pkgMethod, ok := flagPkgCallName(call); ok {
			checkErr = fmt.Errorf("%s: disallowed flag.%s in parser helper %s", fset.Position(call.Pos()), pkgMethod, fn.Name.Name)
			return false
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if flagVarMethods[sel.Sel.Name] || flagValueMethods[sel.Sel.Name] {
				checkErr = fmt.Errorf("%s: parser helper %s registers flag via %s", fset.Position(call.Pos()), fn.Name.Name, sel.Sel.Name)
				return false
			}
		}
		return true
	})
	return checkErr
}

func extractFlagSetsInFunc(
	fset *token.FileSet,
	fn *ast.FuncDecl,
	consts map[string]string,
	helpers map[string]flagSetHelperInfo,
	out map[string][]string,
) error {
	fsVars := make(map[string]string)            // varName -> FlagSet name
	fsFlags := make(map[string][]string)         // FlagSet name -> flag names
	trackedCalls := make(map[*ast.CallExpr]bool) // NewFlagSet / helper calls bound to fsVars
	var inspectErr error

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		switch node := n.(type) {
		case *ast.ValueSpec:
			for _, val := range node.Values {
				if call, ok := val.(*ast.CallExpr); ok {
					if isFlagPkgCall(call, "NewFlagSet") {
						inspectErr = fmt.Errorf("%s: unrecognized var FlagSet declaration in %s", fset.Position(node.Pos()), fn.Name.Name)
						return false
					}
					if ident, ok := call.Fun.(*ast.Ident); ok {
						if _, isHelper := helpers[ident.Name]; isHelper {
							inspectErr = fmt.Errorf("%s: unrecognized var FlagSet helper declaration in %s", fset.Position(node.Pos()), fn.Name.Name)
							return false
						}
					}
				}
			}
		case *ast.AssignStmt:
			if len(node.Lhs) != len(node.Rhs) {
				return true
			}
			for i, rhs := range node.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					// Fail closed if an existing FlagSet variable is aliased.
					if ident, ok := rhs.(*ast.Ident); ok {
						if _, isFS := fsVars[ident.Name]; isFS {
							inspectErr = fmt.Errorf("%s: unrecognized alias of FlagSet variable %s in %s", fset.Position(node.Pos()), ident.Name, fn.Name.Name)
							return false
						}
					}
					continue
				}
				if isFlagPkgCall(call, "NewFlagSet") {
					lhs, ok := node.Lhs[i].(*ast.Ident)
					if !ok || len(call.Args) < 1 {
						inspectErr = fmt.Errorf("%s: unrecognized flag.NewFlagSet assignment in %s", fset.Position(node.Pos()), fn.Name.Name)
						return false
					}
					fsName, err := evalStringExpr(call.Args[0], consts, nil)
					if err != nil {
						inspectErr = fmt.Errorf("%s: non-constant FlagSet name in %s: %w", fset.Position(call.Pos()), fn.Name.Name, err)
						return false
					}
					if _, dup := out[fsName]; dup {
						inspectErr = fmt.Errorf("%s: duplicate FlagSet %q in %s", fset.Position(call.Pos()), fsName, fn.Name.Name)
						return false
					}
					if _, dup := fsFlags[fsName]; dup {
						inspectErr = fmt.Errorf("%s: duplicate FlagSet %q in %s", fset.Position(call.Pos()), fsName, fn.Name.Name)
						return false
					}
					trackedCalls[call] = true
					fsVars[lhs.Name] = fsName
					fsFlags[fsName] = nil
					continue
				}
				if ident, ok := call.Fun.(*ast.Ident); ok {
					if helper, isHelper := helpers[ident.Name]; isHelper {
						lhs, ok := node.Lhs[i].(*ast.Ident)
						if !ok || len(call.Args) < 1 {
							inspectErr = fmt.Errorf("%s: unrecognized FlagSet helper assignment %s in %s", fset.Position(node.Pos()), ident.Name, fn.Name.Name)
							return false
						}
						argVal, err := evalStringExpr(call.Args[0], consts, nil)
						if err != nil {
							inspectErr = fmt.Errorf("%s: non-constant name arg to %s in %s: %w", fset.Position(call.Pos()), ident.Name, fn.Name.Name, err)
							return false
						}
						fsName, err := evalStringExpr(helper.nameTemplate, consts, map[string]string{helper.paramName: argVal})
						if err != nil {
							inspectErr = fmt.Errorf("%s: cannot resolve helper FlagSet name in %s: %w", fset.Position(call.Pos()), fn.Name.Name, err)
							return false
						}
						if _, dup := out[fsName]; dup {
							inspectErr = fmt.Errorf("%s: duplicate FlagSet %q in %s", fset.Position(call.Pos()), fsName, fn.Name.Name)
							return false
						}
						trackedCalls[call] = true
						fsVars[lhs.Name] = fsName
						fsFlags[fsName] = append([]string(nil), helper.baseFlags...)
					}
				}
			}
		case *ast.CallExpr:
			if pkgMethod, ok := flagPkgCallName(node); ok && pkgMethod != "NewFlagSet" {
				inspectErr = fmt.Errorf("%s: disallowed package-level flag.%s call in %s", fset.Position(node.Pos()), pkgMethod, fn.Name.Name)
				return false
			}
			if isFlagPkgCall(node, "NewFlagSet") && !trackedCalls[node] {
				inspectErr = fmt.Errorf("%s: untracked flag.NewFlagSet call in %s", fset.Position(node.Pos()), fn.Name.Name)
				return false
			}
			if ident, ok := node.Fun.(*ast.Ident); ok {
				if _, isHelper := helpers[ident.Name]; isHelper && !trackedCalls[node] {
					inspectErr = fmt.Errorf("%s: untracked FlagSet helper %s call in %s", fset.Position(node.Pos()), ident.Name, fn.Name.Name)
					return false
				}
			}
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if recv, ok := sel.X.(*ast.Ident); ok {
					if fsName, isFS := fsVars[recv.Name]; isFS {
						flagName, registered, err := classifyFlagSetMethodCall(fset, fn.Name.Name, node, sel.Sel.Name, consts)
						if err != nil {
							inspectErr = err
							return false
						}
						if registered {
							fsFlags[fsName] = append(fsFlags[fsName], flagName)
						}
						return true
					}
				}
			}
			// Fail closed if a FlagSet variable is passed to an unrecognized function.
			calleeName := callExprName(node.Fun)
			for _, arg := range node.Args {
				if ident, ok := arg.(*ast.Ident); ok {
					if _, isFS := fsVars[ident.Name]; isFS {
						if !allowedFlagSetParseHelpers[calleeName] {
							inspectErr = fmt.Errorf("%s: FlagSet %s passed to unrecognized function %s in %s", fset.Position(node.Pos()), ident.Name, calleeName, fn.Name.Name)
							return false
						}
					}
				}
			}
		}
		return true
	})

	if inspectErr != nil {
		return inspectErr
	}
	for fsName, flags := range fsFlags {
		out[fsName] = uniqueSorted(flags)
	}
	return nil
}

func classifyFlagSetMethodCall(
	fset *token.FileSet,
	fnName string,
	call *ast.CallExpr,
	method string,
	consts map[string]string,
) (flagName string, registered bool, err error) {
	if flagVarMethods[method] {
		if len(call.Args) < 2 {
			return "", false, fmt.Errorf("%s: FlagSet.%s missing flag name argument in %s", fset.Position(call.Pos()), method, fnName)
		}
		name, err := evalStringExpr(call.Args[1], consts, nil)
		if err != nil || strings.TrimSpace(name) == "" {
			return "", false, fmt.Errorf("%s: dynamic or empty flag name in FlagSet.%s (%s): %v", fset.Position(call.Pos()), method, fnName, err)
		}
		return name, true, nil
	}
	if flagValueMethods[method] {
		if len(call.Args) < 1 {
			return "", false, fmt.Errorf("%s: FlagSet.%s missing flag name argument in %s", fset.Position(call.Pos()), method, fnName)
		}
		name, err := evalStringExpr(call.Args[0], consts, nil)
		if err != nil || strings.TrimSpace(name) == "" {
			return "", false, fmt.Errorf("%s: dynamic or empty flag name in FlagSet.%s (%s): %v", fset.Position(call.Pos()), method, fnName, err)
		}
		return name, true, nil
	}
	if flagRuntimeMethods[method] {
		return "", false, nil
	}
	return "", false, fmt.Errorf("%s: unrecognized FlagSet method %s in %s", fset.Position(call.Pos()), method, fnName)
}

func flagPkgCallName(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "flag" {
		return "", false
	}
	return sel.Sel.Name, true
}

func isFlagPkgCall(call *ast.CallExpr, method string) bool {
	name, ok := flagPkgCallName(call)
	return ok && name == method
}

func callExprName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return callExprName(e.X) + "." + e.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func extractMapLiteralKeys(fset *token.FileSet, expr ast.Expr, consts map[string]string) ([]string, error) {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("%s: topicHelp must be a composite literal", fset.Position(expr.Pos()))
	}
	var keys []string
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("%s: topicHelp entry is not key:value", fset.Position(elt.Pos()))
		}
		key, err := evalStringExpr(kv.Key, consts, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: non-constant topicHelp key: %w", fset.Position(kv.Pos()), err)
		}
		keys = append(keys, key)
	}
	return uniqueSorted(keys), nil
}

func extractSwitchCases(fset *token.FileSet, fn *ast.FuncDecl, wantTag string, consts map[string]string) ([]string, error) {
	var cases []string
	var found bool
	var inspectErr error

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		sw, ok := n.(*ast.SwitchStmt)
		if !ok || sw.Tag == nil {
			return true
		}
		if exprText(sw.Tag) != wantTag {
			return true
		}
		found = true
		for _, stmt := range sw.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}
			for _, expr := range cc.List {
				val, err := evalStringExpr(expr, consts, nil)
				if err != nil {
					inspectErr = fmt.Errorf("%s: non-constant switch case in %s (%s): %w", fset.Position(expr.Pos()), fn.Name.Name, wantTag, err)
					return false
				}
				cases = append(cases, val)
			}
		}
		return false
	})

	if inspectErr != nil {
		return nil, inspectErr
	}
	if !found {
		return nil, fmt.Errorf("%s: switch on %s not found in %s", fset.Position(fn.Pos()), wantTag, fn.Name.Name)
	}
	return uniqueSorted(cases), nil
}

func extractRunEnvHelpEarlyTopics(fset *token.FileSet, fn *ast.FuncDecl, consts map[string]string) ([]string, error) {
	var topics []string
	var inspectErr error

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		cc, ok := n.(*ast.CaseClause)
		if !ok || len(cc.List) != 1 {
			return true
		}
		val, err := evalStringExpr(cc.List[0], consts, nil)
		if err != nil || val != "help" {
			return true
		}
		for _, stmt := range cc.Body {
			ifStmt, ok := stmt.(*ast.IfStmt)
			if !ok {
				continue
			}
			// Look for comparisons of args[1] == "<topic>" inside early-return branches (skipping the error-check if).
			var hasReturn bool
			ast.Inspect(ifStmt.Body, func(bn ast.Node) bool {
				if _, ok := bn.(*ast.ReturnStmt); ok {
					hasReturn = true
				}
				return true
			})
			if !hasReturn {
				continue
			}
			ast.Inspect(ifStmt.Cond, func(cn ast.Node) bool {
				be, ok := cn.(*ast.BinaryExpr)
				if !ok || be.Op != token.EQL {
					return true
				}
				if exprText(be.X) == "args[1]" {
					topic, err := evalStringExpr(be.Y, consts, nil)
					if err != nil {
						inspectErr = fmt.Errorf("%s: non-constant help topic comparison in runEnv: %w", fset.Position(be.Pos()), err)
						return false
					}
					topics = append(topics, topic)
				}
				return true
			})
		}
		return false
	})

	if inspectErr != nil {
		return nil, inspectErr
	}
	return uniqueSorted(topics), nil
}

func extractUsageText(fset *token.FileSet, fn *ast.FuncDecl, consts map[string]string) (string, error) {
	var b strings.Builder
	var inspectErr error

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "fmt" {
			return true
		}
		switch sel.Sel.Name {
		case "Fprint":
			if len(call.Args) != 2 {
				inspectErr = fmt.Errorf("%s: unexpected fmt.Fprint arity in %s", fset.Position(call.Pos()), fn.Name.Name)
				return false
			}
			s, err := evalStringExpr(call.Args[1], consts, nil)
			if err != nil {
				inspectErr = fmt.Errorf("%s: non-constant usage string in %s: %w", fset.Position(call.Pos()), fn.Name.Name, err)
				return false
			}
			b.WriteString(s)
		case "Fprintln":
			if len(call.Args) != 2 {
				inspectErr = fmt.Errorf("%s: unexpected fmt.Fprintln arity in %s", fset.Position(call.Pos()), fn.Name.Name)
				return false
			}
			s, err := evalStringExpr(call.Args[1], consts, nil)
			if err != nil {
				inspectErr = fmt.Errorf("%s: non-constant usage string in %s: %w", fset.Position(call.Pos()), fn.Name.Name, err)
				return false
			}
			b.WriteString(s)
			b.WriteByte('\n')
		}
		return true
	})

	if inspectErr != nil {
		return "", inspectErr
	}
	text := b.String()
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%s: empty usage text in %s", fset.Position(fn.Pos()), fn.Name.Name)
	}
	return text, nil
}

func exprText(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		return exprText(e.X) + "[" + exprText(e.Index) + "]"
	case *ast.BasicLit:
		return e.Value
	case *ast.SelectorExpr:
		return exprText(e.X) + "." + e.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func uniqueSorted(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func TestDocumentationSupportedCoreSelectionExcludesExperimental(t *testing.T) {
	facts, err := extractCLISourceFactsFromDir(".")
	if err != nil {
		t.Fatalf("extractCLISourceFactsFromDir: %v", err)
	}
	for _, f := range facts.SelectedGoFiles {
		if f == "opencode.go" {
			t.Fatalf("selected supported-core files unexpectedly include experimental file %q", f)
		}
	}
	if !containsString(facts.SelectedGoFiles, "opencode_disabled.go") {
		t.Fatalf("selected supported-core files missing opencode_disabled.go: %v", facts.SelectedGoFiles)
	}
	if !containsString(facts.IgnoredGoFiles, "opencode.go") {
		t.Fatalf("ignored files missing experimental opencode.go: %v", facts.IgnoredGoFiles)
	}
}

func TestDocumentationDispatchInventoryParity(t *testing.T) {
	facts, err := extractCLISourceFactsFromDir(".")
	if err != nil {
		t.Fatalf("extractCLISourceFactsFromDir: %v", err)
	}

	wantTopLevel := []string{
		"apply",
		"backups",
		"fleet",
		"guide",
		"help",
		"logs",
		"mcp",
		"opencode",
		"opencode-harness",
		"preview",
		"restore",
		"review",
		"run",
		"task",
		"tui",
		"version",
	}
	assertStringSetEqual(t, "runEnv top-level commands", facts.TopLevelCommands, wantTopLevel)

	wantHelpTopics := []string{
		"apply",
		"backups",
		"fleet",
		"guide",
		"logs",
		"mcp",
		"opencode",
		"opencode-harness",
		"orchestrate",
		"preview",
		"restore",
		"review",
		"run",
		"task",
		"tui",
		"version",
	}
	assertStringSetEqual(t, "isHelpTopic topics", facts.HelpTopics, wantHelpTopics)

	// Every valid help topic must be owned by either topicHelp or runEnv early-return branches (disjoint union).
	seenTopic := make(map[string]bool, len(facts.TopicHelpKeys))
	for _, k := range facts.TopicHelpKeys {
		seenTopic[k] = true
	}
	for _, k := range facts.HelpEarlyTopics {
		if seenTopic[k] {
			t.Fatalf("help topic %q is duplicated across topicHelp and runEnv early branches", k)
		}
	}
	combinedTopics := append([]string{}, facts.TopicHelpKeys...)
	combinedTopics = append(combinedTopics, facts.HelpEarlyTopics...)
	assertStringSetEqual(t, "topicHelp + runEnv early topics", combinedTopics, wantHelpTopics)

	wantFamilies := map[string][]string{
		"runRestore":     {"apply", "list", "preview"},
		"runFleet":       {"-h", "--help", "apply", "doctor", "help", "plan", "restore", "status"},
		"runMCP":         {"-h", "--help", "apply", "doctor", "help", "preview", "restore", "status"},
		"runMCPRestore":  {"-h", "--help", "apply", "help", "list", "preview"},
		"runRun":         {"check", "clean", "help", "init", "record", "status"},
		"runTaskCommand": {"brief", "help"},
		"runReview":      {"help", "package"},
	}
	if len(facts.FamilySubcommands) != len(wantFamilies) {
		t.Fatalf("family dispatch count = %d, want %d; got keys %v", len(facts.FamilySubcommands), len(wantFamilies), mapKeys(facts.FamilySubcommands))
	}
	for family, wantSubs := range wantFamilies {
		gotSubs, ok := facts.FamilySubcommands[family]
		if !ok {
			t.Fatalf("missing family dispatch %q", family)
		}
		assertStringSetEqual(t, family+" subcommands", gotSubs, wantSubs)
	}
}

func TestDocumentationFlagSetInventoryParity(t *testing.T) {
	facts, err := extractCLISourceFactsFromDir(".")
	if err != nil {
		t.Fatalf("extractCLISourceFactsFromDir: %v", err)
	}

	wantFlagSets := map[string][]string{
		"asdx-core":                     {"apply", "home", "opencode-free-first", "repo", "version", "with-opencode"},
		"asdx-core backups":             {"home", "repo"},
		"asdx-core restore":             {"apply", "home", "repo"},
		"asdx-core logs":                {"home"},
		"asdx-core guide":               {"home", "repo"},
		"asdx-core tui":                 {"fleet-config", "home", "opencode-free-first", "repo", "with-opencode"},
		"asdx-core fleet status":        {"config", "home", "repo"},
		"asdx-core fleet plan":          {"adopt", "config", "home", "output", "repo"},
		"asdx-core fleet apply":         {"home", "plan"},
		"asdx-core fleet restore":       {"backup", "home"},
		"asdx-core fleet doctor":        {"config", "opencode-bin"},
		"asdx-core mcp status":          {"config", "home", "json", "project"},
		"asdx-core mcp doctor":          {"codegraph-bin", "config", "connect", "home", "json", "network", "opencode-bin", "project"},
		"asdx-core mcp preview":         {"auth", "config", "home", "json", "servers"},
		"asdx-core mcp apply":           {"auth", "config", "expect-config", "home", "json", "opencode-bin", "servers"},
		"asdx-core mcp restore list":    {"config", "home", "json"},
		"asdx-core mcp restore preview": {"config", "home", "json"},
		"asdx-core mcp restore apply":   {"config", "expect-config", "home", "json"},
		"asdx-core run init":            {"batch", "direct", "feature", "format", "label", "repo", "run-root"},
		"asdx-core run status":          {"format", "repo", "run", "run-root"},
		"asdx-core run record":          {"format", "repo", "result", "run", "run-root", "task"},
		"asdx-core run clean":           {"format", "repo", "run", "run-root", "yes"},
		"asdx-core run check":           {"format", "max-output-bytes", "repo", "run", "run-root", "task", "timeout"},
		"asdx-core task brief":          {"force", "format", "repo", "run", "run-root", "task"},
		"asdx-core review package":      {"base", "format", "head", "max-file-bytes", "max-total-bytes", "repo", "run", "run-root"},
	}

	if len(facts.FlagSets) != len(wantFlagSets) {
		t.Fatalf("FlagSet count = %d, want %d; got %v", len(facts.FlagSets), len(wantFlagSets), mapKeys(facts.FlagSets))
	}
	for name, wantFlags := range wantFlagSets {
		gotFlags, ok := facts.FlagSets[name]
		if !ok {
			t.Fatalf("missing FlagSet %q", name)
		}
		assertStringSetEqual(t, "FlagSet "+name, gotFlags, wantFlags)
	}
}

func TestDocumentationHelpTextParityAndKnownExceptions(t *testing.T) {
	facts, err := extractCLISourceFactsFromDir(".")
	if err != nil {
		t.Fatalf("extractCLISourceFactsFromDir: %v", err)
	}

	globalUsage, ok := facts.UsageTexts["printUsage"]
	if !ok {
		t.Fatalf("missing printUsage text")
	}

	// Supported top-level commands must be documented in printUsage; experimental commands must not be.
	for _, cmd := range facts.TopLevelCommands {
		switch cmd {
		case "opencode", "opencode-harness":
			if strings.Contains(strings.ToLower(globalUsage), cmd) {
				t.Fatalf("printUsage unexpectedly advertises experimental command %q", cmd)
			}
		default:
			if !strings.Contains(globalUsage, cmd) {
				t.Fatalf("printUsage missing supported command %q", cmd)
			}
		}
	}

	// Family usage texts must document every non-help subcommand in their dispatch switch.
	familyUsageExpectations := []struct {
		dispatchFn string
		usageFn    string
	}{
		{"runRestore", "printUsage"},
		{"runFleet", "printFleetUsage"},
		{"runMCP", "printMCPUsage"},
		{"runMCPRestore", "printMCPRestoreUsage"},
		{"runRun", "printRunUsage"},
		{"runTaskCommand", "printTaskUsage"},
		{"runReview", "printReviewUsage"},
	}
	for _, check := range familyUsageExpectations {
		subs := facts.FamilySubcommands[check.dispatchFn]
		usage := facts.UsageTexts[check.usageFn]
		for _, sub := range subs {
			if sub == "-h" || sub == "--help" || sub == "help" {
				continue
			}
			if !strings.Contains(usage, sub) {
				t.Fatalf("%s missing subcommand %q dispatched by %s", check.usageFn, sub, check.dispatchFn)
			}
		}
	}

	// Curated FlagSet -> owning usage function parity.
	flagSetUsageOwner := map[string]string{
		"asdx-core":                     "printUsage",
		"asdx-core backups":             "printUsage",
		"asdx-core restore":             "printUsage",
		"asdx-core logs":                "printUsage",
		"asdx-core guide":               "printUsage",
		"asdx-core tui":                 "printTUIUsage",
		"asdx-core fleet status":        "printFleetStatusUsage",
		"asdx-core fleet plan":          "printFleetPlanUsage",
		"asdx-core fleet apply":         "printFleetApplyUsage",
		"asdx-core fleet restore":       "printFleetRestoreUsage",
		"asdx-core fleet doctor":        "printFleetDoctorUsage",
		"asdx-core mcp status":          "printMCPStatusUsage",
		"asdx-core mcp doctor":          "printMCPDoctorUsage",
		"asdx-core mcp preview":         "printMCPPreviewUsage",
		"asdx-core mcp apply":           "printMCPApplyUsage",
		"asdx-core mcp restore list":    "printMCPRestoreUsage",
		"asdx-core mcp restore preview": "printMCPRestoreUsage",
		"asdx-core mcp restore apply":   "printMCPRestoreUsage",
		"asdx-core run init":            "printRunUsage",
		"asdx-core run status":          "printRunUsage",
		"asdx-core run record":          "printRunUsage",
		"asdx-core run clean":           "printRunUsage",
		"asdx-core run check":           "printRunUsage",
		"asdx-core task brief":          "printTaskUsage",
		"asdx-core review package":      "printReviewUsage",
	}

	// Curated known source/help exceptions: flags registered on a FlagSet whose
	// documentation lives elsewhere (or is an intentional/known omission).
	curatedOmittedFlags := map[string]map[string]string{
		"asdx-core": {
			"with-opencode":       "documented in printOrchestrateUsage, omitted from global printUsage",
			"opencode-free-first": "documented in printOrchestrateUsage, omitted from global printUsage",
		},
	}

	for fsName, flags := range facts.FlagSets {
		owner, ok := flagSetUsageOwner[fsName]
		if !ok {
			t.Fatalf("FlagSet %q has no usage-owner mapping", fsName)
		}
		usageText, ok := facts.UsageTexts[owner]
		if !ok {
			t.Fatalf("usage function %q not extracted", owner)
		}
		for _, fl := range flags {
			if _, excepted := curatedOmittedFlags[fsName][fl]; excepted {
				if strings.Contains(usageText, "--"+fl) {
					t.Fatalf("curated exception stale: %s now mentions --%s in %s", fsName, fl, owner)
				}
				continue
			}
			if !strings.Contains(usageText, "--"+fl) {
				t.Fatalf("usage function %s for %s does not document flag --%s", owner, fsName, fl)
			}
		}
	}

	// Verify printOrchestrateUsage documents the excepted top-level orchestrate flags.
	orchUsage := facts.UsageTexts["printOrchestrateUsage"]
	for _, fl := range []string{"with-opencode", "opencode-free-first"} {
		if !strings.Contains(orchUsage, "--"+fl) {
			t.Fatalf("printOrchestrateUsage missing --%s", fl)
		}
	}

	// Verify documented Spec 020 source/help inconsistencies honestly rather than masking them:
	// 1. P-01: global printUsage describes mcp as read-only and omits mcp apply/restore.
	if strings.Contains(globalUsage, "asdx-core mcp apply") || strings.Contains(globalUsage, "asdx-core mcp restore") {
		t.Fatalf("global printUsage unexpectedly documents mcp apply/restore (P-01 may have been resolved)")
	}
	mcpUsage := facts.UsageTexts["printMCPUsage"]
	for _, sub := range []string{"status", "doctor", "preview", "apply", "restore", "help"} {
		if !strings.Contains(mcpUsage, sub) {
			t.Fatalf("printMCPUsage missing subcommand %q", sub)
		}
	}

	// 2. printUsage help topic synopsis omits guide, version, orchestrate even though isHelpTopic accepts them.
	for _, omittedTopic := range []string{"guide", "version", "orchestrate"} {
		if !containsString(facts.HelpTopics, omittedTopic) {
			t.Fatalf("isHelpTopic unexpectedly lost %q", omittedTopic)
		}
	}
	if strings.Contains(globalUsage, "asdx-core help [preview|apply|backups|logs|restore|tui|fleet|mcp|run|task|review]") != true {
		t.Fatalf("printUsage help synopsis line changed unexpectedly:\n%s", globalUsage)
	}

	// 3. mcp restore list registers --config via newMCPWriteFlagSet, but printMCPRestoreUsage synopsis omits [--config PATH] on the list line.
	mcpRestoreUsage := facts.UsageTexts["printMCPRestoreUsage"]
	if !strings.Contains(mcpRestoreUsage, "asdx-core mcp restore list [--home PATH] [--json]") {
		t.Fatalf("printMCPRestoreUsage restore list line changed unexpectedly:\n%s", mcpRestoreUsage)
	}

	// 4. run, task, review families dispatch "help" only, not "--help" / "-h" (unlike fleet and mcp).
	for _, fam := range []string{"runRun", "runTaskCommand", "runReview"} {
		subs := facts.FamilySubcommands[fam]
		if containsString(subs, "--help") || containsString(subs, "-h") {
			t.Fatalf("%s unexpectedly added --help/-h dispatch: %v", fam, subs)
		}
	}

	// Every topicHelp key must correspond to a supported top-level command.
	for _, topic := range facts.TopicHelpKeys {
		if !containsString(facts.TopLevelCommands, topic) {
			t.Fatalf("topicHelp key %q is not a top-level command", topic)
		}
	}

	// Default build printOpenCodeUsage must only state unavailability.
	openCodeUsage := facts.UsageTexts["printOpenCodeUsage"]
	if !strings.Contains(openCodeUsage, "unavailable in this build") || !strings.Contains(openCodeUsage, "agentic_sdd_experimental") {
		t.Fatalf("printOpenCodeUsage unexpected text:\n%s", openCodeUsage)
	}

	// 5. trace.go commandName omits fleet, mcp, run, task, review.
	wantTrace := []string{"apply", "backups", "guide", "help", "logs", "opencode", "opencode-harness", "preview", "restore", "tui", "version"}
	assertStringSetEqual(t, "trace.go commandName cases", facts.TraceCommands, wantTrace)
}

func TestDocumentationASTExtractorFailsClosedOnUnrecognizedPatterns(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "dynamic flag registration in loop",
			src: `package main
import "flag"
func runDynamic(args []string) {
	fs := flag.NewFlagSet("asdx-core dynamic", flag.ContinueOnError)
	for _, name := range []string{"a", "b"} {
		fs.String(name, "", "usage")
	}
}`,
			wantErr: "dynamic or empty flag name",
		},
		{
			name: "top-level package flag registration",
			src: `package main
import "flag"
func init() {
	_ = flag.String("sneaky", "", "usage")
}`,
			wantErr: "disallowed package-level flag.String",
		},
		{
			name: "unrecognized helper receiving FlagSet",
			src: `package main
import "flag"
func runHelper(args []string) {
	fs := flag.NewFlagSet("asdx-core helper", flag.ContinueOnError)
	addExtraFlags(fs)
}`,
			wantErr: "passed to unrecognized function",
		},
		{
			name: "non-literal FlagSet name",
			src: `package main
import "flag"
func runNonLiteral(name string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	_ = fs
}`,
			wantErr: "non-constant FlagSet name",
		},
		{
			name: "non-literal switch case in dispatch",
			src: `package main
func runEnv(args []string) int {
	dyn := "x"
	switch args[0] {
	case dyn:
		return 0
	}
	return 0
}`,
			wantErr: "non-constant switch case",
		},
		{
			name: "unrecognized FlagSet method",
			src: `package main
import "flag"
func runUnknownMethod() {
	fs := flag.NewFlagSet("asdx-core custom", flag.ContinueOnError)
	fs.CustomRegister("x")
}`,
			wantErr: "unrecognized FlagSet method",
		},
		{
			name: "untracked inline NewFlagSet call",
			src: `package main
import "flag"
func runInline() {
	_ = parseSomething(flag.NewFlagSet("asdx-core inline", flag.ContinueOnError))
}`,
			wantErr: "untracked flag.NewFlagSet call",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := extractCLISourceFactsFromSource("synthetic.go", tc.src)
			if err == nil {
				t.Fatalf("expected fail-closed error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func assertStringSetEqual(t *testing.T, label string, got, want []string) {
	t.Helper()
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if !reflect.DeepEqual(gotSorted, wantSorted) {
		t.Fatalf("%s mismatch:\n  got:  %v\n  want: %v", label, gotSorted, wantSorted)
	}
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
