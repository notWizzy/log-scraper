package reader

import (
	"context"
	"io"
	"os"
	"runtime"

	"github.com/notWizzy/log-scraper/internal/model"
)

const (
	minChunkSize = 4 * 1024 * 1024  // 4MB
	stdinBufSize = 4 * 1024 * 1024  // 4MB
)

// ReadFiles chunks multiple files and sends chunks on the channel.
// Closes the channel when all files are processed.
func ReadFiles(ctx context.Context, paths []string, out chan<- model.Chunk) error {
	for _, path := range paths {
		if err := chunkFile(ctx, path, out); err != nil {
			return err
		}
	}
	return nil
}

// ReadStdin reads from stdin in fixed-size blocks.
func ReadStdin(ctx context.Context, out chan<- model.Chunk) error {
	buf := make([]byte, stdinBufSize)
	var offset int64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := os.Stdin.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			select {
			case out <- model.Chunk{
				Source:    "stdin",
				StartByte: offset,
				EndByte:   offset + int64(n),
				Data:     data,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
			offset += int64(n)
		}

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func chunkFile(ctx context.Context, path string, out chan<- model.Chunk) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	size := info.Size()
	if size == 0 {
		return nil
	}

	numChunks := int64(runtime.NumCPU())
	chunkSize := size / numChunks
	if chunkSize < minChunkSize {
		chunkSize = minChunkSize
	}
	if chunkSize > size {
		chunkSize = size
	}

	var start int64
	for start < size {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := start + chunkSize
		if end > size {
			end = size
		}

		// Align end to next newline boundary (unless at EOF)
		if end < size {
			aligned, err := alignToNewline(f, end)
			if err != nil {
				return err
			}
			end = aligned
		}

		select {
		case out <- model.Chunk{
			Source:    path,
			StartByte: start,
			EndByte:   end,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}

		start = end
	}

	return nil
}

// alignToNewline seeks forward from pos to find the next newline,
// returning the position just after it.
func alignToNewline(f *os.File, pos int64) (int64, error) {
	buf := make([]byte, 4096)
	offset := pos

	for {
		n, err := f.ReadAt(buf, offset)
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				return offset + int64(i) + 1, nil
			}
		}
		offset += int64(n)

		if err == io.EOF {
			return offset, nil
		}
		if err != nil {
			return 0, err
		}
	}
}

// IsStdin returns true if stdin is a pipe (not a terminal).
func IsStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}
