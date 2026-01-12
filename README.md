# LogRefactor - Structured Logging Migration Tool

A powerful AST-based tool for migrating Go codebases from unstructured to structured logging. Works with **any logging library** - completely customizable and library-agnostic.

## 🎯 What This Tool Does

Migrates your logging from unstructured formats like:
```go
log.Printf("error processing user %s: %v", username, err)
```

To structured logging like:
```go
// slog
log.Error("error processing user", 
    slog.String("username", username),
    slog.Any("error", err))

// zap
logger.Error("error processing user",
    zap.String("username", username),
    zap.Error("error", err))

// zerolog
log.Error().
    Str("username", username).
    Err(err).
    Msg("error processing user")
```

## Key Features

- ✅ **Library-Agnostic**: Customizable templates for any logging library
- ✅ **Smart Variable Extraction**: Automatically identifies variables and suggests field names  
- ✅ **Format Verb Analysis**: Matches variables with `%s`, `%v`, `%d` format verbs
- ✅ **Multiple Output Formats**: Built-in support for slog, zap, zerolog, logrus + custom
- ✅ **Safe Migration**: Dry-run mode and CSV workflow for review
- ✅ **Incremental**: Process specific packages or entire projects

## Quick Start

```bash
# Build
go build -o logrefactor

# 1. Collect log statements
./logrefactor collect -path ./myproject -output logs.csv

# 2. Edit logs.csv - fill in NewMessage and StructuredFields columns

# 3. Transform (preview)
./logrefactor transform -input logs.csv -config templates/slog.json -dry-run

# 4. Apply changes
./logrefactor transform -input logs.csv -config templates/slog.json
```

## Complete Example

**Original code:**
```go
log.Printf("processing user %s with age %d", username, age)
log.Printf("error for user %s: %v", username, err)
```

**After running collect, edit the CSV:**
```csv
NewMessage,StructuredFields
Processing user,"username=username, age=age"
Failed to process user,"username=username, error=err"
```

**Transformed code (slog):**
```go
logger.Info("Processing user", 
    slog.String("username", username),
    slog.Int("age", age))
logger.Error("Failed to process user",
    slog.String("username", username),
    slog.Any("error", err))
```

## CSV Schema

The collector generates a CSV with these columns:

| Column | You Fill | Description |
|--------|----------|-------------|
| MessageTemplate | - | Original format string |
| ArgumentDetails | - | Extracted variables with types |
| **NewMessage** | ✏️ | Improved message (no format verbs) |
| **StructuredFields** | ✏️ | Field mappings: `key=expr, key2=expr2` or JSON |
| NewCall | ✏️ (optional) | Target logging function |

## Supported Logging Libraries

Built-in templates for popular libraries:

```bash
# slog (Go standard library)
./logrefactor transform -config templates/slog.json

# zap
./logrefactor transform -config templates/zap.json

# zerolog
./logrefactor transform -config templates/zerolog.json

# logrus
./logrefactor transform -config templates/logrus.json

# Custom (your own format)
./logrefactor transform -config my-template.json
```

## Custom Templates

Create `my-template.json`:
```json
{
  "style": "custom",
  "loggerVar": "myLogger",
  "template": "{{.Logger}}.{{.Level}}(\"{{.Message}}\"{{range .Fields}}, \"{{.Key}}\", {{.Expression}}{{end}})"
}
```

This gives you complete control over the output format.

## Command Reference

### collect
```bash
./logrefactor collect -path ./myproject -output logs.csv -pattern "log\\.|logger\\."
```

- `-path` - Directory to scan
- `-output` - CSV filename
- `-pattern` - Regex to match log calls

### transform
```bash
./logrefactor transform -input logs.csv -path ./myproject -config templates/slog.json -dry-run
```

- `-input` - CSV with your edits
- `-path` - Directory to transform
- `-config` - Template config file
- `-dry-run` - Preview without applying

## Migration Strategies

### Package-by-Package
```bash
./logrefactor collect -path ./pkg/api -output api.csv
# Edit api.csv
./logrefactor transform -input api.csv -path ./pkg/api -config templates/slog.json
```

### By Log Level
```bash
# Errors first
./logrefactor collect -pattern "log\\.(Error|Fatal)" -output errors.csv
```

## ArgumentDetails Format

Shows what variables were found:

- `username(unknown)=user.Name[%s]` - String from struct
- `error(error)=err[%v]` - Error variable  
- `count(int)=len(items)[%d]` - Function result

Use this to understand what's available for structured fields.

## Best Practices

1. **Version control first**: `git commit` before starting
2. **Start small**: Test on one package
3. **Use dry-run**: Always preview changes
4. **Consistent naming**: Establish field name conventions (snake_case)
5. **Message guidelines**: Remove variables, keep messages constant

## Troubleshooting

**No entries collected?**
- Check pattern matches your logs
- Try `-pattern "log"` (broader)

**Transform not working?**
- Fill in NewMessage or NewCall columns
- Verify StructuredFields format
- Check file paths haven't changed

**Wrong output format?**
- Verify template config
- Check loggerVar matches your code
- Use custom template for full control

## Examples

See `/examples` directory for complete examples showing migration to different logging libraries.

## Documentation

- **README.md** (this file) - Overview and quick reference
- **MIGRATION_GUIDE.md** - Detailed migration strategies
- **TEMPLATES.md** - Template system documentation

## License

MIT

---

**Migrate your logging with confidence • Any library • Production-ready**
