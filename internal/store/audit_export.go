package store

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Export writes audit entries matching the filters to the writer in the specified format.
// Supported formats: "csv", "json", "jsonl" (default).
func (r *AuditRepo) Export(filters AuditSearchFilters, format string, w io.Writer) error {
	if r == nil || r.db == nil {
		return errors.New("audit repo is not initialized")
	}
	if w == nil {
		return errors.New("export writer is nil")
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "jsonl"
	}
	switch format {
	case "csv", "json", "jsonl":
	default:
		return errors.New("unsupported export format")
	}

	whereSQL, args := buildAuditWhere(filters)
	query := auditSelectColumns + whereSQL + ` ORDER BY created_at DESC`
	if limit := normalizeAuditExportLimit(filters.Limit); limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
		if offset := normalizeAuditOffset(filters.Offset); offset > 0 {
			query += ` OFFSET ?`
			args = append(args, offset)
		}
	} else if offset := normalizeAuditOffset(filters.Offset); offset > 0 {
		query += ` LIMIT -1 OFFSET ?`
		args = append(args, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("query audit export rows: %w", err)
	}
	defer rows.Close()

	switch format {
	case "csv":
		return exportAuditCSV(rows, w)
	case "json":
		return exportAuditJSON(rows, w)
	default:
		return exportAuditJSONL(rows, w)
	}
}

// --- Export formatters ---

func exportAuditCSV(rows *sql.Rows, w io.Writer) error {
	cw := csv.NewWriter(w)
	header := []string{
		"id", "created_at", "request_id", "route_id", "route_name", "service_name",
		"user_id", "consumer_name", "method", "host", "path", "query",
		"status_code", "latency_ms", "bytes_in", "bytes_out", "client_ip", "user_agent",
		"blocked", "block_reason", "request_headers", "request_body", "response_headers", "response_body", "error_message",
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for rows.Next() {
		entry, err := scanAuditRows(rows)
		if err != nil {
			return err
		}
		reqHeaders, _ := json.Marshal(entry.RequestHeaders)
		resHeaders, _ := json.Marshal(entry.ResponseHeaders)
		record := []string{
			entry.ID,
			entry.CreatedAt.UTC().Format(time.RFC3339Nano),
			entry.RequestID,
			entry.RouteID,
			entry.RouteName,
			entry.ServiceName,
			entry.UserID,
			entry.ConsumerName,
			entry.Method,
			entry.Host,
			entry.Path,
			entry.Query,
			fmt.Sprint(entry.StatusCode),
			fmt.Sprint(entry.LatencyMS),
			fmt.Sprint(entry.BytesIn),
			fmt.Sprint(entry.BytesOut),
			entry.ClientIP,
			entry.UserAgent,
			fmt.Sprint(entry.Blocked),
			entry.BlockReason,
			string(reqHeaders),
			entry.RequestBody,
			string(resHeaders),
			entry.ResponseBody,
			entry.ErrorMessage,
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate export rows: %w", err)
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("flush csv writer: %w", err)
	}
	return nil
}

func exportAuditJSONL(rows *sql.Rows, w io.Writer) error {
	for rows.Next() {
		entry, err := scanAuditRows(rows)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal jsonl entry: %w", err)
		}
		if _, err := w.Write(encoded); err != nil {
			return fmt.Errorf("write jsonl entry: %w", err)
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return fmt.Errorf("write jsonl newline: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate export rows: %w", err)
	}
	return nil
}

func exportAuditJSON(rows *sql.Rows, w io.Writer) error {
	if _, err := io.WriteString(w, "["); err != nil {
		return fmt.Errorf("write json export prefix: %w", err)
	}
	first := true
	for rows.Next() {
		entry, err := scanAuditRows(rows)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal json export entry: %w", err)
		}
		if !first {
			if _, err := io.WriteString(w, ","); err != nil {
				return fmt.Errorf("write json export separator: %w", err)
			}
		}
		first = false
		if _, err := w.Write(encoded); err != nil {
			return fmt.Errorf("write json export entry: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate export rows: %w", err)
	}
	if _, err := io.WriteString(w, "]"); err != nil {
		return fmt.Errorf("write json export suffix: %w", err)
	}
	return nil
}
