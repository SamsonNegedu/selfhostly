package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/selfhostly/internal/constants"
)

const jobLogTrimBatch = 100

// AppendJobLogLines appends lines for a job, assigning monotonic seq per job.
// When total stored bytes exceed JobLogMaxBytes, oldest lines are removed in batches.
func (db *DB) AppendJobLogLines(jobID string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}

	tx, err := db.BeginTx(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var maxSeq sql.NullInt64
	err = tx.QueryRow(`SELECT MAX(seq) FROM job_logs WHERE job_id = ?`, jobID).Scan(&maxSeq)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	seq := int64(0)
	if maxSeq.Valid {
		seq = maxSeq.Int64
	}

	for _, line := range lines {
		seq++
		_, err = tx.Exec(`INSERT INTO job_logs (job_id, seq, line) VALUES (?, ?, ?)`, jobID, seq, line)
		if err != nil {
			return fmt.Errorf("insert job log line: %w", err)
		}
	}

	if err := trimJobLogsTx(tx, jobID); err != nil {
		return err
	}

	return tx.Commit()
}

func trimJobLogsTx(tx *Tx, jobID string) error {
	// Check total size once before trimming loop
	var total sql.NullInt64
	err := tx.QueryRow(`SELECT COALESCE(SUM(LENGTH(line)), 0) FROM job_logs WHERE job_id = ?`, jobID).Scan(&total)
	if err != nil {
		return err
	}
	if !total.Valid || total.Int64 <= int64(constants.JobLogMaxBytes) {
		return nil
	}

	// Calculate how many bytes to remove (with 10% buffer to avoid frequent re-trimming)
	targetRemoval := total.Int64 - int64(constants.JobLogMaxBytes) + (int64(constants.JobLogMaxBytes) / 10)

	// Delete oldest lines until we've removed enough bytes
	// Use a single query with LIMIT based on estimated lines to remove
	estimatedLinesToRemove := (targetRemoval / 100) + jobLogTrimBatch // assume ~100 bytes/line average
	if estimatedLinesToRemove > 1000 {
		estimatedLinesToRemove = 1000 // cap at 1000 lines per trim
	}

	_, err = tx.Exec(`
		DELETE FROM job_logs 
		WHERE job_id = ? AND seq IN (
			SELECT seq FROM job_logs 
			WHERE job_id = ? 
			ORDER BY seq ASC 
			LIMIT ?
		)`, jobID, jobID, estimatedLinesToRemove)
	
	return err
}

// GetJobLogsAfter returns log lines with seq > afterSeq, ordered by seq, at most limit rows.
func (db *DB) GetJobLogsAfter(jobID string, afterSeq int64, limit int) ([]JobLogLine, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 2000 {
		limit = 2000
	}

	rows, err := db.Query(
		`SELECT seq, line FROM job_logs WHERE job_id = ? AND seq > ? ORDER BY seq ASC LIMIT ?`,
		jobID, afterSeq, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []JobLogLine
	for rows.Next() {
		var line JobLogLine
		if err := rows.Scan(&line.Seq, &line.Text); err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	return out, rows.Err()
}

// DeleteJobLogs removes all stored lines for a job (optional cleanup).
func (db *DB) DeleteJobLogs(jobID string) error {
	_, err := db.Exec(`DELETE FROM job_logs WHERE job_id = ?`, jobID)
	return err
}
