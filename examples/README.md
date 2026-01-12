package main

import (
	"fmt"
	"log"
	"time"
)

type User struct {
	ID   string
	Name string
	Email string
}

func main() {
	log.Printf("application starting on port %d", 8080)
	
	user := User{ID: "user123", Name: "Alice"}
	if err := processUser(user); err != nil {
		log.Printf("error processing user %s: %v", user.ID, err)
	}
}

type User struct {
	ID   string
	Name string
	Age  int
}

func handleRequest(requestID string, userID string, duration int) {
	log.Printf("processing request %s for user %s", requestID, userID)
	
	// Simulate processing
	if duration > 1000 {
		log.Printf("slow request: %s took %dms", requestID, duration)
	}
	
	log.Printf("request %s completed in %dms", requestID, duration)
}

func connectDatabase(host string, port int) error {
	log.Printf("connecting to database at %s:%d", host, port)
	
	// Simulate connection
	err := fmt.Errorf("connection refused")
	if err != nil {
		log.Printf("database connection failed: %v", err)
		return err
	}
	
	log.Println("database connection established")
	return nil
}

func processUser(userID string, age int) error {
	log.Printf("processing user %s with age %d", userID, age)
	
	// Business logic here
	if age < 18 {
		log.Printf("user %s is underage", userID)
		return fmt.Errorf("user too young")
	}
	
	log.Println("user processed successfully")
	return nil
}
```

**Key Migration Points:**
1. `log.Printf` with format strings and variables
2. Multiple arguments of different types
3. Error handling logs
4. Business logic logging

## Running the Example

### 1. Collect Logs
```bash
cd examples/before
../../logrefactor collect -path . -output logs.csv
```

### 2. Edit CSV

Fill in `NewMessage` and `StructuredFields` columns.

### 3. Transform to slog

```bash
./logrefactor transform -input logs.csv -path ./examples/before -config templates/slog.json -dry-run
```

This will show what changes would be made. If satisfied:

```bash
cp -r examples/before examples/after-slog
./logrefactor transform -input logs.csv -path examples/after-slog -config templates/slog.json
```

### 4. Transform to zap

```bash
./logrefactor transform -input logs.csv -config templates/zap.json -dry-run
```

## What You've Learned

✅ How to identify variables in log statements  
✅ How to map variables to structured fields  
✅ How to choose field names  
✅ How to use different logging library templates  
✅ How to verify migrations

## Next Steps

1. **Try on your project**: `./logrefactor collect -path ./yourproject`
2. **Review MIGRATION_GUIDE.md** for detailed strategies
3. **Check TEMPLATES.md** for customization options
4. **Start with one package** as proof of concept

---

Ready to modernize your logging! 🚀
