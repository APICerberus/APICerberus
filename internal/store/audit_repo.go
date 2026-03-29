package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/APICerberus/APICerebrus/internal/pkg/uuid"
)

type AuditEntry struct {
	ID              string
	RequestID       string
	RouteID         string
	RouteName       string
	ServiceName     string
	UserID          string
	ConsumerName    string
	Method          string
	Host            string
	Path            string
	Query           string
	StatusCode      int
	LatencyMS       int64
	BytesIn         int64
	BytesOut        int64
	ClientIP        string
	UserAgent       string
	Blocked         bool
	BlockReason     string
	RequestHeaders  map[string]any
	RequestBody     string
	ResponseHeaders map[string]any
	ResponseBody    string
	ErrorMessage    string
	CreatedAt       time.Time
}

type AuditListOptions struct {
	UserID    string
	RouteID   string
	Method    string
	StatusMin int
	StatusMax int
	Limit     int
	Offset    int
}

type AuditListResult struct {
	Entries []AuditEntry
	Total   int
}

type AuditRepo struct {
	db  *sql.DB
	now func() time.Time
}

func (s *Store) Audits() *AuditRepo {
	if s == nil || s.db == nil {
		return nil
	}
	return &AuditRepo{
		db:  s.db,
		now: time.Now,
	}
}

func (r *AuditRepo) BatchInsert(entries []AuditEntry) error {
	if r == nil || r.db == nil {
		return errors.New("audit repo is not initialized")
	}
	if len(entries) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin audit batch insert: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO audit_logs(
			id, request_id, route_id, route_name, service_name,
			user_id, consumer_name, method, host, path, query,
			status_code, latency_ms, bytes_in, bytes_out,
			client_ip, user_agent, blocked, block_reason,
			request_headers, request_body, response_headers, response_body,
			error_message, created_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare audit insert: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		if strings.TrimSpace(entry.ID) == "" {
			id, genErr := uuid.NewString()
			if genErr != nil {
				_ = tx.Rollback()
				return genErr
			}
			entry.ID = id
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = r.now().UTC()
		}

		requestHeaders, marshalErr := marshalJSON(entry.RequestHeaders, "{}")
		if marshalErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal audit request headers: %w", marshalErr)
		}
		responseHeaders, marshalErr := marshalJSON(entry.ResponseHeaders, "{}")
		if marshalErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal audit response headers: %w", marshalErr)
		}

		blocked := 0
		if entry.Blocked {
			blocked = 1
		}

		if _, err := stmt.Exec(
			entry.ID,
			strings.TrimSpace(entry.RequestID),
			strings.TrimSpace(entry.RouteID),
			strings.TrimSpace(entry.RouteName),
			strings.TrimSpace(entry.ServiceName),
			strings.TrimSpace(entry.UserID),
			strings.TrimSpace(entry.ConsumerName),
			strings.TrimSpace(strings.ToUpper(entry.Method)),
			strings.TrimSpace(entry.Host),
			strings.TrimSpace(entry.Path),
			strings.TrimSpace(entry.Query),
			entry.StatusCode,
			entry.LatencyMS,
			entry.BytesIn,
			entry.BytesOut,
			strings.TrimSpace(entry.ClientIP),
			strings.TrimSpace(entry.UserAgent),
			blocked,
			strings.TrimSpace(entry.BlockReason),
			requestHeaders,
			entry.RequestBody,
			responseHeaders,
			entry.ResponseBody,
			entry.ErrorMessage,
			entry.CreatedAt.UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert audit entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit audit batch insert: %w", err)
	}
	return nil
}

func (r *AuditRepo) FindByID(id string) (*AuditEntry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("audit repo is not initialized")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("audit id is required")
	}

	row := r.db.QueryRow(`
		SELECT id, request_id, route_id, route_name, service_name,
		       user_id, consumer_name, method, host, path, query,
		       status_code, latency_ms, bytes_in, bytes_out,
		       client_ip, user_agent, blocked, block_reason,
		       request_headers, request_body, response_headers, response_body,
		       error_message, created_at
		  FROM audit_logs
		 WHERE id = ?
	`, id)
	entry, err := scanAuditRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return entry, nil
}

func (r *AuditRepo) List(opts AuditListOptions) (*AuditListResult, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("audit repo is not initialized")
	}

	where := make([]string, 0, 5)
	args := make([]any, 0, 8)

	if value := strings.TrimSpace(opts.UserID); value != "" {
		where = append(where, "user_id = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(opts.RouteID); value != "" {
		where = append(where, "route_id = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(opts.Method); value != "" {
		where = append(where, "method = ?")
		args = append(args, strings.ToUpper(value))
	}
	if opts.StatusMin > 0 {
		where = append(where, "status_code >= ?")
		args = append(args, opts.StatusMin)
	}
	if opts.StatusMax > 0 {
		where = append(where, "status_code <= ?")
		args = append(args, opts.StatusMax)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	limit := opts.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	countSQL := "SELECT COUNT(*) FROM audit_logs" + whereSQL
	var total int
	if err := r.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count audit logs: %w", err)
	}

	query := `
		SELECT id, request_id, route_id, route_name, service_name,
		       user_id, consumer_name, method, host, path, query,
		       status_code, latency_ms, bytes_in, bytes_out,
		       client_ip, user_agent, blocked, block_reason,
		       request_headers, request_body, response_headers, response_body,
		       error_message, created_at
		  FROM audit_logs` + whereSQL + `
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`
	queryArgs := append(append([]any(nil), args...), limit, offset)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]AuditEntry, 0, limit)
	for rows.Next() {
		entry, scanErr := scanAuditRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}

	return &AuditListResult{
		Entries: items,
		Total:   total,
	}, nil
}

func scanAuditRow(row *sql.Row) (*AuditEntry, error) {
	var (
		entry                                 AuditEntry
		blockedInt                            int
		requestHeadersRaw, responseHeadersRaw string
		createdAtRaw                          string
	)
	if err := row.Scan(
		&entry.ID,
		&entry.RequestID,
		&entry.RouteID,
		&entry.RouteName,
		&entry.ServiceName,
		&entry.UserID,
		&entry.ConsumerName,
		&entry.Method,
		&entry.Host,
		&entry.Path,
		&entry.Query,
		&entry.StatusCode,
		&entry.LatencyMS,
		&entry.BytesIn,
		&entry.BytesOut,
		&entry.ClientIP,
		&entry.UserAgent,
		&blockedInt,
		&entry.BlockReason,
		&requestHeadersRaw,
		&entry.RequestBody,
		&responseHeadersRaw,
		&entry.ResponseBody,
		&entry.ErrorMessage,
		&createdAtRaw,
	); err != nil {
		return nil, err
	}
	if err := decodeAuditFields(&entry, blockedInt, requestHeadersRaw, responseHeadersRaw, createdAtRaw); err != nil {
		return nil, err
	}
	return &entry, nil
}

func scanAuditRows(rows *sql.Rows) (*AuditEntry, error) {
	var (
		entry                                 AuditEntry
		blockedInt                            int
		requestHeadersRaw, responseHeadersRaw string
		createdAtRaw                          string
	)
	if err := rows.Scan(
		&entry.ID,
		&entry.RequestID,
		&entry.RouteID,
		&entry.RouteName,
		&entry.ServiceName,
		&entry.UserID,
		&entry.ConsumerName,
		&entry.Method,
		&entry.Host,
		&entry.Path,
		&entry.Query,
		&entry.StatusCode,
		&entry.LatencyMS,
		&entry.BytesIn,
		&entry.BytesOut,
		&entry.ClientIP,
		&entry.UserAgent,
		&blockedInt,
		&entry.BlockReason,
		&requestHeadersRaw,
		&entry.RequestBody,
		&responseHeadersRaw,
		&entry.ResponseBody,
		&entry.ErrorMessage,
		&createdAtRaw,
	); err != nil {
		return nil, err
	}
	if err := decodeAuditFields(&entry, blockedInt, requestHeadersRaw, responseHeadersRaw, createdAtRaw); err != nil {
		return nil, err
	}
	return &entry, nil
}

func decodeAuditFields(entry *AuditEntry, blockedInt int, requestHeadersRaw, responseHeadersRaw, createdAtRaw string) error {
	if entry == nil {
		return errors.New("audit entry is nil")
	}
	entry.Blocked = blockedInt == 1

	entry.RequestHeaders = map[string]any{}
	if strings.TrimSpace(requestHeadersRaw) == "" {
		requestHeadersRaw = "{}"
	}
	if err := json.Unmarshal([]byte(requestHeadersRaw), &entry.RequestHeaders); err != nil {
		return fmt.Errorf("decode audit request_headers: %w", err)
	}

	entry.ResponseHeaders = map[string]any{}
	if strings.TrimSpace(responseHeadersRaw) == "" {
		responseHeadersRaw = "{}"
	}
	if err := json.Unmarshal([]byte(responseHeadersRaw), &entry.ResponseHeaders); err != nil {
		return fmt.Errorf("decode audit response_headers: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return fmt.Errorf("decode audit created_at: %w", err)
	}
	entry.CreatedAt = createdAt
	return nil
}
