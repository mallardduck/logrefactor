.PHONY: build clean install test example help

# Build the tool
build:
	@echo "Building logrefactor..."
	@go build -o logrefactor ./main.go
	@echo "✓ Built successfully: ./logrefactor"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f logrefactor
	@rm -f *.csv
	@echo "✓ Cleaned"

# Install to GOPATH/bin
install:
	@echo "Installing logrefactor..."
	@go install
	@echo "✓ Installed to $(shell go env GOPATH)/bin/logrefactor"

# Run tests (if you add them later)
test:
	@echo "Running tests..."
	@go test ./...

# Run example workflow
example: build
	@echo "Running example workflow..."
	@echo "\n==> Step 1: Collecting log entries from example project"
	@./logrefactor collect -path ./example -output example_logs.csv
	@echo "\n==> Generated CSV preview:"
	@head -n 10 example_logs.csv
	@echo "\n==> To continue:"
	@echo "1. Edit example_logs.csv and fill in the NewText column"
	@echo "2. Run: ./logrefactor transform -input example_logs.csv -path ./example -dry-run"
	@echo "3. If satisfied, run: ./logrefactor transform -input example_logs.csv -path ./example"

# Show help
help:
	@echo "LogRefactor - Go AST-based Log Message Refactoring Tool"
	@echo ""
	@echo "Targets:"
	@echo "  make build     - Build the logrefactor binary"
	@echo "  make clean     - Remove build artifacts and generated files"
	@echo "  make install   - Install to GOPATH/bin"
	@echo "  make test      - Run tests"
	@echo "  make example   - Run example workflow"
	@echo "  make help      - Show this help message"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. make build"
	@echo "  2. ./logrefactor collect -path ./your-project -output logs.csv"
	@echo "  3. Edit logs.csv (fill in NewText column)"
	@echo "  4. ./logrefactor transform -input logs.csv -path ./your-project -dry-run"
	@echo "  5. ./logrefactor transform -input logs.csv -path ./your-project"
	@echo ""
	@echo "See README.md and QUICKSTART.md for detailed documentation"

# Default target
.DEFAULT_GOAL := help
