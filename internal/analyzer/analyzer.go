package analyzer

import (
	"regexp"
	"sort"
	"time"

	"github.com/notWizzy/log-scraper/internal/model"
)

const defaultChainWindow = 5 * time.Second

// Common timestamp formats found in logs, ordered by prevalence.
var tsFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.000",
	"2006-01-02 15:04:05,000",
	"2006-01-02 15:04:05",
	"Jan  2 15:04:05",
	"Jan 2 15:04:05",
	"2006/01/02 15:04:05",
	"02/Jan/2006:15:04:05",
}

// Regex to extract a timestamp-like prefix from a log line.
var tsExtractRe = regexp.MustCompile(
	`^(\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}[.,]?\d*(?:Z|[+-]\d{2}:?\d{2})?)` +
		`|^([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})` +
		`|^(\d{2}/[A-Z][a-z]{2}/\d{4}:\d{2}:\d{2}:\d{2})`,
)

// Analyze groups matches into failure chains and identifies root causes.
func Analyze(matches []model.Match, chainWindow time.Duration) []model.FailureChain {
	if len(matches) == 0 {
		return nil
	}

	if chainWindow == 0 {
		chainWindow = defaultChainWindow
	}

	parseTimestamps(matches)
	sortMatches(matches)

	chains := groupIntoChains(matches, chainWindow)
	chains = mergeOverlapping(chains)

	for i := range chains {
		identifyRootCause(&chains[i])
		fillChainMeta(&chains[i])
		chains[i].ID = i + 1
	}

	sort.Slice(chains, func(i, j int) bool {
		if chains[i].Severity != chains[j].Severity {
			return chains[i].Severity > chains[j].Severity
		}
		return chains[i].RootCause.Timestamp.Before(chains[j].RootCause.Timestamp)
	})

	for i := range chains {
		chains[i].ID = i + 1
	}

	return chains
}

func parseTimestamps(matches []model.Match) {
	for i := range matches {
		matches[i].Timestamp = extractTimestamp(matches[i].Entry.Line)
	}
}

func extractTimestamp(line []byte) time.Time {
	sub := tsExtractRe.FindSubmatch(line)
	if sub == nil {
		return time.Time{}
	}

	var tsStr string
	for _, s := range sub[1:] {
		if len(s) > 0 {
			tsStr = string(s)
			break
		}
	}
	if tsStr == "" {
		return time.Time{}
	}

	for _, format := range tsFormats {
		if t, err := time.Parse(format, tsStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

func sortMatches(matches []model.Match) {
	sort.SliceStable(matches, func(i, j int) bool {
		ti, tj := matches[i].Timestamp, matches[j].Timestamp
		if !ti.IsZero() && !tj.IsZero() {
			return ti.Before(tj)
		}
		if !ti.IsZero() {
			return true
		}
		if !tj.IsZero() {
			return false
		}
		if matches[i].Entry.Source != matches[j].Entry.Source {
			return matches[i].Entry.Source < matches[j].Entry.Source
		}
		return matches[i].Entry.LineNum < matches[j].Entry.LineNum
	})
}

func groupIntoChains(matches []model.Match, window time.Duration) []model.FailureChain {
	if len(matches) == 0 {
		return nil
	}

	var chains []model.FailureChain
	current := model.FailureChain{
		RootCause: matches[0],
	}

	for i := 1; i < len(matches); i++ {
		prev := lastInChain(current)
		gap := matchGap(prev, matches[i])

		if gap > window {
			chains = append(chains, current)
			current = model.FailureChain{
				RootCause: matches[i],
			}
		} else {
			current.Cascade = append(current.Cascade, matches[i])
		}
	}
	chains = append(chains, current)

	for i := range chains {
		fillChainMeta(&chains[i])
	}

	return chains
}

func lastInChain(chain model.FailureChain) model.Match {
	if len(chain.Cascade) > 0 {
		return chain.Cascade[len(chain.Cascade)-1]
	}
	return chain.RootCause
}

func matchGap(a, b model.Match) time.Duration {
	if a.Timestamp.IsZero() || b.Timestamp.IsZero() {
		// Without timestamps, use line proximity within same source
		if a.Entry.Source == b.Entry.Source {
			lineDiff := b.Entry.LineNum - a.Entry.LineNum
			if lineDiff < 50 {
				return 0
			}
		}
		return defaultChainWindow + 1
	}
	return b.Timestamp.Sub(a.Timestamp)
}

func identifyRootCause(chain *model.FailureChain) {
	if len(chain.Cascade) == 0 {
		return
	}

	// Check if a higher-causal-weight match exists near the start
	bestIdx := -1
	bestWeight := chain.RootCause.Pattern.Category.CausalWeight()

	for i, m := range chain.Cascade {
		w := m.Pattern.Category.CausalWeight()
		if w > bestWeight {
			bestIdx = i
			bestWeight = w
		}
	}

	if bestIdx >= 0 {
		// Swap root cause with the higher-weight match
		old := chain.RootCause
		chain.RootCause = chain.Cascade[bestIdx]
		chain.Cascade[bestIdx] = old
		sort.SliceStable(chain.Cascade, func(i, j int) bool {
			if !chain.Cascade[i].Timestamp.IsZero() && !chain.Cascade[j].Timestamp.IsZero() {
				return chain.Cascade[i].Timestamp.Before(chain.Cascade[j].Timestamp)
			}
			return chain.Cascade[i].Entry.LineNum < chain.Cascade[j].Entry.LineNum
		})
	}
}

func fillChainMeta(chain *model.FailureChain) {
	chain.Category = chain.RootCause.Pattern.Category
	chain.Severity = chain.RootCause.Pattern.Severity

	for _, m := range chain.Cascade {
		if m.Pattern.Severity > chain.Severity {
			chain.Severity = m.Pattern.Severity
		}
	}

	if !chain.RootCause.Timestamp.IsZero() && len(chain.Cascade) > 0 {
		last := chain.Cascade[len(chain.Cascade)-1]
		if !last.Timestamp.IsZero() {
			chain.TimeSpan = last.Timestamp.Sub(chain.RootCause.Timestamp)
		}
	}
}

func mergeOverlapping(chains []model.FailureChain) []model.FailureChain {
	if len(chains) <= 1 {
		return chains
	}

	merged := []model.FailureChain{chains[0]}
	for i := 1; i < len(chains); i++ {
		last := &merged[len(merged)-1]
		curr := chains[i]

		if last.Category == curr.Category && chainsOverlap(*last, curr) {
			last.Cascade = append(last.Cascade, curr.RootCause)
			last.Cascade = append(last.Cascade, curr.Cascade...)
			fillChainMeta(last)
		} else {
			merged = append(merged, curr)
		}
	}
	return merged
}

func chainsOverlap(a, b model.FailureChain) bool {
	aEnd := lastInChain(a)
	if aEnd.Timestamp.IsZero() || b.RootCause.Timestamp.IsZero() {
		return false
	}
	return b.RootCause.Timestamp.Sub(aEnd.Timestamp) <= defaultChainWindow
}
