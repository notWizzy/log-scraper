package pipeline

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/notWizzy/log-scraper/internal/analyzer"
	"github.com/notWizzy/log-scraper/internal/matcher"
	"github.com/notWizzy/log-scraper/internal/model"
	"github.com/notWizzy/log-scraper/internal/reader"
	"github.com/notWizzy/log-scraper/internal/scanner"
)

type Config struct {
	Sources     []string
	Stdin       bool
	ChainWindow time.Duration
}

func Run(ctx context.Context, cfg Config) (*model.Result, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	m, err := matcher.NewDefault()
	if err != nil {
		return nil, err
	}

	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}

	chunks := make(chan model.Chunk, 2*numWorkers)
	entries := make(chan model.LogEntry, 4096)
	matches := make(chan model.Match, 1024)

	start := time.Now()
	var totalLines int64
	var lineCountMu sync.Mutex

	// Stage 1: Reader
	var readerErr error
	var readerWg sync.WaitGroup
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		defer close(chunks)
		if cfg.Stdin {
			readerErr = reader.ReadStdin(ctx, chunks)
		} else {
			readerErr = reader.ReadFiles(ctx, cfg.Sources, chunks)
		}
	}()

	// Stage 2: Scanners (fan-out)
	var scanWg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		scanWg.Add(1)
		go func() {
			defer scanWg.Done()
			scanner.ScanChunks(ctx, chunks, entries, &totalLines, &lineCountMu)
		}()
	}
	go func() {
		scanWg.Wait()
		close(entries)
	}()

	// Stage 3: Matchers (fan-out)
	var matchWg sync.WaitGroup
	var matchCount atomic.Int64
	for i := 0; i < numWorkers; i++ {
		matchWg.Add(1)
		go func() {
			defer matchWg.Done()
			countingMatchEntries(ctx, m, entries, matches, &matchCount)
		}()
	}
	go func() {
		matchWg.Wait()
		close(matches)
	}()

	// Stage 4: Collect matches
	var allMatches []model.Match
	for match := range matches {
		allMatches = append(allMatches, match)
	}

	readerWg.Wait()
	if readerErr != nil {
		return nil, readerErr
	}

	// Stage 5: Analyze
	chains := analyzer.Analyze(allMatches, cfg.ChainWindow)

	// Build result
	byCategory := make(map[model.Category]int)
	bySeverity := make(map[model.Severity]int)
	for _, m := range allMatches {
		byCategory[m.Pattern.Category]++
		bySeverity[m.Pattern.Severity]++
	}

	sources := cfg.Sources
	if cfg.Stdin {
		sources = []string{"stdin"}
	}

	return &model.Result{
		Chains:       chains,
		TotalLines:   totalLines,
		TotalMatches: int(matchCount.Load()),
		ByCategory:   byCategory,
		BySeverity:   bySeverity,
		Duration:     time.Since(start),
		Sources:      sources,
	}, nil
}

func countingMatchEntries(ctx context.Context, m *matcher.Matcher, entries <-chan model.LogEntry, out chan<- model.Match, count *atomic.Int64) {
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-entries:
			if !ok {
				return
			}
			if match := m.MatchLine(entry); match != nil {
				count.Add(1)

				select {
				case out <- *match:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
