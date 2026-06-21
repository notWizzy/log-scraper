package scanner

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"sync"

	"github.com/notWizzy/log-scraper/internal/model"
)

const (
	initialBufSize = 64 * 1024  // 64KB
	maxLineSize    = 1024 * 1024 // 1MB max line length
)

var bufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, initialBufSize)
		return &buf
	},
}

// ScanChunks reads chunks from the channel and emits LogEntry values.
// For file-backed chunks (Data == nil), it reads from disk.
// For stdin chunks (Data != nil), it scans the in-memory buffer.
func ScanChunks(ctx context.Context, chunks <-chan model.Chunk, out chan<- model.LogEntry, lineCount *int64, mu *sync.Mutex) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-chunks:
			if !ok {
				return
			}
			scanChunk(ctx, chunk, out, lineCount, mu)
		}
	}
}

func scanChunk(ctx context.Context, chunk model.Chunk, out chan<- model.LogEntry, lineCount *int64, mu *sync.Mutex) {
	var r io.Reader

	if chunk.Data != nil {
		r = bytes.NewReader(chunk.Data)
	} else {
		f, err := os.Open(chunk.Source)
		if err != nil {
			return
		}
		defer f.Close()

		if chunk.StartByte > 0 {
			if _, err := f.Seek(chunk.StartByte, io.SeekStart); err != nil {
				return
			}
		}
		r = io.LimitReader(f, chunk.EndByte-chunk.StartByte)
	}

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)

	sc := bufio.NewScanner(r)
	sc.Buffer(*bufPtr, maxLineSize)

	var localCount int64
	// Compute starting line number from byte offset for file chunks.
	// For stdin, we use a global counter.
	var lineNum int64
	if chunk.Data != nil {
		mu.Lock()
		lineNum = *lineCount
		mu.Unlock()
	} else {
		lineNum = estimateLineStart(chunk)
	}

	for sc.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		localCount++
		lineNum++

		raw := sc.Bytes()
		line := make([]byte, len(raw))
		copy(line, raw)

		select {
		case out <- model.LogEntry{
			Source:  chunk.Source,
			LineNum: lineNum,
			Line:    line,
		}:
		case <-ctx.Done():
			return
		}
	}

	mu.Lock()
	*lineCount += localCount
	mu.Unlock()
}

// estimateLineStart returns 0 for the first chunk.
// For subsequent chunks, line numbers will be approximate since we
// don't know exact line counts without scanning. The pipeline
// re-numbers sequentially per source after collection.
func estimateLineStart(chunk model.Chunk) int64 {
	if chunk.StartByte == 0 {
		return 0
	}
	// Rough estimate: average 100 bytes per line
	return chunk.StartByte / 100
}
