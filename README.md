# log-scraper

A fast Go CLI tool for parsing massive log files from distributed systems. Pipe gigabytes of logs into it, and it concurrently scans for known failure patterns — OOM kills, timeouts, deadlocks, panics, signals, disk errors, network failures — then surfaces the root cause in a clean Terminal UI.

## Install

```bash
go install github.com/notWizzy/log-scraper/cmd/logscraper@latest
```

Or build from source:

```bash
git clone https://github.com/notWizzy/log-scraper.git
cd log-scraper
make build
```

## Usage

```bash
# Scan log files
logscraper app.log worker.log

# Pipe from other tools
kubectl logs pod-name | logscraper
cat /var/log/syslog | logscraper
docker logs container-name 2>&1 | logscraper

# Output as JSON (for scripting)
logscraper --json app.log

# Plain text output (no TUI)
logscraper --no-tui app.log
```

## What it detects

| Category | Example patterns |
|---|---|
| OOM Kill | `Out of memory: Kill`, `OOMKilled`, `Cannot allocate memory` |
| Timeout | `context deadline exceeded`, `i/o timeout`, `connection timed out` |
| DB Lock | `deadlock detected`, `Lock wait timeout`, `database is locked` |
| Panic | `panic:`, `goroutine N [running]`, `Traceback`, `Exception in thread` |
| Signal | `SIGKILL`, `SIGTERM`, `SIGSEGV`, `core dumped` |
| Disk I/O | `No space left on device`, `read-only file system`, `input/output error` |
| Network | `connection refused`, `no such host`, `DNS NXDOMAIN` |

## TUI Navigation

| Key | Action |
|---|---|
| `Enter` / `l` | Open selected item / enter view |
| `Esc` / `h` | Go back |
| `j` / `k` | Navigate up/down |
| `Space` | Expand/collapse chain details |
| `Tab` | Switch to summary view |
| `q` | Quit |

## How it works

1. **Reader** chunks files at line boundaries (4MB minimum per chunk) for parallel processing
2. **Scanner** pool (`NumCPU` goroutines) scans chunks line-by-line with pooled buffers via `sync.Pool`
3. **Matcher** pool runs pre-compiled regexes against each line — first match wins
4. **Analyzer** sorts matches by timestamp, groups temporally-related errors into failure chains, and identifies the root cause using causal weight ranking (OOM > Disk > Signal > DB Lock > Panic > Timeout > Network)
5. **TUI** renders results with three navigable views: Summary, Failure List, and Detail

## Flags

```
-json       Output results as JSON
-no-tui     Output plain text summary
-version    Print version and exit
```

## Build

```bash
make build       # Build binary to bin/logscraper
make test        # Run tests with race detector
make bench       # Run benchmarks
make vet         # Run go vet
make release     # Cross-compile for linux/darwin amd64/arm64
make clean       # Remove build artifacts
```
