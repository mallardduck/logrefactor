package collector

import (
	"encoding/csv"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LogEntry represents a single log statement with all its arguments for structured logging migration
type LogEntry struct {
	ID              string
	FilePath        string
	Line            int
	Column          int
	Package         string
	OriginalCall    string   // e.g., "log.Printf"
	LogLevel        string   // e.g., "Info", "Error", "Debug" (extracted if possible)
	MessageTemplate string   // The format string or message
	Arguments       []Argument
	NewCall         string   // To be filled: new logging function call
	NewMessage      string   // To be filled: improved message
	StructuredFields string  // To be filled: JSON or comma-separated field mappings
	Notes           string
}

// Argument represents a single argument passed to the log function
type Argument struct {
	Index       int    // Position in argument list (0-based)
	Expression  string // The actual Go expression (e.g., "user.Name", "err")
	VarName     string // Simplified variable name for field key
	Type        string // Inferred type if possible
	FormatVerb  string // Associated format verb (%s, %v, %d, etc.) if applicable
	SuggestedKey string // Suggested field name for structured logging
}

// Collect scans the specified path for log entries and exports them to CSV
func Collect(rootPath, outputFile, pattern string) error {
	logPattern, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	var entries []LogEntry
	entryID := 1

	// Walk through the directory tree
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse the file
		fileEntries, err := parseFile(path, logPattern, &entryID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		entries = append(entries, fileEntries...)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Export to CSV
	return exportToCSV(entries, outputFile)
}

// parseFile parses a single Go file and extracts log entries with full argument details
func parseFile(filePath string, logPattern *regexp.Regexp, entryID *int) ([]LogEntry, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	packageName := node.Name.Name

	// Walk the AST
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Get the function selector
		funcName := getFunctionName(call)
		if funcName == "" || !logPattern.MatchString(funcName) {
			return true
		}

		// Extract position information
		pos := fset.Position(call.Pos())

		// Extract log level from function name if possible
		logLevel := extractLogLevel(funcName)

		// Extract message and all arguments
		messageTemplate, arguments := extractLogDetails(call, fset)

		entry := LogEntry{
			ID:              fmt.Sprintf("LOG-%04d", *entryID),
			FilePath:        filePath,
			Line:            pos.Line,
			Column:          pos.Column,
			Package:         packageName,
			OriginalCall:    funcName,
			LogLevel:        logLevel,
			MessageTemplate: messageTemplate,
			Arguments:       arguments,
			NewCall:         "", // To be filled by user
			NewMessage:      "", // To be filled by user
			StructuredFields: "", // To be filled by user
			Notes:           "",
		}

		entries = append(entries, entry)
		(*entryID)++

		return true
	})

	return entries, nil
}

// getFunctionName extracts the function name from a call expression
func getFunctionName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if ident, ok := fun.X.(*ast.Ident); ok {
			return ident.Name + "." + fun.Sel.Name
		}
		// Handle chained calls like logger.WithField().Info()
		return formatExpr(fun)
	case *ast.Ident:
		return fun.Name
	}
	return ""
}

// extractLogLevel tries to extract the log level from the function name
func extractLogLevel(funcName string) string {
	funcLower := strings.ToLower(funcName)
	
	levels := []string{"trace", "debug", "info", "warn", "warning", "error", "fatal", "panic"}
	for _, level := range levels {
		if strings.Contains(funcLower, level) {
			// Capitalize first letter
			return strings.ToUpper(level[:1]) + level[1:]
		}
	}
	
	// Check for Print variants
	if strings.Contains(funcLower, "print") {
		return "Info"
	}
	
	return "Unknown"
}

// extractLogDetails extracts the message template and all arguments with metadata
func extractLogDetails(call *ast.CallExpr, fset *token.FileSet) (string, []Argument) {
	if len(call.Args) == 0 {
		return "", nil
	}

	var messageTemplate string
	var arguments []Argument
	var formatVerbs []string

	// First argument is usually the message or format string
	firstArg := call.Args[0]
	if lit, ok := firstArg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		messageTemplate = lit.Value
		// Extract format verbs from the template
		formatVerbs = extractFormatVerbs(messageTemplate)
	} else {
		// If first arg is not a string literal, it might be a variable
		messageTemplate = formatExpr(firstArg)
	}

	// Process remaining arguments
	for i := 1; i < len(call.Args); i++ {
		arg := call.Args[i]
		expr := formatExpr(arg)
		varName := extractVarName(expr)
		inferredType := inferType(arg)
		
		// Match with format verb if available
		formatVerb := ""
		if i-1 < len(formatVerbs) {
			formatVerb = formatVerbs[i-1]
		}

		// Suggest a field key name
		suggestedKey := generateFieldKey(varName, formatVerb, inferredType)

		arguments = append(arguments, Argument{
			Index:        i - 1,
			Expression:   expr,
			VarName:      varName,
			Type:         inferredType,
			FormatVerb:   formatVerb,
			SuggestedKey: suggestedKey,
		})
	}

	return messageTemplate, arguments
}

// extractFormatVerbs extracts format verbs (%s, %v, %d, etc.) from a format string
func extractFormatVerbs(formatStr string) []string {
	// Remove quotes
	cleanStr := strings.Trim(formatStr, `"'`+"`")
	
	// Regex to match format verbs
	re := regexp.MustCompile(`%[-+# 0]*[\d]*\.?[\d]*[vTtbcdoqxXUeEfFgGsp]`)
	matches := re.FindAllString(cleanStr, -1)
	
	return matches
}

// formatExpr converts an expression to a string representation
func formatExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return formatExpr(e.X) + "." + e.Sel.Name
	case *ast.CallExpr:
		// For function calls, just show function name
		return formatExpr(e.Fun) + "()"
	case *ast.IndexExpr:
		// For array/slice access
		return formatExpr(e.X) + "[...]"
	case *ast.UnaryExpr:
		return e.Op.String() + formatExpr(e.X)
	case *ast.BinaryExpr:
		return formatExpr(e.X) + " " + e.Op.String() + " " + formatExpr(e.Y)
	default:
		// For complex expressions, use type name
		return fmt.Sprintf("<%T>", expr)
	}
}

// extractVarName extracts a simple variable name from an expression
func extractVarName(expr string) string {
	// For "user.Name" -> "Name"
	// For "err" -> "err"
	// For "count" -> "count"
	
	parts := strings.Split(expr, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return expr
}

// inferType tries to infer the type from the expression
func inferType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float"
		case token.STRING:
			return "string"
		case token.CHAR:
			return "rune"
		}
	case *ast.Ident:
		name := e.Name
		if name == "true" || name == "false" {
			return "bool"
		}
		if name == "nil" {
			return "nil"
		}
		// Check common variable name patterns
		if strings.HasPrefix(name, "is") || strings.HasPrefix(name, "has") {
			return "bool"
		}
		if name == "err" || strings.HasSuffix(name, "Error") {
			return "error"
		}
	case *ast.CallExpr:
		return "func_result"
	}
	return "unknown"
}

// generateFieldKey generates a suggested field key name for structured logging
func generateFieldKey(varName, formatVerb, inferredType string) string {
	// Convert to snake_case and lowercase
	key := toSnakeCase(varName)
	
	// Remove common prefixes
	key = strings.TrimPrefix(key, "p_")
	key = strings.TrimPrefix(key, "m_")
	
	// Special handling for common names
	switch key {
	case "err", "error":
		return "error"
	case "msg", "message":
		return "message"
	case "id":
		return "id"
	case "name":
		return "name"
	case "status", "state":
		return "status"
	}
	
	return key
}

// toSnakeCase converts camelCase or PascalCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// exportToCSV writes the log entries to a CSV file with enhanced columns
func exportToCSV(entries []LogEntry, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header with enhanced columns
	header := []string{
		"ID",
		"FilePath",
		"Line",
		"Column",
		"Package",
		"OriginalCall",
		"LogLevel",
		"MessageTemplate",
		"ArgumentCount",
		"ArgumentDetails",
		"NewCall",
		"NewMessage",
		"StructuredFields",
		"Notes",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write entries
	for _, entry := range entries {
		// Format argument details as a readable string
		argDetails := formatArgumentDetails(entry.Arguments)
		
		row := []string{
			entry.ID,
			entry.FilePath,
			strconv.Itoa(entry.Line),
			strconv.Itoa(entry.Column),
			entry.Package,
			entry.OriginalCall,
			entry.LogLevel,
			entry.MessageTemplate,
			strconv.Itoa(len(entry.Arguments)),
			argDetails,
			entry.NewCall,
			entry.NewMessage,
			entry.StructuredFields,
			entry.Notes,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// formatArgumentDetails formats the arguments into a readable string for CSV
func formatArgumentDetails(args []Argument) string {
	if len(args) == 0 {
		return ""
	}

	var parts []string
	for _, arg := range args {
		detail := fmt.Sprintf("%s(%s)=%s",
			arg.SuggestedKey,
			arg.Type,
			arg.Expression,
		)
		if arg.FormatVerb != "" {
			detail += fmt.Sprintf("[%s]", arg.FormatVerb)
		}
		parts = append(parts, detail)
	}
	
	return strings.Join(parts, "; ")
}
