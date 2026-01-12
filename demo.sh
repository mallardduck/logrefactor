#!/bin/bash
# Demo script showing the complete LogRefactor workflow

set -e

echo "======================================"
echo "LogRefactor Demo - Complete Workflow"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Build
echo -e "${BLUE}Step 1: Building the tool...${NC}"
go build -o logrefactor
echo -e "${GREEN}✓ Built successfully${NC}"
echo ""

# Step 2: Collect
echo -e "${BLUE}Step 2: Collecting log entries from example project...${NC}"
./logrefactor collect -path ./example -output demo_logs.csv
echo -e "${GREEN}✓ Collected log entries${NC}"
echo ""

# Show preview
echo -e "${BLUE}Step 3: Preview of collected entries:${NC}"
echo "----------------------------------------"
head -n 6 demo_logs.csv | column -t -s ','
echo "... (showing first 5 entries)"
echo ""

# Create a modified version with some improvements
echo -e "${BLUE}Step 4: Simulating CSV editing (normally you'd do this manually)...${NC}"
cat > demo_logs_edited.csv << 'EOF'
ID,FilePath,Line,Column,Package,FunctionCall,OriginalText,NewText,Arguments,Notes
LOG-0001,example/main.go,12,2,main,log.Println,"starting application",Application initialization started,,Improved for clarity
LOG-0002,example/main.go,17,3,main,log.Fatal,"db connection failed",Failed to establish database connection,,More descriptive
LOG-0003,example/main.go,21,2,main,log.Println,"database connected",Successfully connected to database,,Professional tone
LOG-0004,example/main.go,25,2,main,log.Printf,"server starting on port %d",HTTP server listening on port %d,8080,Consistent with others
LOG-0005,example/main.go,28,3,main,log.Fatal,"server failed to start",Failed to start HTTP server,,Better context
EOF

echo -e "${GREEN}✓ Created edited CSV with improvements${NC}"
echo ""

echo "Sample improvements:"
echo "  Before: \"starting application\""
echo "  After:  \"Application initialization started\""
echo ""

# Step 5: Dry run
echo -e "${BLUE}Step 5: Preview changes (dry run)...${NC}"
echo "----------------------------------------"
./logrefactor transform -input demo_logs_edited.csv -path ./example -dry-run
echo ""

# Step 6: Prompt for confirmation
echo -e "${YELLOW}Would you like to apply these changes? (yes/no)${NC}"
read -p "> " answer

if [ "$answer" = "yes" ] || [ "$answer" = "y" ]; then
    echo ""
    echo -e "${BLUE}Step 6: Applying changes...${NC}"
    ./logrefactor transform -input demo_logs_edited.csv -path ./example
    echo ""
    echo -e "${GREEN}✓ Changes applied successfully!${NC}"
    echo ""
    
    echo -e "${BLUE}Step 7: Viewing updated code:${NC}"
    echo "----------------------------------------"
    echo "Example: example/main.go (lines 10-15)"
    sed -n '10,15p' example/main.go
    echo "----------------------------------------"
    echo ""
    
    echo -e "${GREEN}Demo completed successfully!${NC}"
    echo ""
    echo "The log messages in example/ have been updated."
    echo "You can review the changes with: git diff example/"
else
    echo ""
    echo -e "${YELLOW}Skipped applying changes. Demo completed in dry-run mode.${NC}"
fi

echo ""
echo "======================================"
echo "Next Steps:"
echo "======================================"
echo "1. Try on your own project:"
echo "   ./logrefactor collect -path ./your-project"
echo ""
echo "2. Use AI to improve messages:"
echo "   python scripts/improve_logs.py your_logs.csv improved.csv"
echo ""
echo "3. Read the guides:"
echo "   - QUICKSTART.md for tutorial"
echo "   - README.md for full docs"
echo ""
echo "4. Clean up demo files:"
echo "   rm -f logrefactor demo_logs*.csv"
echo ""
