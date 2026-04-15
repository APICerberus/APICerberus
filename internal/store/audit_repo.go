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

type AuditRepo struct {
	db  DB
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

// BatchInsert inserts multiple audit entries in a single transaction.
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

// FindByID retrieves a single audit entry by its ID.
func (r *AuditRepo) FindByID(id string) (*AuditEntry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("audit repo is not initialized")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("audit id is required")
	}

	row := r.db.QueryRow(auditSelectColumns+`
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

// List returns a paginated list of audit entries.
// Delegates to Search with basic filter mapping.
func (r *AuditRepo) List(opts AuditListOptions) (*AuditListResult, error) {
	return r.Search(AuditSearchFilters{
		UserID:    opts.UserID,
		Route:     opts.RouteID,
		Method:    opts.Method,
		StatusMin: opts.StatusMin,
		StatusMax: opts.StatusMax,
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	})
}

// DeleteByIDs deletes audit entries by their IDs.
func (r *AuditRepo) DeleteByIDs(ids []string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("audit repo is not initialized")
	}
	normalized := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(normalized)), ",")
	query := "DELETE FROM audit_logs WHERE id IN (" + placeholders + ")"
	args := make([]any, 0, len(normalized))
	for _, id := range normalized {
		args = append(args, id)
	}
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete audit logs by ids: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

// queryEntries is a shared helper for executing audit queries and scanning results.
func (r *AuditRepo) queryEntries(query string, args ...any) ([]AuditEntry, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]AuditEntry, 0, 32)
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
	return items, nil
}

// --- Scanning helpers ---

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
