# Structured Logging Migration Guide

Complete guide for migrating from unstructured to structured logging using LogRefactor.

## Table of Contents

1. [Why Structured Logging?](#why-structured-logging)
2. [Before You Start](#before-you-start)
3. [Complete Workflow](#complete-workflow)
4. [Library-Specific Guides](#library-specific-guides)
5. [Field Mapping Strategies](#field-mapping-strategies)
6. [Common Patterns](#common-patterns)
7. [Troubleshooting](#troubleshooting)

## Why Structured Logging?

### Unstructured
```go
log.Printf("user %s logged in from %s at %d", username, ipAddr, timestamp)
```
❌ Hard to search  
❌ Hard to aggregate  
❌ Hard to alert on specific fields

### Structured
```go
log.Info("user logged in",
    slog.String("username", username),
    slog.String("ip_address", ipAddr),
    slog.Int64("timestamp", timestamp))
```
✅ Easy to query by username  
✅ Easy to aggregate by IP  
✅ Easy to set alerts on patterns

## Before You Start

### 1. Choose Your Target Library

| Library | Best For | Style |
|---------|----------|-------|
| **slog** | New projects, stdlib | Simple key-value |
| **zap** | High performance | Key-value fields |
| **zerolog** | Low allocation | Method chaining |
| **logrus** | Existing logrus users | WithFields pattern |

### 2. Establish Field Naming Conventions

```
user_id      ✅ snake_case, descriptive
userId       ❌ camelCase  
uid          ❌ too abbreviated
```

**Standard fields to use consistently:**
- `error` - for errors
- `duration_ms` - for timing
- `request_id` - for request tracing
- `user_id` - for user identification
- `status_code` - for HTTP status

### 3. Set Up Version Control

```bash
git checkout -b migrate-structured-logging
git add -A && git commit -m "Before structured logging migration"
```

## Complete Workflow

### Step 1: Collect

```bash
./logrefactor collect -path ./myproject -output logs.csv
```

**Generated CSV includes:**
```csv
ID,MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
LOG-0001,"error: %v","error(error)=err[%v]",,
LOG-0002,"user %s login from %s","name(unknown)=username[%s]; ip(unknown)=ipAddr[%s]",,
```

### Step 2: Fill In CSV

For each row, decide:
1. **NewMessage**: The constant message (no variables)
2. **StructuredFields**: How to map variables to fields

**Example edits:**

| MessageTemplate | ArgumentDetails | NewMessage | StructuredFields |
|----------------|-----------------|------------|-----------------|
| `"error: %v"` | `error(error)=err[%v]` | `Failed to process request` | `error=err` |
| `"user %s login from %s"` | `name(unknown)=username[%s]; ip(unknown)=ipAddr[%s]` | `User logged in` | `username=username, ip_address=ipAddr` |

**StructuredFields Format Options:**

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

### Step 3: Choose Template

```bash
# Preview with slog
./logrefactor transform -input logs.csv -config templates/slog.json -dry-run
```

### Step 4: Review Output

The tool shows what will change:

```
handler.go:45:2
  Old: log.Printf("error: %v", err)
  New: logger.Error("Failed to process request", slog.Any("error", err))

handler.go:67:3
  Old: log.Printf("user %s login from %s", username, ipAddr)
  New: logger.Info("User logged in", slog.String("username", username), slog.String("ip_address", ipAddr))
```

### Step 5: Apply

```bash
# Apply changes
./logrefactor transform -input logs.csv -config templates/slog.json

# Verify
git diff
```

### Step 6: Test

```bash
go test ./...
go build ./...
```

### Step 7: Update Imports

Add required imports:

```go
import (
    "log/slog"  // for slog
    // or
    "go.uber.org/zap"  // for zap
    // or
    "github.com/rs/zerolog/log"  // for zerolog
)
```

## Library-Specific Guides

### Migrating to slog

**Before:**
```go
log.Printf("error processing %s: %v", userID, err)
```

**CSV Edit:**
```csv
NewMessage: Failed to process user
StructuredFields: user_id=userID, error=err
```

**After:**
```go
logger.Info("Failed to process user",
    slog.String("user_id", userID),
    slog.Any("error", err))
```

**Template:** `templates/slog.json`

**Common slog patterns:**
- `slog.String(key, val)` - strings
- `slog.Int(key, val)` - integers
- `slog.Any(key, val)` - any type (uses reflection)
- `slog.Group(name, attrs...)` - nested fields

### Migrating to zap

**Before:**
```go
log.Printf("request took %dms for user %s", duration, username)
```

**CSV Edit:**
```csv
NewMessage: Request completed
StructuredFields: duration_ms=duration, username=username
```

**After:**
```go
logger.Info("Request completed",
    zap.Int("duration_ms", duration),
    zap.String("username", username))
```

**Template:** `templates/zap.json`

**Common zap patterns:**
- `zap.String(key, val)`
- `zap.Int(key, val)`
- `zap.Error(val)` - special for errors
- `zap.Duration(key, val)` - time.Duration

### Migrating to zerolog

**Before:**
```go
log.Printf("cache miss for key %s, took %dms", cacheKey, duration)
```

**CSV Edit:**
```csv
NewMessage: Cache miss
StructuredFields: cache_key=cacheKey, duration_ms=duration
```

**After:**
```go
log.Info().
    Str("cache_key", cacheKey).
    Int("duration_ms", duration).
    Msg("Cache miss")
```

**Template:** `templates/zerolog.json`

**Common zerolog patterns:**
- `.Str(key, val)` - strings
- `.Int(key, val)` - integers
- `.Err(err)` - errors
- `.Msg(msg)` - final message

### Migrating to logrus

**Before:**
```go
log.Printf("user %s logged out after %d minutes", username, sessionTime)
```

**CSV Edit:**
```csv
NewMessage: User logged out
StructuredFields: username=username, session_minutes=sessionTime
```

**After:**
```go
log.WithFields(log.Fields{
    "username": username,
    "session_minutes": sessionTime,
}).Info("User logged out")
```

**Template:** `templates/logrus.json`

## Field Mapping Strategies

### Strategy 1: Direct Mapping

**When to use:** Variable name is good as field name

```go
// Before: log.Printf("user: %s", username)
// Fields: username=username
// After:  slog.String("username", username)
```

### Strategy 2: Renamed Mapping

**When to use:** Variable name needs improvement

```go
// Before: log.Printf("id: %s", id)
// Fields: user_id=id
// After:  slog.String("user_id", id)
```

### Strategy 3: Computed Mapping

**When to use:** Need to extract or compute value

```go
// Before: log.Printf("user: %s", user.Name)
// Fields: username=user.Name
// After:  slog.String("username", user.Name)

// Before: log.Printf("count: %d", len(items))
// Fields: item_count=len(items)
// After:  slog.Int("item_count", len(items))
```

### Strategy 4: Standard Error Handling

**Always use:**
```go
// Before: log.Printf("error: %v", err)
// Fields: error=err
// After:  slog.Any("error", err)  // or zap.Error("error", err)
```

## Common Patterns

### Pattern 1: Error Logging

**Before:**
```go
log.Printf("failed to connect to database: %v", err)
log.Printf("error processing request %s: %v", requestID, err)
```

**After (slog):**
```go
logger.Error("Failed to connect to database",
    slog.Any("error", err))
logger.Error("Failed to process request",
    slog.String("request_id", requestID),
    slog.Any("error", err))
```

### Pattern 2: HTTP Request Logging

**Before:**
```go
log.Printf("%s %s from %s took %dms", method, path, remoteAddr, duration)
```

**After (slog):**
```go
logger.Info("HTTP request",
    slog.String("method", method),
    slog.String("path", path),
    slog.String("remote_addr", remoteAddr),
    slog.Int("duration_ms", duration))
```

### Pattern 3: Database Operations

**Before:**
```go
log.Printf("query executed in %dms: %s", duration, query)
```

**After (slog):**
```go
logger.Debug("Database query executed",
    slog.Int("duration_ms", duration),
    slog.String("query", query))
```

### Pattern 4: Business Logic Events

**Before:**
```go
log.Printf("order %s created for user %s, total: $%.2f", orderID, userID, total)
```

**After (slog):**
```go
logger.Info("Order created",
    slog.String("order_id", orderID),
    slog.String("user_id", userID),
    slog.Float64("total_amount", total))
```

## Advanced Techniques

### Technique 1: Grouping Related Fields

**slog:**
```go
logger.Info("Request processed",
    slog.Group("request",
        slog.String("id", requestID),
        slog.String("method", method),
    ),
    slog.Group("response",
        slog.Int("status", statusCode),
        slog.Int("bytes", responseSize),
    ))
```

**zap:**
```go
logger.Info("Request processed",
    zap.Namespace("request"),
    zap.String("id", requestID),
    zap.String("method", method),
    zap.Namespace("response"),
    zap.Int("status", statusCode),
    zap.Int("bytes", responseSize))
```

### Technique 2: Context-Aware Logging

```go
// Create logger with context
logger := baseLogger.With(
    slog.String("service", "api"),
    slog.String("version", version),
)

// All subsequent logs include context
logger.Info("Server started")  // includes service and version
```

### Technique 3: Conditional Fields

```go
fields := []slog.Attr{}
if err != nil {
    fields = append(fields, slog.Any("error", err))
}
if requestID != "" {
    fields = append(fields, slog.String("request_id", requestID))
}
logger.LogAttrs(ctx, slog.LevelInfo, "Operation completed", fields...)
```

## Incremental Migration Patterns

### Pattern 1: By Package

```bash
# Week 1: Migrate API layer
./logrefactor collect -path ./api -output api.csv
# Edit and transform

# Week 2: Migrate database layer
./logrefactor collect -path ./db -output db.csv
# Edit and transform

# Week 3: Migrate business logic
./logrefactor collect -path ./services -output services.csv
# Edit and transform
```

### Pattern 2: By Log Level

```bash
# Phase 1: Critical logs (Error, Fatal)
./logrefactor collect -pattern "log\\.(Error|Fatal|Panic)" -output critical.csv

# Phase 2: Important logs (Warn, Info)
./logrefactor collect -pattern "log\\.(Warn|Info)" -output important.csv

# Phase 3: Debug logs
./logrefactor collect -pattern "log\\.(Debug|Print)" -output debug.csv
```

### Pattern 3: By Feature

```bash
# Authentication logs
./logrefactor collect -path ./auth -output auth.csv

# Payment logs
./logrefactor collect -path ./payment -output payment.csv

# etc.
```

## Verification Checklist

After migration:

- [ ] All tests pass: `go test ./...`
- [ ] Code compiles: `go build ./...`
- [ ] Logs still appear in output
- [ ] Log format matches expectations
- [ ] All structured fields present
- [ ] No format verbs (%s, %v) in messages
- [ ] Error fields use standard name ("error")
- [ ] Field names follow conventions
- [ ] No loss of information
- [ ] Performance acceptable

## Troubleshooting

### Issue: Fields not appearing

**Problem:** Logs show message but no fields

**Solution:** Check logger initialization:
```go
// Correct slog setup
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Correct zap setup
logger, _ := zap.NewProduction()
```

### Issue: Wrong field types

**Problem:** Numbers appearing as strings

**Solution:** Update StructuredFields:
```csv
# Wrong
user_age=age

# Right (with type hint in JSON format)
[{"key":"user_age","expression":"age","type":"int"}]
```

Or let the tool infer types - it will use `.Int()` for int variables.

### Issue: Complex expressions not working

**Problem:** `len(items)` shows as literal string

**Solution:** The tool preserves expressions:
```csv
StructuredFields: count=len(items)
# Produces: slog.Int("count", len(items))
```

### Issue: Some logs missed

**Problem:** Not all logs were collected

**Solution:** Check your pattern:
```bash
# Too specific
-pattern "log\\.Printf"

# Better
-pattern "log\\."

# Even better - all variations
-pattern "log\\.|logger\\."
```

## Performance Considerations

### slog
- Use `Logger.With()` for common fields
- Avoid `slog.Any()` for known types
- Use `LogAttrs()` for bulk attributes

### zap
- Use `Logger.With()` for context
- Prefer typed fields (`zap.String`, not `zap.Any`)
- Use `zap.NewProduction()` for performance

### zerolog
- Most efficient - minimal allocations
- Chain fields for best performance
- Use `Disabled()` to skip expensive ops

## Next Steps

1. **Choose your target library**
2. **Migrate one package as proof of concept**
3. **Review with team**
4. **Establish field naming standards**
5. **Migrate incrementally**
6. **Update monitoring/alerting to use structured fields**

---

**Questions?** See README.md or create an issue.
