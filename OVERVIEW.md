# LogRefactor - Structured Logging Migration Tool

## What You Have

A complete, production-ready tool for migrating Go codebases from unstructured to structured logging. This is the **enhanced version** that supports migrating to ANY logging library through customizable templates.

## Key Enhancements

### 1. Variable Extraction & Field Mapping
The collector now **extracts all variables** from log statements and suggests structured field names:

```go
// Before
log.Printf("error processing user %s: %v", username, err)

// Collector identifies:
// - username (string) -> suggested field: "username"
// - err (error) -> suggested field: "error"
// - Format verbs: %s, %v
```

### 2. Library-Agnostic Templates
Built-in support for popular libraries + custom templates for anything else:

- **slog** (Go stdlib)
- **zap** (uber-go/zap)
- **zerolog** (rs/zerolog)
- **logrus** (sirupsen/logrus)
- **custom** (your own format)

### 3. Smart CSV Format
Enhanced CSV includes:
- **ArgumentDetails**: Shows all variables found with types
- **StructuredFields**: Where you map variables to structured fields
- **NewMessage**: Improved message without format verbs

## Complete Workflow

### 1. Collect & Analyze
```bash
./logrefactor collect -path ./myproject -output logs.csv
```

**CSV Generated:**
```csv
MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
"error: %v","error(error)=err[%v]",,
"user %s login from %s","name(unknown)=username[%s]; ip(unknown)=ipAddr[%s]",,
```

### 2. Fill in CSV
Edit the CSV to specify new format:

| MessageTemplate | ArgumentDetails | **NewMessage** | **StructuredFields** |
|----------------|-----------------|----------------|---------------------|
| `"error: %v"` | `error(error)=err[%v]` | `Failed to process request` | `error=err` |
| `"user %s login from %s"` | `name(unknown)=username[%s]; ip(unknown)=ipAddr[%s]` | `User logged in` | `username=username, ip_address=ipAddr` |

### 3. Choose Your Target Library
```bash
# slog (Go standard library)
./logrefactor transform -input logs.csv -config templates/slog.json

# zap
./logrefactor transform -input logs.csv -config templates/zap.json

# zerolog
./logrefactor transform -input logs.csv -config templates/zerolog.json

# Custom (your format)
./logrefactor transform -input logs.csv -config my-template.json
```

### 4. Result

**Original:**
```go
log.Printf("error processing user %s: %v", username, err)
```

**After (slog):**
```go
logger.Error("Failed to process user",
    slog.String("username", username),
    slog.Any("error", err))
```

**After (zap):**
```go
logger.Error("Failed to process user",
    zap.String("username", username),
    zap.Error("error", err))
```

**After (zerolog):**
```go
log.Error().
    Str("username", username).
    Err(err).
    Msg("Failed to process user")
```

## Project Structure

```
logrefactor/
├── main.go                      # CLI entry point
├── go.mod
│
├── internal/
│   ├── collector/
│   │   └── collector.go         # Enhanced AST scanning with variable extraction
│   └── transformer/
│       └── transformer.go       # Template-based code generation
│
├── templates/                   # Logging library templates
│   ├── slog.json
│   ├── zap.json
│   ├── zerolog.json
│   ├── logrus.json
│   └── custom.json
│
├── examples/
│   ├── before/                  # Unstructured logging example
│   └── README.md
│
└── docs/
    ├── README.md                # Main documentation
    ├── MIGRATION_GUIDE.md       # Detailed migration strategies
    └── TEMPLATES.md             # Template system documentation
```

## What Makes This Different

### Before (Simple Text Replacement Tool)
❌ Only rewrites log messages  
❌ Doesn't understand variables  
❌ Fixed output format  
❌ Manual field mapping

### Now (Structured Logging Migration Tool)
✅ Extracts and analyzes variables  
✅ Suggests field names automatically  
✅ Any logging library via templates  
✅ Semi-automated field mapping  
✅ Type-aware transformations

## Real Example

### Original Code
```go
func handleRequest(requestID string, userID string, duration int) {
    log.Printf("processing request %s for user %s", requestID, userID)
    
    if duration > 1000 {
        log.Printf("slow request: %s took %dms", requestID, duration)
    }
    
    log.Printf("request %s completed in %dms", requestID, duration)
}
```

### Step 1: Collect

```bash
./logrefactor collect -path . -output logs.csv
```

### Step 2: CSV Output

```csv
ID,OriginalCall,MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
LOG-0001,log.Printf,"processing request %s for user %s","request_id(unknown)=requestID[%s]; user_id(unknown)=userID[%s]",,
LOG-0002,log.Printf,"slow request: %s took %dms","request_id(unknown)=requestID[%s]; duration_ms(int)=duration[%d]",,
LOG-0003,log.Printf,"request %s completed in %dms","request_id(unknown)=requestID[%s]; duration_ms(int)=duration[%d]",,
```

### Step 3: Edit CSV

```csv
ID,NewMessage,StructuredFields
LOG-0001,Processing request,request_id=requestID, user_id=userID
LOG-0002,Slow request detected,request_id=requestID, duration_ms=duration
LOG-0003,Request completed,request_id=requestID, duration_ms=duration
```

### Step 4: Transform to slog

```bash
./logrefactor transform -input logs.csv -config templates/slog.json
```

### Result

```go
func handleRequest(requestID string, userID string, duration int) {
    logger.Info("Processing request",
        slog.String("request_id", requestID),
        slog.String("user_id", userID))
    
    if duration > 1000 {
        logger.Warn("Slow request detected",
            slog.String("request_id", requestID),
            slog.Int("duration_ms", duration))
    }
    
    logger.Info("Request completed",
        slog.String("request_id", requestID),
        slog.Int("duration_ms", duration))
}
```

## Template System

### Using Built-in Templates

```bash
# slog style: logger.Info("msg", slog.String("key", val))
./logrefactor transform -config templates/slog.json

# zap style: logger.Info("msg", zap.String("key", val))
./logrefactor transform -config templates/zap.json

# zerolog style: log.Info().Str("key", val).Msg("msg")
./logrefactor transform -config templates/zerolog.json

# logrus style: log.WithFields(log.Fields{"key": val}).Info("msg")
./logrefactor transform -config templates/logrus.json
```

### Creating Custom Templates

For any logging library, create a JSON template:

```json
{
  "style": "custom",
  "loggerVar": "myLogger",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}, \"{{.Key}}\", {{.Expression}}{{end}})"
}
```

**Template Variables:**
- `{{.Logger}}` - Logger variable name
- `{{.Level}}` - Log level (Info, Error, etc.)
- `{{.Message}}` - The log message
- `{{.Fields}}` - Array of fields with `.Key` and `.Expression`

**Example Output:**
```go
myLogger.Error("Failed to connect", "host", hostname, "port", port, "error", err)
```

## Command Reference

### collect
```bash
./logrefactor collect [options]

Options:
  -path string       Directory to scan (default: ".")
  -output string     CSV filename (default: "log_entries.csv")
  -pattern string    Regex for log calls (default: "log\\.|logger\\.")
```

### transform
```bash
./logrefactor transform [options]

Options:
  -input string      CSV with your edits (default: "log_entries.csv")
  -path string       Directory to transform (default: ".")
  -config string     Template config file (required)
  -dry-run          Preview without applying (default: false)
```

## StructuredFields Format

Two formats supported:

**Simple (recommended):**
```
error=err, user_id=userID, count=len(items)
```

**JSON (for complex cases):**
```json
[
  {"key": "error", "expression": "err", "type": "error"},
  {"key": "user_id", "expression": "userID", "type": "string"}
]
```

## Migration Strategies

### 1. Package-by-Package
```bash
./logrefactor collect -path ./pkg/api -output api.csv
# Edit api.csv
./logrefactor transform -input api.csv -path ./pkg/api -config templates/slog.json
```

### 2. By Log Level
```bash
# Critical logs first
./logrefactor collect -pattern "log\\.(Error|Fatal)" -output errors.csv
```

### 3. Incremental
Process 10-20 log statements at a time, review, apply, test.

## Documentation

- **README.md** - Overview and quick reference (this file)
- **MIGRATION_GUIDE.md** - Detailed migration strategies and patterns
- **TEMPLATES.md** - Complete template system documentation

## Quick Start

```bash
# 1. Build
go build -o logrefactor

# 2. Try on example
cd examples/before
../../logrefactor collect -path . -output logs.csv

# 3. Edit logs.csv (fill NewMessage and StructuredFields)

# 4. Transform
../../logrefactor transform -input logs.csv -config ../../templates/slog.json -dry-run
../../logrefactor transform -input logs.csv -config ../../templates/slog.json

# 5. Review
cat main.go
```

## FAQ

**Q: Do I have to use the suggested field names?**
No, edit StructuredFields to use any names you want.

**Q: Can I process multiple packages with different templates?**
Yes! Each package can use a different template config.

**Q: What if my logging library isn't supported?**
Create a custom template - works with ANY library.

**Q: How does it handle format verbs like %s, %v?**
The collector identifies them and matches them to variables. You just specify which variables to use as fields.

**Q: Can I use this for partial migration?**
Yes! Leave some rows empty in CSV to skip them.

## Benefits

✅ **Faster debugging**: Query logs by specific fields  
✅ **Better observability**: Feed structured logs to monitoring tools  
✅ **Easier alerting**: Alert on specific field values  
✅ **Consistent logging**: Standardize across your codebase  
✅ **Future-proof**: Easy to change logging libraries later  

## License

MIT

---

**Transform your logging with confidence • Any library • Production-ready**
