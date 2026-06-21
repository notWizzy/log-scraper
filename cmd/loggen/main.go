package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

var severities = []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}

var normalMessages = []string{
	"Processing request from client %s",
	"Query executed in %dms",
	"Cache hit for key user:%d",
	"Worker %d picked up job %d",
	"Batch %d completed: %d records processed",
	"Health check passed",
	"Connection pool: %d active, %d idle",
	"Scheduled task %s started",
	"Metrics exported: cpu=%.1f%% mem=%.1f%%",
	"Session %s authenticated successfully",
	"Rate limiter: %d/%d requests used",
	"Replication lag: %dms",
	"GC pause: %dms, heap: %dMB",
	"TLS handshake completed with %s",
	"DNS resolved %s in %dms",
}

var oomPatterns = []string{
	"Out of memory: Killed process %d (java) total-vm:%dkB",
	"OOMKilled container %s in pod %s",
	"Cannot allocate memory for buffer of size %d",
	"memory cgroup out of memory: Kill process %d",
	"java.lang.OutOfMemoryError: Java heap space",
}

var timeoutPatterns = []string{
	"context deadline exceeded",
	"i/o timeout after %ds",
	"read tcp 10.0.%d.%d:%d->10.0.%d.%d:%d: i/o timeout",
	"connection timed out to %s:%d",
	"request timed out after %ds",
	"context canceled",
}

var dbPatterns = []string{
	"deadlock detected in transaction %d",
	"Lock wait timeout exceeded; try restarting transaction",
	"could not obtain lock on row in relation \"%s\"",
	"database is locked (SQLITE_BUSY)",
	"lock on row (id=%d) held by transaction %d",
}

var panicPatterns = []string{
	"panic: runtime error: index out of range [%d] with length %d",
	"goroutine %d [running]:",
	"fatal error: concurrent map writes",
	"Exception in thread \"main\" java.lang.NullPointerException",
	"\tat com.app.service.UserService.getUser(UserService.java:%d)",
	"Traceback (most recent call last):",
	"  File \"/app/main.py\", line %d, in <module>",
	"ValueError: invalid literal for int() with base 10: '%s'",
	"unhandledRejection: TypeError: Cannot read properties of undefined",
	"thread 'main' panicked at 'index out of bounds'",
}

var signalPatterns = []string{
	"signal: killed",
	"SIGKILL received by process %d",
	"signal: terminated",
	"SIGTERM: shutting down",
	"signal: segmentation fault",
	"SIGSEGV in thread %d",
	"core dumped",
	"SIGABRT: abort",
}

var diskPatterns = []string{
	"No space left on device",
	"input/output error on /dev/sda%d",
	"read-only file system: /data/logs",
	"disk quota exceeded for user %d",
	"EIO on block %d",
	"too many open files (ulimit: %d)",
}

var networkPatterns = []string{
	"connection refused to %s:%d",
	"no such host: %s.internal.svc",
	"DNS lookup failed: NXDOMAIN for %s",
	"connection reset by peer on socket %d",
	"broken pipe writing to %s:%d",
	"network is unreachable: 10.%d.%d.%d",
	"no route to host %s",
}

func main() {
	lines := flag.Int("lines", 10000, "Number of log lines to generate")
	errorRate := flag.Float64("error-rate", 0.05, "Fraction of lines that are errors (0.0-1.0)")
	output := flag.String("output", "", "Output file (default: stdout)")
	scenario := flag.String("scenario", "mixed", "Scenario: mixed, cascade, sparse, dense, noerrors, allerrors, notimestamp, multiformats, longlines, binaryish")
	seed := flag.Int64("seed", 0, "Random seed (0 = random)")
	flag.Parse()

	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(*seed))

	var w *os.File
	if *output != "" {
		var err error
		w, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	baseTime := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)

	switch *scenario {
	case "mixed":
		generateMixed(w, rng, baseTime, *lines, *errorRate)
	case "cascade":
		generateCascade(w, rng, baseTime, *lines)
	case "sparse":
		generateSparse(w, rng, baseTime, *lines)
	case "dense":
		generateDense(w, rng, baseTime, *lines)
	case "noerrors":
		generateNoErrors(w, rng, baseTime, *lines)
	case "allerrors":
		generateAllErrors(w, rng, baseTime, *lines)
	case "notimestamp":
		generateNoTimestamp(w, rng, *lines, *errorRate)
	case "multiformats":
		generateMultiFormats(w, rng, baseTime, *lines, *errorRate)
	case "longlines":
		generateLongLines(w, rng, baseTime, *lines)
	case "binaryish":
		generateBinaryish(w, rng, baseTime, *lines, *errorRate)
	default:
		fmt.Fprintf(os.Stderr, "Unknown scenario: %s\n", *scenario)
		os.Exit(1)
	}
}

func generateMixed(w *os.File, rng *rand.Rand, base time.Time, n int, errorRate float64) {
	allErrors := collectAllErrors()
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 100 * time.Millisecond)
		if rng.Float64() < errorRate {
			line := randomError(rng, allErrors)
			fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(time.RFC3339), fillPattern(rng, line))
		} else {
			sev := severities[rng.Intn(3)] // DEBUG/INFO/WARN
			msg := normalMessages[rng.Intn(len(normalMessages))]
			fmt.Fprintf(w, "%s %s %s\n", ts.Format(time.RFC3339), sev, fillPattern(rng, msg))
		}
	}
}

// Cascade: burst of related errors within short time windows
func generateCascade(w *os.File, rng *rand.Rand, base time.Time, n int) {
	categories := [][]string{oomPatterns, timeoutPatterns, dbPatterns, panicPatterns, signalPatterns, diskPatterns, networkPatterns}
	i := 0
	for i < n {
		// Normal lines
		normalCount := 50 + rng.Intn(200)
		for j := 0; j < normalCount && i < n; j++ {
			ts := base.Add(time.Duration(i) * 100 * time.Millisecond)
			msg := normalMessages[rng.Intn(len(normalMessages))]
			fmt.Fprintf(w, "%s INFO %s\n", ts.Format(time.RFC3339), fillPattern(rng, msg))
			i++
		}
		// Error cascade: pick a category, emit 5-15 errors rapidly
		cat := categories[rng.Intn(len(categories))]
		burstSize := 5 + rng.Intn(11)
		for j := 0; j < burstSize && i < n; j++ {
			ts := base.Add(time.Duration(i)*100*time.Millisecond + time.Duration(j)*50*time.Millisecond)
			pattern := cat[rng.Intn(len(cat))]
			fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(time.RFC3339Nano), fillPattern(rng, pattern))
			i++
		}
	}
}

// Sparse: errors spread far apart (>5s gaps, should be separate chains)
func generateSparse(w *os.File, rng *rand.Rand, base time.Time, n int) {
	allErrors := collectAllErrors()
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 10 * time.Second)
		if i%100 == 50 {
			line := randomError(rng, allErrors)
			fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(time.RFC3339), fillPattern(rng, line))
		} else {
			msg := normalMessages[rng.Intn(len(normalMessages))]
			fmt.Fprintf(w, "%s INFO %s\n", ts.Format(time.RFC3339), fillPattern(rng, msg))
		}
	}
}

// Dense: extremely high error rate, many categories intermixed
func generateDense(w *os.File, rng *rand.Rand, base time.Time, n int) {
	allErrors := collectAllErrors()
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 10 * time.Millisecond)
		line := randomError(rng, allErrors)
		fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(time.RFC3339Nano), fillPattern(rng, line))
	}
}

func generateNoErrors(w *os.File, rng *rand.Rand, base time.Time, n int) {
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 100 * time.Millisecond)
		msg := normalMessages[rng.Intn(len(normalMessages))]
		fmt.Fprintf(w, "%s INFO %s\n", ts.Format(time.RFC3339), fillPattern(rng, msg))
	}
}

func generateAllErrors(w *os.File, rng *rand.Rand, base time.Time, n int) {
	allErrors := collectAllErrors()
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 50 * time.Millisecond)
		line := randomError(rng, allErrors)
		fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(time.RFC3339), fillPattern(rng, line))
	}
}

// No timestamps at all
func generateNoTimestamp(w *os.File, rng *rand.Rand, n int, errorRate float64) {
	allErrors := collectAllErrors()
	for i := 0; i < n; i++ {
		if rng.Float64() < errorRate {
			line := randomError(rng, allErrors)
			fmt.Fprintf(w, "ERROR %s\n", fillPattern(rng, line))
		} else {
			msg := normalMessages[rng.Intn(len(normalMessages))]
			fmt.Fprintf(w, "INFO %s\n", fillPattern(rng, msg))
		}
	}
}

// Multiple timestamp formats in the same file
func generateMultiFormats(w *os.File, rng *rand.Rand, base time.Time, n int, errorRate float64) {
	allErrors := collectAllErrors()
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05,000",
		"Jan  2 15:04:05",
		"2006/01/02 15:04:05",
	}
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 100 * time.Millisecond)
		format := formats[rng.Intn(len(formats))]
		if rng.Float64() < errorRate {
			line := randomError(rng, allErrors)
			fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(format), fillPattern(rng, line))
		} else {
			msg := normalMessages[rng.Intn(len(normalMessages))]
			fmt.Fprintf(w, "%s INFO %s\n", ts.Format(format), fillPattern(rng, msg))
		}
	}
}

// Lines exceeding 1MB to stress the scanner buffer
func generateLongLines(w *os.File, rng *rand.Rand, base time.Time, n int) {
	allErrors := collectAllErrors()
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 100 * time.Millisecond)
		if i%100 == 0 {
			// Generate a very long line (500KB - 900KB)
			padding := strings.Repeat("x", 500000+rng.Intn(400000))
			line := randomError(rng, allErrors)
			fmt.Fprintf(w, "%s ERROR %s %s\n", ts.Format(time.RFC3339), fillPattern(rng, line), padding)
		} else if i%50 == 0 {
			line := randomError(rng, allErrors)
			fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(time.RFC3339), fillPattern(rng, line))
		} else {
			msg := normalMessages[rng.Intn(len(normalMessages))]
			fmt.Fprintf(w, "%s INFO %s\n", ts.Format(time.RFC3339), fillPattern(rng, msg))
		}
	}
}

// Lines with binary-ish content mixed in (null bytes, high bytes)
func generateBinaryish(w *os.File, rng *rand.Rand, base time.Time, n int, errorRate float64) {
	allErrors := collectAllErrors()
	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * 100 * time.Millisecond)
		if i%200 == 0 {
			// Binary-ish line with some high bytes
			garbage := make([]byte, 50+rng.Intn(100))
			for j := range garbage {
				garbage[j] = byte(rng.Intn(256))
			}
			// Replace newlines to avoid breaking line scanning
			for j := range garbage {
				if garbage[j] == '\n' || garbage[j] == '\r' {
					garbage[j] = '?'
				}
			}
			fmt.Fprintf(w, "%s WARN binary-content: %s\n", ts.Format(time.RFC3339), string(garbage))
		} else if rng.Float64() < errorRate {
			line := randomError(rng, allErrors)
			fmt.Fprintf(w, "%s ERROR %s\n", ts.Format(time.RFC3339), fillPattern(rng, line))
		} else {
			msg := normalMessages[rng.Intn(len(normalMessages))]
			fmt.Fprintf(w, "%s INFO %s\n", ts.Format(time.RFC3339), fillPattern(rng, msg))
		}
	}
}

func collectAllErrors() []string {
	var all []string
	all = append(all, oomPatterns...)
	all = append(all, timeoutPatterns...)
	all = append(all, dbPatterns...)
	all = append(all, panicPatterns...)
	all = append(all, signalPatterns...)
	all = append(all, diskPatterns...)
	all = append(all, networkPatterns...)
	return all
}

func randomError(rng *rand.Rand, errors []string) string {
	return errors[rng.Intn(len(errors))]
}

func fillPattern(rng *rand.Rand, pattern string) string {
	result := pattern
	for strings.Contains(result, "%d") {
		result = strings.Replace(result, "%d", fmt.Sprintf("%d", rng.Intn(10000)), 1)
	}
	for strings.Contains(result, "%s") {
		words := []string{"app-worker", "api-server", "db-primary", "cache-01", "web-frontend", "batch-processor", "queue-consumer", "scheduler"}
		result = strings.Replace(result, "%s", words[rng.Intn(len(words))], 1)
	}
	for strings.Contains(result, "%.1f") {
		result = strings.Replace(result, "%.1f", fmt.Sprintf("%.1f", rng.Float64()*100), 1)
	}
	return result
}
