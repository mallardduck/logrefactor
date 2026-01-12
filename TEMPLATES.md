# Template System Documentation

Guide to using and creating custom templates for LogRefactor.

## Overview

Templates define how your structured log calls will be generated. LogRefactor includes built-in templates for popular libraries and supports custom templates for any format.

## Built-in Templates

### slog (Go standard library)

**File:** `templates/slog.json`
```json
{
  "style": "slog",
  "loggerVar": "log"
}
```

**Output Format:**
```go
log.info("message", slog.String("key", value))
```

### zap (uber-go/zap)

**File:** `templates/zap.json`
```json
{
  "style": "zap",
  "loggerVar": "logger"
}
```

**Output Format:**
```go
logger.Info("message", zap.String("key", value))
```

### zerolog (rs/zerolog)

**File:** `templates/zerolog.json`
```json
{
  "style": "zerolog",
  "loggerVar": "log"
}
```

**Output Format:**
```go
log.Info().Str("key", value).Msg("message")
```

### logrus (sirupsen/logrus)

**File:** `templates/logrus.json`
```json
{
  "style": "logrus",
  "loggerVar": "log"
}
```

**Output Format:**
```go
log.WithFields(log.Fields{"key": value}).Info("message")
```

## Template Configuration

### Configuration Schema

```json
{
  "style": "slog|zap|zerolog|logrus|custom",
  "loggerVar": "name_of_logger_variable",
  "template": "custom_template_string"
}
```

**Fields:**
- `style` (required): Template style to use
- `loggerVar` (required): Name of logger variable in your code
- `template` (required for custom): Custom template string

## Custom Templates

### Basic Custom Template

```json
{
  "style": "custom",
  "loggerVar": "myLogger",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}, \"{{.Key}}\", {{.Expression}}{{end}})"
}
```

**Output:**
```go
myLogger.Error("Failed to connect", "host", hostname, "port", port)
```

### Template Variables

Available in all custom templates:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `{{.Logger}}` | string | Logger variable name | `log` |
| `{{.Level}}` | string | Log level (capitalized) | `Error` |
| `{{.Message}}` | string | Log message | `Failed to connect` |
| `{{.Fields}}` | []Field | Array of fields | See below |

**Field object:**
- `{{.Key}}` - Field key name
- `{{.Expression}}` - Go expression for value
- `{{.Type}}` - Inferred type (string, int, error, etc.)

### Template Examples

#### Example 1: Simple Key-Value Format

**Template:**
```json
{
  "style": "custom",
  "loggerVar": "logger",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}, \"{{.Key}}\", {{.Expression}}{{end}})"
}
```

**Input:**
```csv
NewMessage: Database connection failed
StructuredFields: host=hostname, port=port, error=err
```

**Output:**
```go
logger.Error("Database connection failed", "host", hostname, "port", port, "error", err)
```

#### Example 2: Map-Based Format

**Template:**
```json
{
  "style": "custom",
  "loggerVar": "log",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\", map[string]interface{}{  {{range $i, $f := .Fields}}{{if $i}}, {{end}}\"{{$f.Key}}\": {{$f.Expression}}{{end}}})"
}
```

**Output:**
```go
log.Error("Database connection failed", map[string]interface{}{"host": hostname, "port": port, "error": err})
```

#### Example 3: Context-Based Format

**Template:**
```json
{
  "style": "custom",
  "loggerVar": "logger",
  "template": "{{.Logger}}.{{.Level}}(ctx, \"{{.Message}}\"{{range .Fields}}, {{.Expression}}{{end}})"
}
```

**Output:**
```go
logger.Error(ctx, "Database connection failed", hostname, port, err)
```

#### Example 4: With Field Names in Arguments

**Template:**
```json
{
  "style": "custom",
  "loggerVar": "log",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}, Field(\"{{.Key}}\", {{.Expression}}){{end}})"
}
```

**Output:**
```go
log.Error("Database connection failed", Field("host", hostname), Field("port", port), Field("error", err))
```

## Advanced Template Techniques

### Conditional Fields

```json
{
  "style": "custom",
  "loggerVar": "logger",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{if .Fields}}, map[string]interface{}{  {{range $i, $f := .Fields}}{{if $i}}, {{end}}\"{{$f.Key}}\": {{$f.Expression}}{{end}}}{{end}})"
}
```

This only includes the map if there are fields.

### Type-Aware Fields

While the template engine provides the type, most uses won't need it:

```json
{
  "style": "custom",
  "loggerVar": "logger",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}{{if eq .Type \"error\"}}, Err({{.Expression}}){{else}}, Any(\"{{.Key}}\", {{.Expression}}){{end}}{{end}})"
}
```

This uses `Err()` for errors and `Any()` for everything else.

### Level Mapping

If your library uses different level names:

```json
{
  "style": "custom",
  "loggerVar": "log",
  "template": "{{.Logger}}.{{if eq .Level \"Error\"}}ERROR{{else if eq .Level \"Info\"}}INFO{{else}}{{.Level}}{{end}}(\"{{.Message}}\"{{range .Fields}}, \"{{.Key}}\", {{.Expression}}{{end}})"
}
```

## Logger Variable Names

The `loggerVar` field specifies what your logger variable is named in the code.

**Common patterns:**
- `log` - for standard library or singleton loggers
- `logger` - for injected loggers
- `l` - for short-named loggers
- `zap` or `zerolog` - when using package name directly

**Example:**

If your code uses:
```go
var logger *zap.Logger
logger.Info("message")
```

Your template should have:
```json
{
  "loggerVar": "logger"
}
```

## Testing Your Template

### Step 1: Create Test CSV

Create a simple test CSV:
```csv
ID,FilePath,Line,Column,Package,OriginalCall,LogLevel,MessageTemplate,ArgumentCount,ArgumentDetails,NewCall,NewMessage,StructuredFields,Notes
TEST-01,test.go,1,1,test,log.Printf,Error,"error: %v",1,"error(error)=err[%v]",,Connection failed,error=err,
```

### Step 2: Test Transform

```bash
./logrefactor transform -input test.csv -config my-template.json -dry-run
```

### Step 3: Verify Output

Check that the generated code looks correct.

## Common Pitfalls

### Pitfall 1: Missing Quotes

**Wrong:**
```json
{
  "template": "{{.Logger}}.{{.Level}}({{.Message}})"
}
```

**Right:**
```json
{
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\")"
}
```

Messages need quotes!

### Pitfall 2: Forgetting Commas

**Wrong:**
```json
{
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}\"{{.Key}}\" {{.Expression}}{{end}})"
}
```

**Right:**
```json
{
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}, \"{{.Key}}\", {{.Expression}}{{end}})"
}
```

Need commas between arguments!

### Pitfall 3: Wrong Logger Variable

If your template has `"loggerVar": "log"` but your code uses `logger`, the generated code won't compile.

**Solution:** Match your actual code's variable name.

## Real-World Template Examples

### Example: apex/log Style

```json
{
  "style": "custom",
  "loggerVar": "log",
  "template": "{{.Logger}}.{{if eq .Level \"Error\"}}Error{{else if eq .Level \"Info\"}}Info{{else if eq .Level \"Warn\"}}Warn{{else if eq .Level \"Debug\"}}Debug{{end}}(\"{{.Message}}\"){{range .Fields}}.WithField(\"{{.Key}}\", {{.Expression}}){{end}}"
}
```

### Example: Custom Key-Value Logger

```json
{
  "style": "custom",
  "loggerVar": "kv",
  "template": "{{.Logger}}.Log(\"{{.Level}}\", \"{{.Message}}\"{{range .Fields}}, \"{{.Key}}=\"+fmt.Sprint({{.Expression}}){{end}})"
}
```

### Example: JSON String Logger

```json
{
  "style": "custom",
  "loggerVar": "jsonlog",
  "template": "{{.Logger}}.Print(LogEntry{Level: \"{{.Level}}\", Message: \"{{.Message}}\"{{if .Fields}}, Fields: map[string]interface{}{  {{range $i, $f := .Fields}}{{if $i}}, {{end}}\"{{$f.Key}}\": {{$f.Expression}}{{end}}}{{end}}})"
}
```

## Template Best Practices

### 1. Keep It Simple

Start with simple templates and add complexity only if needed.

### 2. Match Your Style Guide

Ensure generated code matches your team's style guide:
- Function naming (PascalCase vs camelCase)
- Argument ordering
- Indentation

### 3. Test Incrementally

Test your template on a small subset of logs before applying broadly.

### 4. Document Your Template

Include comments in your template config:

```json
{
  "style": "custom",
  "loggerVar": "logger",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}, \"{{.Key}}\", {{.Expression}}{{end}})",
  "_comment": "Custom template for our internal logger, follows key-value pattern"
}
```

### 5. Version Control

Keep your template files in version control alongside your code.

## Debugging Templates

### Enable Verbose Output

Run with `-dry-run` to see what will be generated:

```bash
./logrefactor transform -input logs.csv -config my-template.json -dry-run
```

### Check for Syntax Errors

If transformation fails, check:
1. JSON is valid
2. Template string is properly escaped
3. Logger variable matches code
4. Field references are correct

### Common Error Messages

**"template: parse error"**
- Check template syntax
- Ensure `{{}}` are balanced
- Verify Go template syntax

**"unknown style"**
- Style must be one of: slog, zap, zerolog, logrus, custom
- Check spelling

## FAQ

**Q: Can I use multiple templates in one project?**

A: Yes! Process different packages with different templates:
```bash
./logrefactor transform -input api.csv -path ./api -config templates/slog.json
./logrefactor transform -input db.csv -path ./db -config templates/zap.json
```

**Q: How do I handle logger initialization?**

A: The template only handles the call sites. You'll need to manually update logger initialization:
```go
// Add to your code:
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

**Q: Can templates call functions?**

A: Yes, expressions in fields can be any valid Go expression:
```csv
StructuredFields: count=len(items), name=strings.ToLower(user.Name)
```

**Q: What if I need different templates for different log levels?**

A: Use conditional logic in your template based on `{{.Level}}`:
```json
{
  "template": "{{if eq .Level \"Error\"}}{{.Logger}}.ErrorWithStack{{else}}{{.Logger}}.{{.Level}}{{end}}(\"{{.Message}}\"...)"
}
```

**Q: Can I contribute new built-in templates?**

A: Yes! Submit a PR with your template in the `templates/` directory.

---

For more examples, see the `templates/` directory in the repository.
