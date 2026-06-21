package matcher

import (
	"bytes"
	"context"
	"regexp"
	"strings"

	"github.com/notWizzy/log-scraper/internal/model"
)

type CompiledPattern struct {
	Pattern  model.Pattern
	Re       *regexp.Regexp
	Literals [][]byte // fast pre-filter: line must contain at least one
}

type Matcher struct {
	patterns []CompiledPattern
}

func New(patterns []model.Pattern) (*Matcher, error) {
	compiled := make([]CompiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			return nil, err
		}
		cp := CompiledPattern{
			Pattern:  p,
			Re:       re,
			Literals: extractLiterals(p.Regex),
		}
		compiled = append(compiled, cp)
	}

	return &Matcher{patterns: compiled}, nil
}

func NewDefault() (*Matcher, error) {
	return New(DefaultPatterns)
}

// MatchLine tests a single line against all patterns.
// Returns the first match found (highest priority = earliest in pattern list).
func (m *Matcher) MatchLine(entry model.LogEntry) *model.Match {
	lower := bytes.ToLower(entry.Line)

	for i := range m.patterns {
		cp := &m.patterns[i]

		if len(cp.Literals) > 0 && !containsAnyLiteral(lower, cp.Literals) {
			continue
		}

		if cp.Re.Match(entry.Line) {
			match := &model.Match{
				Entry:   entry,
				Pattern: cp.Pattern,
			}
			sub := cp.Re.FindSubmatch(entry.Line)
			if len(sub) > 1 {
				match.Submatch = make([]string, len(sub)-1)
				for j := 1; j < len(sub); j++ {
					match.Submatch[j-1] = string(sub[j])
				}
			}
			return match
		}
	}
	return nil
}


func containsAnyLiteral(lowerLine []byte, literals [][]byte) bool {
	for _, lit := range literals {
		if bytes.Contains(lowerLine, lit) {
			return true
		}
	}
	return false
}

// extractLiterals pulls lowercase literal substrings from a regex
// that must appear in any matching line.
func extractLiterals(regex string) [][]byte {
	// Strip common regex prefixes/suffixes
	s := regex
	s = strings.TrimPrefix(s, "(?i)")
	s = strings.TrimPrefix(s, "(?:^|\\s)")
	s = strings.TrimPrefix(s, "^")
	s = strings.TrimSuffix(s, "$")

	// Split on regex alternation
	parts := strings.Split(s, "|")

	var literals [][]byte
	for _, part := range parts {
		lit := extractLiteralFromPart(part)
		if len(lit) >= 3 {
			literals = append(literals, []byte(strings.ToLower(lit)))
		}
	}
	return literals
}

func extractLiteralFromPart(part string) string {
	var result strings.Builder
	var best string

	for i := 0; i < len(part); i++ {
		ch := part[i]
		switch ch {
		case '\\':
			if i+1 < len(part) {
				next := part[i+1]
				switch next {
				case 's', 'd', 'w', 'b', 'S', 'D', 'W', 'B':
					if result.Len() > len(best) {
						best = result.String()
					}
					result.Reset()
				default:
					result.WriteByte(next)
				}
				i++
			}
		case '[', '(', ')', '*', '+', '?', '{', '}', '.', '^', '$':
			if result.Len() > len(best) {
				best = result.String()
			}
			result.Reset()
			// Skip character class
			if ch == '[' {
				for i < len(part) && part[i] != ']' {
					i++
				}
			}
			// Skip group content for non-capturing groups
			if ch == '(' {
				depth := 1
				for i+1 < len(part) && depth > 0 {
					i++
					if part[i] == '(' {
						depth++
					} else if part[i] == ')' {
						depth--
					}
				}
			}
		default:
			result.WriteByte(ch)
		}
	}
	if result.Len() > len(best) {
		best = result.String()
	}
	return best
}

// MatchEntries reads LogEntry values from entries and sends matches to out.
func (m *Matcher) MatchEntries(ctx context.Context, entries <-chan model.LogEntry, out chan<- model.Match) {
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-entries:
			if !ok {
				return
			}
			if match := m.MatchLine(entry); match != nil {
				select {
				case out <- *match:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
