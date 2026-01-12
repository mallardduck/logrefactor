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

# 2. Edit logs.csv - just fill in NewMessage column
#    (StructuredFields can be left empty - it will auto-map from ArgumentDetails!)

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

**After running collect, the CSV shows:**
```csv
ArgumentDetails: "username(unknown)=username[%s]; age(int)=age[%d]"
ArgumentDetails: "username(unknown)=username[%s]; error(error)=err[%v]"
```

**Edit CSV (only NewMessage - StructuredFields auto-maps!):**
```csv
NewMessage
Processing user
Failed to process user
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

✨ **No manual field mapping needed!** The tool used `ArgumentDetails` automatically.

## CSV Schema

The collector generates a CSV with these columns:

| Column | You Fill | Description |
|--------|----------|-------------|
| MessageTemplate | - | Original format string |
| ArgumentDetails | - | Extracted variables with types |
| **NewMessage** | ✏️ | Improved message (no format verbs) |
| **StructuredFields** | ✏️ (optional) | Field mappings: `key=expr, key2=expr2` or JSON |
| NewCall | ✏️ (optional) | Target logging function |

### 🚀 Auto-Mapping Feature

**NEW:** If you leave `StructuredFields` empty, the tool automatically generates field mappings from `ArgumentDetails`!

This means you only need to fill in `NewMessage` for a quick migration:

```csv
MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
"error: %v","error(error)=err[%v]",Failed to process,
```

The tool automatically maps: `error=err` → `slog.Any("error", err)`

See [AUTO_MAPPING.md](AUTO_MAPPING.md) for details.

**Override when needed:**
```csv
StructuredFields: "db_error=err, retry_count=retries"
```

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
- `-auto-map` - Auto-generate fields from ArgumentDetails when StructuredFields is empty (default: true)

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

**Migrate your logging with confidence • Any library • Maybe Production-ready (Or might eat your cat)**

_**Provided without warranty - Your mileage may vary**_