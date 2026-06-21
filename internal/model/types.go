package model

import (
	"encoding/json"
	"time"
)

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityError
	SeverityCritical
	SeverityFatal
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarn:
		return "WARN"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	case SeverityFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Category int

const (
	CategoryOOMKill Category = iota
	CategoryTimeout
	CategoryDBLock
	CategoryPanic
	CategorySignal
	CategoryDiskIO
	CategoryNetwork
)

func (c Category) String() string {
	switch c {
	case CategoryOOMKill:
		return "OOM Kill"
	case CategoryTimeout:
		return "Timeout"
	case CategoryDBLock:
		return "DB Lock"
	case CategoryPanic:
		return "Panic"
	case CategorySignal:
		return "Signal"
	case CategoryDiskIO:
		return "Disk I/O"
	case CategoryNetwork:
		return "Network"
	default:
		return "Unknown"
	}
}

// CausalWeight returns how likely this category is to be a root cause.
// Higher = more likely root cause.
func (c Category) CausalWeight() int {
	switch c {
	case CategoryOOMKill:
		return 7
	case CategoryDiskIO:
		return 6
	case CategorySignal:
		return 5
	case CategoryDBLock:
		return 4
	case CategoryPanic:
		return 3
	case CategoryTimeout:
		return 2
	case CategoryNetwork:
		return 1
	default:
		return 0
	}
}

func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (c Category) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

type Pattern struct {
	Name     string   `json:"name"`
	Category Category `json:"category"`
	Severity Severity `json:"severity"`
	Regex    string   `json:"-"`
}

type LogEntry struct {
	Source  string `json:"source"`
	LineNum int64  `json:"line_num"`
	Line    []byte `json:"-"`
	LineStr string `json:"line,omitempty"`
}

func (e LogEntry) MarshalJSON() ([]byte, error) {
	type Alias LogEntry
	return json.Marshal(&struct {
		Alias
		Line string `json:"line"`
	}{
		Alias: Alias(e),
		Line:  string(e.Line),
	})
}

type Match struct {
	Entry     LogEntry  `json:"entry"`
	Pattern   Pattern   `json:"pattern"`
	Submatch  []string  `json:"submatch,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type FailureChain struct {
	ID        int           `json:"id"`
	RootCause Match         `json:"root_cause"`
	Cascade   []Match       `json:"cascade,omitempty"`
	TimeSpan  time.Duration `json:"time_span_ms"`
	Category  Category      `json:"category"`
	Severity  Severity      `json:"severity"`
}

func (fc FailureChain) MarshalJSON() ([]byte, error) {
	type Alias FailureChain
	return json.Marshal(&struct {
		Alias
		TimeSpanMs int64 `json:"time_span_ms"`
	}{
		Alias:      Alias(fc),
		TimeSpanMs: fc.TimeSpan.Milliseconds(),
	})
}

type Result struct {
	Chains       []FailureChain     `json:"chains"`
	TotalLines   int64              `json:"total_lines"`
	TotalMatches int                `json:"total_matches"`
	ByCategory   map[Category]int   `json:"by_category"`
	BySeverity   map[Severity]int   `json:"by_severity"`
	Duration     time.Duration      `json:"-"`
	DurationMs   int64              `json:"duration_ms"`
	Sources      []string           `json:"sources"`
}

func (r Result) MarshalJSON() ([]byte, error) {
	byCat := make(map[string]int, len(r.ByCategory))
	for k, v := range r.ByCategory {
		byCat[k.String()] = v
	}
	bySev := make(map[string]int, len(r.BySeverity))
	for k, v := range r.BySeverity {
		bySev[k.String()] = v
	}

	return json.Marshal(&struct {
		Chains       []FailureChain `json:"chains"`
		TotalLines   int64          `json:"total_lines"`
		TotalMatches int            `json:"total_matches"`
		ByCategory   map[string]int `json:"by_category"`
		BySeverity   map[string]int `json:"by_severity"`
		DurationMs   int64          `json:"duration_ms"`
		Sources      []string       `json:"sources"`
	}{
		Chains:       r.Chains,
		TotalLines:   r.TotalLines,
		TotalMatches: r.TotalMatches,
		ByCategory:   byCat,
		BySeverity:   bySev,
		DurationMs:   r.Duration.Milliseconds(),
		Sources:      r.Sources,
	})
}

type Chunk struct {
	Source    string
	StartByte int64
	EndByte   int64
	Data     []byte
}
