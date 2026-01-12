package transformer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// LogUpdate represents an update to apply
type LogUpdate struct {
	ID               string
	FilePath         string
	Line             int
	Column           int
	OriginalCall     string
	LogLevel         string
	MessageTemplate  string
	ArgumentDetails  string
	NewCall          string
	NewMessage       string
	StructuredFields string
}

// FieldMapping represents a structured logging field
type FieldMapping struct {
	Key        string `json:"key"`
	Expression string `json:"expression"`
	Type       string `json:"type"`
}

// TemplateConfig defines how to generate structured logging calls
type TemplateConfig struct {
	Style      string // "slog", "zap", "zerolog", "logrus", "custom"
	LoggerVar  string // Variable name for logger (e.g., "log", "logger")
	Template   string // Custom template if style is "custom"
}

// Transform reads the CSV and applies the transformations to the source files
func Transform(csvFile, rootPath string, dryRun bool, configFile string, autoMap bool) error {
	// Load template configuration
	config, err := loadTemplateConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load template config: %w", err)
	}

	updates, err := loadUpdates(csvFile)
	if err != nil {
		return fmt.Errorf("failed to load updates: %w", err)
	}

	// Group updates by file
	fileUpdates := make(map[string][]LogUpdate)
	for _, update := range updates {
		// Only process entries with NewMessage or NewCall
		if (update.NewMessage == "" && update.NewCall == "") ||
		   (update.NewMessage == update.MessageTemplate && update.NewCall == update.OriginalCall) {
			continue
		}
		fileUpdates[update.FilePath] = append(fileUpdates[update.FilePath], update)
	}

	if len(fileUpdates) == 0 {
		fmt.Println("No updates to apply")
		return nil
	}

	// Process each file
	for filePath, updates := range fileUpdates {
		if err := transformFile(filePath, updates, config, dryRun, autoMap); err != nil {
			return fmt.Errorf("failed to transform %s: %w", filePath, err)
		}
	}

	return nil
}

// loadTemplateConfig loads the template configuration
func loadTemplateConfig(configFile string) (*TemplateConfig, error) {
	if configFile == "" {
		// Default to slog style
		return &TemplateConfig{
			Style:     "slog",
			LoggerVar: "log",
		}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config TemplateConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// loadUpdates reads the CSV file and returns a list of updates
func loadUpdates(csvFile string) ([]LogUpdate, error) {
	file, err := os.Open(csvFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	var updates []LogUpdate

	// Skip header row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 13 {
			fmt.Fprintf(os.Stderr, "Warning: skipping malformed row %d\n", i+1)
			continue
		}

		line, _ := strconv.Atoi(record[2])
		column, _ := strconv.Atoi(record[3])

		update := LogUpdate{
			ID:               record[0],
			FilePath:         record[1],
			Line:             line,
			Column:           column,
			OriginalCall:     record[5],
			LogLevel:         record[6],
			MessageTemplate:  record[7],
			ArgumentDetails:  record[9],
			NewCall:          record[10],
			NewMessage:       record[11],
			StructuredFields: record[12],
		}

		updates = append(updates, update)
	}

	return updates, nil
}

// transformFile applies updates to a single file
func transformFile(filePath string, updates []LogUpdate, config *TemplateConfig, dryRun bool, autoMap bool) error {
	// Read the original file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return err
	}

	// Create a map of line:column -> update
	updateMap := make(map[string]LogUpdate)
	for _, update := range updates {
		key := fmt.Sprintf("%d:%d", update.Line, update.Column)
		updateMap[key] = update
	}

	// Track modifications
	var modifications []string
	modified := false

	// Walk the AST and apply replacements
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		startPos := fset.Position(call.Pos())
		key := fmt.Sprintf("%d:%d", startPos.Line, startPos.Column)

		update, exists := updateMap[key]
		if !exists {
			return true
		}

		// Generate the new log call
		newCode, err := generateStructuredLogCall(update, config, autoMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to generate code for %s: %v\n", update.ID, err)
			return true
		}

		// Record the modification
		modification := fmt.Sprintf("%s:%d:%d\n  Old: %s\n  New: %s",
			filepath.Base(filePath), startPos.Line, startPos.Column,
			truncateCode(formatCallExpr(call, fset), 80),
			truncateCode(newCode, 80))
		modifications = append(modifications, modification)

		if !dryRun {
			// Replace the call expression
			replaceCallExpr(call, newCode, fset, &content)
			modified = true
		}

		return true
	})

	// Print modifications
	for _, mod := range modifications {
		fmt.Println(mod)
		fmt.Println()
	}

	// Write back if modified and not dry run
	if modified && !dryRun {
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return err
		}
		fmt.Printf("Updated: %s (%d changes)\n", filePath, len(modifications))
	} else if len(modifications) > 0 && dryRun {
		fmt.Printf("Would update: %s (%d changes)\n", filePath, len(modifications))
	}

	return nil
}

// generateStructuredLogCall generates the new structured logging call based on template
func generateStructuredLogCall(update LogUpdate, config *TemplateConfig, autoMap bool) (string, error) {
	// Parse structured fields
	var fields []FieldMapping
	if update.StructuredFields != "" {
		if err := json.Unmarshal([]byte(update.StructuredFields), &fields); err != nil {
			// Try parsing as simple key=value format
			fields = parseSimpleFields(update.StructuredFields)
		}
	} else if autoMap && update.ArgumentDetails != "" {
		// Auto-generate field mappings from ArgumentDetails if StructuredFields is empty
		fields = autoGenerateFieldsFromArguments(update.ArgumentDetails)
	}

	// Use NewMessage if provided, otherwise use MessageTemplate
	message := update.NewMessage
	if message == "" {
		message = update.MessageTemplate
	}
	message = strings.Trim(message, `"'`+"`")

	// Generate based on style
	switch config.Style {
	case "slog":
		return generateSlogCall(config.LoggerVar, update.LogLevel, message, fields), nil
	case "zap":
		return generateZapCall(config.LoggerVar, update.LogLevel, message, fields), nil
	case "zerolog":
		return generateZerologCall(config.LoggerVar, update.LogLevel, message, fields), nil
	case "logrus":
		return generateLogrusCall(config.LoggerVar, update.LogLevel, message, fields), nil
	case "custom":
		return generateCustomCall(config.Template, config.LoggerVar, update.LogLevel, message, fields)
	default:
		return "", fmt.Errorf("unknown style: %s", config.Style)
	}
}

// generateSlogCall generates a slog-style structured log call
func generateSlogCall(loggerVar, level, message string, fields []FieldMapping) string {
	levelFunc := strings.ToLower(level)
	if levelFunc == "warning" {
		levelFunc = "warn"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf(`%s.%s("%s"`, loggerVar, levelFunc, message))

	for _, field := range fields {
		parts = append(parts, fmt.Sprintf(`slog.Any("%s", %s)`, field.Key, field.Expression))
	}

	return strings.Join(parts, ", ") + ")"
}

// generateZapCall generates a zap-style structured log call
func generateZapCall(loggerVar, level, message string, fields []FieldMapping) string {
	levelFunc := strings.Title(strings.ToLower(level))
	if levelFunc == "Warning" {
		levelFunc = "Warn"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf(`%s.%s("%s"`, loggerVar, levelFunc, message))

	for _, field := range fields {
		zapFunc := getZapFieldFunc(field.Type)
		parts = append(parts, fmt.Sprintf(`zap.%s("%s", %s)`, zapFunc, field.Key, field.Expression))
	}

	return strings.Join(parts, ", ") + ")"
}

// generateZerologCall generates a zerolog-style structured log call
func generateZerologCall(loggerVar, level, message string, fields []FieldMapping) string {
	levelFunc := strings.ToLower(level)
	if levelFunc == "warning" {
		levelFunc = "warn"
	}

	parts := []string{fmt.Sprintf("%s.%s()", loggerVar, levelFunc)}

	for _, field := range fields {
		zerologFunc := getZerologFieldFunc(field.Type)
		parts = append(parts, fmt.Sprintf(`%s("%s", %s)`, zerologFunc, field.Key, field.Expression))
	}

	parts = append(parts, fmt.Sprintf(`Msg("%s")`, message))

	return strings.Join(parts, ".")
}

// generateLogrusCall generates a logrus-style structured log call
func generateLogrusCall(loggerVar, level, message string, fields []FieldMapping) string {
	levelFunc := strings.Title(strings.ToLower(level))
	if levelFunc == "Warning" {
		levelFunc = "Warn"
	}

	if len(fields) == 0 {
		return fmt.Sprintf(`%s.%s("%s")`, loggerVar, levelFunc, message)
	}

	// Build fields map
	var fieldPairs []string
	for _, field := range fields {
		fieldPairs = append(fieldPairs, fmt.Sprintf(`"%s": %s`, field.Key, field.Expression))
	}

	return fmt.Sprintf(`%s.WithFields(%s.Fields{%s}).%s("%s")`,
		loggerVar, loggerVar, strings.Join(fieldPairs, ", "), levelFunc, message)
}

// generateCustomCall generates a custom template-based log call
func generateCustomCall(tmplStr, loggerVar, level, message string, fields []FieldMapping) (string, error) {
	tmpl, err := template.New("log").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"Logger":  loggerVar,
		"Level":   level,
		"Message": message,
		"Fields":  fields,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// parseSimpleFields parses simple key=value field format
func parseSimpleFields(fieldsStr string) []FieldMapping {
	var fields []FieldMapping
	
	// Split by semicolon or comma
	parts := strings.Split(fieldsStr, ";")
	if len(parts) == 1 {
		parts = strings.Split(fieldsStr, ",")
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse "key=expression" or "key:expression"
		var key, expr string
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key = strings.TrimSpace(kv[0])
			expr = strings.TrimSpace(kv[1])
		} else if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			key = strings.TrimSpace(kv[0])
			expr = strings.TrimSpace(kv[1])
		} else {
			// Assume it's just a variable name
			key = part
			expr = part
		}

		fields = append(fields, FieldMapping{
			Key:        key,
			Expression: expr,
			Type:       "unknown",
		})
	}

	return fields
}

// autoGenerateFieldsFromArguments parses ArgumentDetails and auto-generates field mappings
// ArgumentDetails format: "key(type)=expression[formatVerb]; key2(type2)=expression2[formatVerb2]"
// Example: "error(error)=err[%v]; username(unknown)=user.Name[%s]"
func autoGenerateFieldsFromArguments(argumentDetails string) []FieldMapping {
	var fields []FieldMapping
	
	// Split by semicolon
	parts := strings.Split(argumentDetails, ";")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Parse: "key(type)=expression[formatVerb]"
		// Example: "error(error)=err[%v]"
		
		// Find the key (everything before '(')
		openParen := strings.Index(part, "(")
		if openParen == -1 {
			continue
		}
		key := strings.TrimSpace(part[:openParen])
		
		// Find the type (between '(' and ')')
		closeParen := strings.Index(part, ")")
		if closeParen == -1 || closeParen <= openParen {
			continue
		}
		typ := strings.TrimSpace(part[openParen+1 : closeParen])
		
		// Find the expression (between '=' and '[' or end of string)
		equals := strings.Index(part, "=")
		if equals == -1 || equals <= closeParen {
			continue
		}
		
		// Extract expression (might have [formatVerb] at the end)
		exprPart := strings.TrimSpace(part[equals+1:])
		openBracket := strings.Index(exprPart, "[")
		
		var expr string
		if openBracket != -1 {
			expr = strings.TrimSpace(exprPart[:openBracket])
		} else {
			expr = exprPart
		}
		
		fields = append(fields, FieldMapping{
			Key:        key,
			Expression: expr,
			Type:       typ,
		})
	}
	
	return fields
}

// getZapFieldFunc returns the appropriate zap field function
func getZapFieldFunc(typ string) string {
	switch typ {
	case "string":
		return "String"
	case "int":
		return "Int"
	case "bool":
		return "Bool"
	case "error":
		return "Error"
	default:
		return "Any"
	}
}

// getZerologFieldFunc returns the appropriate zerolog field function
func getZerologFieldFunc(typ string) string {
	switch typ {
	case "string":
		return "Str"
	case "int":
		return "Int"
	case "bool":
		return "Bool"
	case "error":
		return "Err"
	default:
		return "Interface"
	}
}

// formatCallExpr formats a call expression back to code
func formatCallExpr(call *ast.CallExpr, fset *token.FileSet) string {
	var buf strings.Builder
	printer.Fprint(&buf, fset, call)
	return buf.String()
}

// replaceCallExpr replaces a call expression in the source code
func replaceCallExpr(call *ast.CallExpr, newCode string, fset *token.FileSet, content *[]byte) {
	// Get the position range of the call
	start := fset.Position(call.Pos())
	end := fset.Position(call.End())

	// Convert to bytes
	lines := strings.Split(string(*content), "\n")
	
	if start.Line > len(lines) || end.Line > len(lines) {
		return
	}

	// Simple line-based replacement
	if start.Line == end.Line {
		// Single line replacement
		line := lines[start.Line-1]
		before := line[:start.Column-1]
		after := line[end.Column-1:]
		lines[start.Line-1] = before + newCode + after
	} else {
		// Multi-line replacement
		firstLine := lines[start.Line-1]
		lastLine := lines[end.Line-1]
		
		before := firstLine[:start.Column-1]
		after := lastLine[end.Column-1:]
		
		// Replace the lines
		newLine := before + newCode + after
		lines = append(lines[:start.Line-1], append([]string{newLine}, lines[end.Line:]...)...)
	}

	*content = []byte(strings.Join(lines, "\n"))
}

// truncateCode truncates code to maxLen characters
func truncateCode(code string, maxLen int) string {
	// Remove extra whitespace
	code = strings.Join(strings.Fields(code), " ")
	
	if len(code) <= maxLen {
		return code
	}
	return code[:maxLen-3] + "..."
}
