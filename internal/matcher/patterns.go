package matcher

import "github.com/notWizzy/log-scraper/internal/model"

var DefaultPatterns = []model.Pattern{
	// OOM Kill
	{Name: "oom_kill", Category: model.CategoryOOMKill, Severity: model.SeverityFatal, Regex: `(?i)out of memory:?\s*kill`},
	{Name: "oom_killed", Category: model.CategoryOOMKill, Severity: model.SeverityFatal, Regex: `(?i)oom[-_]?kill`},
	{Name: "oom_alloc", Category: model.CategoryOOMKill, Severity: model.SeverityCritical, Regex: `(?i)cannot allocate memory`},
	{Name: "oom_cgroup", Category: model.CategoryOOMKill, Severity: model.SeverityFatal, Regex: `(?i)memory cgroup out of memory`},
	{Name: "oom_java_heap", Category: model.CategoryOOMKill, Severity: model.SeverityFatal, Regex: `java\.lang\.OutOfMemoryError`},

	// Timeout
	{Name: "ctx_deadline", Category: model.CategoryTimeout, Severity: model.SeverityError, Regex: `context deadline exceeded`},
	{Name: "io_timeout", Category: model.CategoryTimeout, Severity: model.SeverityError, Regex: `i/o timeout`},
	{Name: "conn_timeout", Category: model.CategoryTimeout, Severity: model.SeverityError, Regex: `(?i)connection timed? ?out`},
	{Name: "req_timeout", Category: model.CategoryTimeout, Severity: model.SeverityError, Regex: `(?i)request timed? ?out`},
	{Name: "read_timeout", Category: model.CategoryTimeout, Severity: model.SeverityError, Regex: `read tcp.*timeout`},
	{Name: "ctx_canceled", Category: model.CategoryTimeout, Severity: model.SeverityWarn, Regex: `context canceled`},

	// DB Lock / Deadlock
	{Name: "deadlock", Category: model.CategoryDBLock, Severity: model.SeverityCritical, Regex: `(?i)deadlock detected`},
	{Name: "lock_wait", Category: model.CategoryDBLock, Severity: model.SeverityCritical, Regex: `(?i)lock wait timeout exceeded`},
	{Name: "obtain_lock", Category: model.CategoryDBLock, Severity: model.SeverityError, Regex: `(?i)could not obtain lock`},
	{Name: "db_locked", Category: model.CategoryDBLock, Severity: model.SeverityError, Regex: `(?i)database is locked`},
	{Name: "row_lock", Category: model.CategoryDBLock, Severity: model.SeverityError, Regex: `(?i)lock on.*row`},

	// Panic / Stack Trace
	{Name: "go_panic", Category: model.CategoryPanic, Severity: model.SeverityFatal, Regex: `(?:^|\s)panic:`},
	{Name: "go_goroutine", Category: model.CategoryPanic, Severity: model.SeverityCritical, Regex: `(?:^|\s)goroutine \d+ \[`},
	{Name: "go_fatal", Category: model.CategoryPanic, Severity: model.SeverityFatal, Regex: `(?:^|\s)fatal error:`},
	{Name: "java_exception", Category: model.CategoryPanic, Severity: model.SeverityError, Regex: `(?i)exception in thread`},
	{Name: "java_stacktrace", Category: model.CategoryPanic, Severity: model.SeverityError, Regex: `^\s+at [a-zA-Z0-9_.]+\(.*:\d+\)`},
	{Name: "python_traceback", Category: model.CategoryPanic, Severity: model.SeverityError, Regex: `Traceback \(most recent call last\)`},
	{Name: "python_error", Category: model.CategoryPanic, Severity: model.SeverityError, Regex: `^\w+Error:`},
	{Name: "node_unhandled", Category: model.CategoryPanic, Severity: model.SeverityFatal, Regex: `(?i)unhandled(?:Promise)?Rejection`},
	{Name: "rust_panic", Category: model.CategoryPanic, Severity: model.SeverityFatal, Regex: `thread '.*' panicked at`},

	// Signal
	{Name: "sigkill", Category: model.CategorySignal, Severity: model.SeverityFatal, Regex: `(?i)signal:\s*killed|SIGKILL`},
	{Name: "sigterm", Category: model.CategorySignal, Severity: model.SeverityCritical, Regex: `(?i)signal:\s*terminated|SIGTERM`},
	{Name: "sigsegv", Category: model.CategorySignal, Severity: model.SeverityFatal, Regex: `(?i)signal:\s*segmentation fault|SIGSEGV`},
	{Name: "core_dump", Category: model.CategorySignal, Severity: model.SeverityFatal, Regex: `(?i)core dumped`},
	{Name: "sigabrt", Category: model.CategorySignal, Severity: model.SeverityFatal, Regex: `SIGABRT|signal:\s*aborted`},

	// Disk I/O
	{Name: "no_space", Category: model.CategoryDiskIO, Severity: model.SeverityCritical, Regex: `(?i)no space left on device`},
	{Name: "io_error", Category: model.CategoryDiskIO, Severity: model.SeverityCritical, Regex: `(?i)input/output error`},
	{Name: "readonly_fs", Category: model.CategoryDiskIO, Severity: model.SeverityCritical, Regex: `(?i)read-only file system`},
	{Name: "disk_quota", Category: model.CategoryDiskIO, Severity: model.SeverityError, Regex: `(?i)disk quota exceeded`},
	{Name: "eio", Category: model.CategoryDiskIO, Severity: model.SeverityCritical, Regex: `\bEIO\b`},
	{Name: "too_many_files", Category: model.CategoryDiskIO, Severity: model.SeverityError, Regex: `(?i)too many open files`},

	// Network
	{Name: "conn_refused", Category: model.CategoryNetwork, Severity: model.SeverityError, Regex: `(?i)connection refused`},
	{Name: "no_host", Category: model.CategoryNetwork, Severity: model.SeverityError, Regex: `(?i)no such host`},
	{Name: "dns_nxdomain", Category: model.CategoryNetwork, Severity: model.SeverityError, Regex: `(?i)dns.*nxdomain`},
	{Name: "conn_reset", Category: model.CategoryNetwork, Severity: model.SeverityError, Regex: `(?i)connection reset by peer`},
	{Name: "broken_pipe", Category: model.CategoryNetwork, Severity: model.SeverityError, Regex: `(?i)broken pipe`},
	{Name: "net_unreachable", Category: model.CategoryNetwork, Severity: model.SeverityError, Regex: `(?i)network is unreachable`},
	{Name: "host_unreachable", Category: model.CategoryNetwork, Severity: model.SeverityError, Regex: `(?i)no route to host`},
}
