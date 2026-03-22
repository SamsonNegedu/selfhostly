package http

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

// getJobLogs returns deployment log lines after a cursor (REST fallback for SSE).
func (s *Server) getJobLogs(c *gin.Context) {
	jobID := c.Param("id")
	after, err := strconv.ParseInt(c.DefaultQuery("after", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid after",
			Details: "after must be an integer sequence cursor",
		})
		return
	}

	job, err := s.database.GetJob(jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Job not found",
				Details: "Could not find job with the specified ID",
			})
			return
		}
		s.handleServiceError(c, "get job", err)
		return
	}

	lines, err := s.database.GetJobLogsAfter(jobID, after, 500)
	if err != nil {
		s.handleServiceError(c, "get job logs", err)
		return
	}

	nextAfter := after
	for _, l := range lines {
		if l.Seq > nextAfter {
			nextAfter = l.Seq
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"lines":      lines,
		"next_after": nextAfter,
		"job_status": job.Status,
	})
}

// streamJobLogs streams deployment log lines as Server-Sent Events until the job finishes and buffers drain.
func (s *Server) streamJobLogs(c *gin.Context) {
	jobID := c.Param("id")
	afterStr := c.DefaultQuery("after", "0")
	if lid := c.GetHeader("Last-Event-ID"); lid != "" {
		afterStr = lid
	}
	after, err := strconv.ParseInt(afterStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid after",
			Details: "after / Last-Event-ID must be an integer sequence cursor",
		})
		return
	}

	if _, err := s.database.GetJob(jobID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Job not found",
				Details: "Could not find job with the specified ID",
			})
			return
		}
		s.handleServiceError(c, "get job", err)
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Streaming unsupported",
			Details: "Response writer does not support flush",
		})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flusher.Flush()

	lastSeq := after
	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()
	// Poll every 500ms - balances responsiveness with DB load
	// Docker output is typically batched, so sub-second polling provides good UX
	poll := time.NewTicker(500 * time.Millisecond)
	defer poll.Stop()

	terminalIdle := 0

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-keepalive.C:
			_, _ = fmt.Fprintf(c.Writer, ": keepalive\n\n")
			flusher.Flush()
		case <-poll.C:
			lines, err := s.database.GetJobLogsAfter(jobID, lastSeq, 500)
			if err != nil {
				return
			}
			for _, line := range lines {
				b, err := json.Marshal(db.JobLogLine{Seq: line.Seq, Text: line.Text})
				if err != nil {
					continue
				}
				_, _ = fmt.Fprintf(c.Writer, "event: line\nid: %d\ndata: %s\n\n", line.Seq, b)
				lastSeq = line.Seq
			}
			flusher.Flush()

			j, err := s.database.GetJob(jobID)
			if err != nil {
				return
			}
			if j.Status == constants.JobStatusCompleted || j.Status == constants.JobStatusFailed {
				if len(lines) == 0 {
					terminalIdle++
					if terminalIdle >= 2 {
						done, _ := json.Marshal(gin.H{"status": j.Status})
						_, _ = fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", done)
						flusher.Flush()
						return
					}
				} else {
					terminalIdle = 0
				}
			}
		}
	}
}
