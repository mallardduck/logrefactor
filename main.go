package main

import (
	"flag"
	"fmt"
	"os"

	"logrefactor/internal/collector"
	"logrefactor/internal/transformer"
)

func main() {
	// Subcommands
	collectCmd := flag.NewFlagSet("collect", flag.ExitOnError)
	collectPath := collectCmd.String("path", ".", "Path to the Go project or package")
	collectOutput := collectCmd.String("output", "log_entries.csv", "Output CSV file")
	collectPattern := collectCmd.String("pattern", "log\\.|logrus\\.|logger\\.", "Regex pattern to match logging calls")

	transformCmd := flag.NewFlagSet("transform", flag.ExitOnError)
	transformInput := transformCmd.String("input", "log_entries.csv", "Input CSV file with updated entries")
	transformPath := transformCmd.String("path", ".", "Path to the Go project or package")
	transformDryRun := transformCmd.Bool("dry-run", false, "Show changes without applying them")
	transformConfig := transformCmd.String("config", "", "Template configuration file (JSON)")

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  logrefactor collect [options]   - Collect and index log entries")
		fmt.Println("  logrefactor transform [options] - Apply transformations from CSV")
		fmt.Println("\nExamples:")
		fmt.Println("  logrefactor collect -path ./mypackage -output logs.csv")
		fmt.Println("  logrefactor transform -input logs.csv -path ./mypackage")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "collect":
		collectCmd.Parse(os.Args[2:])
		if err := collector.Collect(*collectPath, *collectOutput, *collectPattern); err != nil {
			fmt.Fprintf(os.Stderr, "Error collecting log entries: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully collected log entries to %s\n", *collectOutput)

	case "transform":
		transformCmd.Parse(os.Args[2:])
		if err := transformer.Transform(*transformInput, *transformPath, *transformDryRun, *transformConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error transforming log entries: %v\n", err)
			os.Exit(1)
		}
		if *transformDryRun {
			fmt.Println("Dry run completed - no files were modified")
		} else {
			fmt.Println("Successfully transformed log entries")
		}

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
