package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/notWizzy/log-scraper/internal/model"
	"github.com/notWizzy/log-scraper/internal/pipeline"
	"github.com/notWizzy/log-scraper/internal/reader"
	"github.com/notWizzy/log-scraper/internal/tui"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	jsonOut := flag.Bool("json", false, "Output results as JSON")
	noTUI := flag.Bool("no-tui", false, "Output plain text summary instead of TUI")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("logscraper %s (commit: %s, built: %s)\n", version, commit, buildDate)
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	files := flag.Args()
	useStdin := reader.IsStdin()

	if len(files) == 0 && !useStdin {
		fmt.Fprintln(os.Stderr, "Usage: logscraper [flags] <file1.log> [file2.log ...]")
		fmt.Fprintln(os.Stderr, "       cat logs.txt | logscraper")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate files exist
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	}

	cfg := pipeline.Config{
		Sources: files,
		Stdin:   useStdin,
	}

	result, err := pipeline.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	switch {
	case *jsonOut:
		outputJSON(result)
	case *noTUI:
		outputPlain(result)
	default:
		runTUI(result)
	}
}

func outputJSON(result *model.Result) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %s\n", err)
		os.Exit(1)
	}
}

func outputPlain(result *model.Result) {
	fmt.Printf("Log Scraper Results\n")
	fmt.Printf("===================\n")
	fmt.Printf("Lines scanned:  %d\n", result.TotalLines)
	fmt.Printf("Matches found:  %d\n", result.TotalMatches)
	fmt.Printf("Failure chains: %d\n", len(result.Chains))
	fmt.Printf("Duration:       %s\n", result.Duration.Round(1_000_000))
	fmt.Printf("Sources:        %s\n\n", strings.Join(result.Sources, ", "))

	for _, chain := range result.Chains {
		fmt.Printf("Chain #%d [%s] %s\n",
			chain.ID, chain.Category, chain.Severity)
		fmt.Printf("  Root: %s:%d\n", chain.RootCause.Entry.Source, chain.RootCause.Entry.LineNum)
		lineStr := string(chain.RootCause.Entry.Line)
		if len(lineStr) > 200 {
			lineStr = lineStr[:200] + "..."
		}
		fmt.Printf("  Line: %s\n", lineStr)
		if len(chain.Cascade) > 0 {
			fmt.Printf("  Cascade: %d related errors\n", len(chain.Cascade))
		}
		fmt.Println()
	}

	if len(result.Chains) == 0 {
		fmt.Println("No failures detected.")
	}
}

func runTUI(result *model.Result) {
	app := tui.NewApp(result)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %s\n", err)
		os.Exit(1)
	}
}
