package jobs

import (
	"log/slog"
	"sync"
	"time"

	"github.com/selfhostly/internal/db"
)

const (
	jobLogFlushMinLines    = 32
	jobLogFlushMaxInterval = 120 * time.Millisecond
	jobLogMaxBufferLines   = 2000 // prevent unbounded memory growth on DB errors
)

// JobLogWriter buffers deployment log lines and flushes to the database in batches.
type JobLogWriter struct {
	db        *db.DB
	jobID     string
	mu        sync.Mutex
	buf       []string
	lastFlush time.Time
}

// NewJobLogWriter creates a writer for the given job. Call Flush when the job step completes.
func NewJobLogWriter(database *db.DB, jobID string) *JobLogWriter {
	return &JobLogWriter{
		db:        database,
		jobID:     jobID,
		lastFlush: time.Now(),
	}
}

// WriteLine appends a line (implements the docker log callback shape).
func (w *JobLogWriter) WriteLine(line string) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, line)
	if len(w.buf) >= jobLogFlushMinLines || time.Since(w.lastFlush) >= jobLogFlushMaxInterval {
		w.flushLocked()
		w.lastFlush = time.Now()
	}
}

// Flush writes any buffered lines to the database.
func (w *JobLogWriter) Flush() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.flushLocked()
	w.lastFlush = time.Now()
}

func (w *JobLogWriter) flushLocked() {
	if len(w.buf) == 0 {
		return
	}
	lines := append([]string(nil), w.buf...)
	w.buf = w.buf[:0]
	if err := w.db.AppendJobLogLines(w.jobID, lines); err != nil {
		slog.Warn("failed to persist job log batch", "job_id", w.jobID, "error", err)
		// Re-queue failed lines, but cap buffer to prevent unbounded memory growth
		if len(w.buf)+len(lines) <= jobLogMaxBufferLines {
			w.buf = append(lines, w.buf...)
		} else {
			// Buffer overflow - drop oldest lines to stay within limit
			keep := jobLogMaxBufferLines - len(w.buf)
			if keep > 0 && keep < len(lines) {
				w.buf = append(lines[len(lines)-keep:], w.buf...)
			}
			slog.Error("job log buffer overflow, dropping lines", "job_id", w.jobID, "dropped", len(lines)-keep)
		}
	}
}
