# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Log Scraper (aka "Log-Needle Finder" / "Dag-Doctor") — a fast Go CLI tool for parsing massive log files from distributed systems. Users pipe gigabytes of logs into it, and it uses Go concurrency to scan for known failure patterns (OOM kills, timeout regexes, database locks) and highlights where the root failure started in a clean Terminal UI.

## Build & Run

```bash
make build            # Build to bin/logscraper
make test             # Run tests with -race
make bench            # Benchmark matcher/scanner
make vet              # Go vet
go run ./cmd/logscraper testdata/sample.log    # Quick run
```

## Tech Stack

- **Language:** Go 1.26+
- **TUI:** bubbletea v1 + lipgloss v1 (Charm ecosystem)
- **Pattern matching:** Compiled `regexp.Regexp` (stdlib), concurrency-safe
- **Concurrency:** Goroutine fan-out/fan-in with bounded channels

## Architecture

Streaming pipeline: `Reader → Scanner Pool → Matcher Pool → Aggregator → Analyzer → TUI`

- `internal/model/` — shared types (LogEntry, Match, FailureChain, Severity, Category). Zero internal imports.
- `internal/reader/` — file chunking (line-boundary aligned, 4MB min) and stdin streaming.
- `internal/scanner/` — line scanning with `sync.Pool` buffer reuse. `numCPU` goroutines.
- `internal/matcher/` — pre-compiled regex matching. 7 failure categories, ~40 patterns. Patterns defined in `patterns.go`.
- `internal/analyzer/` — timestamp parsing, temporal chain grouping (5s window), root cause identification via causal weight ranking.
- `internal/pipeline/` — orchestrates all stages, manages goroutine lifecycle and channel backpressure.
- `internal/tui/` — bubbletea app with three views (Summary, List, Detail).
- `cmd/logscraper/` — CLI entry point, flag parsing, output mode selection (TUI/JSON/plain).

## Key Design Decisions

- `LogEntry.Line` is `[]byte` (not string) to avoid per-line allocation. Only matched lines are copied.
- Patterns are hardcoded Go structs, not config files. `--pattern-file` can be added later.
- Timestamps parsed lazily — only on matched lines, not every line.
- Root cause promotion: within a failure chain, the highest causal-weight match becomes root cause (OOM > DiskIO > Signal > DBLock > Panic > Timeout > Network).
- Channel buffer sizes: chunks=2N, entries=4096, matches=1024.

## Key Design Goals

- Handle gigabyte-scale log files without freezing — stream, don't load into memory
- Accept piped stdin (`kubectl logs ... | log-scraper`) and file path args
- Scan concurrently across multiple files/log streams
- Ship with built-in patterns for common failures (OOM, timeouts, deadlocks, panics)
- Surface the root cause, not just the first error — trace failure chains

## Role

Operate as a Principal/Staff-level Software and Data Engineer. Evaluate scalability, edge cases, trade-offs, and complete lifecycles before proposing solutions. Think in systems — consider failure modes, memory pressure, concurrency hazards, and operational concerns upfront, not as afterthoughts.

## Communication

- Write with natural, human-like flow. No filler transitions, no "Great question!", no "Let's dive in."
- Match tone to context: crisp and precise for technical work; conversational for brainstorming; inventive for problem-solving.
- When explaining trade-offs, lead with the recommendation and the single most important reason — expand only if asked.

## Accuracy & Honesty

- **No fabrication.** If uncertain, say so. Never invent plausible-sounding answers.
- **Challenge flawed premises.** If my reasoning, approach, or architecture is wrong, say it directly and propose the better alternative. Do not passively agree.
- **Never invent credentials.** For any resume, cover letter, or career content: only use experiences, metrics, and skills I have explicitly provided. Ask targeted follow-up questions to fill gaps rather than fabricating.

## Execution Standards

- Analyze before building. For non-trivial work, identify the 2-3 critical decisions first, state your recommendation, then implement.
- Apply the same rigor to non-technical tasks (career strategy, communication, writing) as to engineering — consider all variables, not just the obvious ones.
- When multiple valid approaches exist, pick one and justify it. Don't present a menu unless the choice genuinely depends on my preference.
