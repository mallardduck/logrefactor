# Auto-Mapping Feature

## Overview

The transformer can **automatically generate field mappings** from the `ArgumentDetails` column when `StructuredFields` is left empty. This makes migration faster by using the collector's suggestions as defaults.

## How It Works

### Step 1: Collector Analyzes Variables

When you run:
```bash
./logrefactor collect -path ./myproject -output logs.csv
```

The collector analyzes each log statement and populates `ArgumentDetails`:

```csv
ArgumentDetails: "error(error)=err[%v]; username(unknown)=user.Name[%s]; count(int)=len(items)[%d]"
```

**Format:** `suggestedKey(type)=expression[formatVerb]`

### Step 2: Leave StructuredFields Empty

In the CSV, just fill in `NewMessage` and leave `StructuredFields` empty:

```csv
ID,MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
LOG-0001,"error: %v","error(error)=err[%v]",Failed to process request,
```

### Step 3: Transform with Auto-Mapping (default)

```bash
./logrefactor transform -input logs.csv -config templates/slog.json
```

The transformer **automatically** uses the field mappings from `ArgumentDetails`:
- `error(error)=err[%v]` → `error=err`

**Generated code:**
```go
logger.Error("Failed to process request", slog.Any("error", err))
```

## Examples

### Example 1: Simple Error Logging

**Original:**
```go
log.Printf("error: %v", err)
```

**CSV (only NewMessage filled):**
```csv
MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
"error: %v","error(error)=err[%v]",Failed to process request,
```

**Auto-mapped result:**
```go
logger.Error("Failed to process request", slog.Any("error", err))
```

### Example 2: Multiple Variables

**Original:**
```go
log.Printf("user %s login from %s", username, ipAddr)
```

**CSV (only NewMessage filled):**
```csv
MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
"user %s login from %s","username(unknown)=username[%s]; ip_address(unknown)=ipAddr[%s]",User logged in,
```

**Auto-mapped result:**
```go
logger.Info("User logged in",
    slog.String("username", username),
    slog.String("ip_address", ipAddr))
```

### Example 3: Mixed Types

**Original:**
```go
log.Printf("request %s took %dms", requestID, duration)
```

**CSV (only NewMessage filled):**
```csv
MessageTemplate,ArgumentDetails,NewMessage,StructuredFields
"request %s took %dms","request_id(unknown)=requestID[%s]; duration_ms(int)=duration[%d]",Request completed,
```

**Auto-mapped result:**
```go
logger.Info("Request completed",
    slog.String("request_id", requestID),
    slog.Int("duration_ms", duration))
```

## When to Override

You can still provide custom `StructuredFields` to override the auto-mapping:

### Scenario 1: Rename Fields

**Auto-mapping would use:** `user_id=userID`

**Override to use better name:**
```csv
StructuredFields: "user_identifier=userID"
```

### Scenario 2: Skip Some Fields

**Auto-mapping would include all 3 fields**

**Override to only use some:**
```csv
StructuredFields: "error=err"  (skips the other fields)
```

### Scenario 3: Add Computed Fields

**Override to add computed values:**
```csv
StructuredFields: "error=err, item_count=len(items), total_bytes=len(data)"
```

## Disabling Auto-Mapping

If you want to explicitly control all field mappings:

```bash
./logrefactor transform -input logs.csv -config templates/slog.json -auto-map=false
```

With `-auto-map=false`:
- Empty `StructuredFields` = no fields generated
- You must explicitly fill in `StructuredFields` for every row

## Comparison

### With Auto-Mapping (default, `-auto-map=true`)

**CSV:**
```csv
NewMessage,StructuredFields
Failed to connect,
```

**Result:**
```go
logger.Error("Failed to connect",
    slog.String("host", hostname),
    slog.Int("port", port),
    slog.Any("error", err))
```

### Without Auto-Mapping (`-auto-map=false`)

**CSV:**
```csv
NewMessage,StructuredFields
Failed to connect,
```

**Result:**
```go
logger.Error("Failed to connect")  // No fields!
```

To get fields, you must fill in `StructuredFields`:
```csv
NewMessage,StructuredFields
Failed to connect,"host=hostname, port=port, error=err"
```

## Best Practices

### 1. Start with Auto-Mapping

For initial migration, use auto-mapping (default) to speed up the process:
```bash
./logrefactor collect -path ./myproject -output logs.csv
# Just fill in NewMessage column
./logrefactor transform -input logs.csv -config templates/slog.json
```

### 2. Review Generated Fields

After transformation, review the generated code:
```bash
git diff
```

Check that field names make sense for your codebase.

### 3. Override When Needed

For logs where auto-mapping doesn't produce ideal results, fill in `StructuredFields`:

```csv
ID,NewMessage,StructuredFields
LOG-0001,User authenticated,  (auto-mapped is fine, leave empty)
LOG-0002,Database query failed,"db_host=host, db_name=database, query_duration_ms=duration, error=err"
```

### 4. Establish Naming Conventions

The collector suggests field names based on variable names. Establish conventions:
- Use `user_id` not `uid` or `userId`
- Use `error` for all error fields
- Use `_ms` suffix for milliseconds: `duration_ms`
- Use `_bytes` suffix for byte counts: `size_bytes`

### 5. Re-run Collector After Refactoring

If you improve variable names in your code:
```go
// Before
id := "user123"
log.Printf("processing: %s", id)

// After (better variable name)
userID := "user123"
log.Printf("processing: %s", userID)
```

Re-run collector to get better auto-generated field names:
```bash
./logrefactor collect -path ./myproject -output logs.csv
```

## Advanced: Editing ArgumentDetails

The collector generates `ArgumentDetails` with suggested field names. You can edit this column to change what gets auto-mapped:

**Original ArgumentDetails:**
```
"id(unknown)=id[%s]; n(int)=count[%d]"
```

**Edited ArgumentDetails (better names):**
```
"user_id(unknown)=id[%s]; item_count(int)=count[%d]"
```

Now auto-mapping will use `user_id` and `item_count` as field names.

## FAQ

**Q: Can I use auto-mapping for some rows and manual mapping for others?**

Yes! Leave `StructuredFields` empty for rows you want auto-mapped, fill it in for rows you want to control.

**Q: Does auto-mapping work with all template styles?**

Yes! Auto-mapping generates `FieldMapping` structures that work with slog, zap, zerolog, logrus, and custom templates.

**Q: What if ArgumentDetails has too many fields?**

Override with `StructuredFields` to specify only the fields you want:
```csv
StructuredFields: "error=err, request_id=reqID"  (skips other fields)
```

**Q: Can I change the suggested field names?**

Yes, two ways:
1. Edit `StructuredFields` column
2. Edit `ArgumentDetails` column (changes the suggestions)

**Q: What happens if ArgumentDetails is empty?**

If both `StructuredFields` and `ArgumentDetails` are empty, only the message is generated with no fields.

## Summary

**Auto-mapping (default):**
✅ Faster migration - just edit NewMessage  
✅ Uses collector's analysis automatically  
✅ Can override by filling StructuredFields  
✅ Enabled by default

**Manual mapping (`-auto-map=false`):**
✅ Explicit control  
✅ Must fill StructuredFields for every row  
❌ More work

**Recommendation:** Use auto-mapping (default) for speed, override with `StructuredFields` when needed.
