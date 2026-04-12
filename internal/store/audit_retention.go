package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ListOlderThan returns audit entries older than the cutoff time.
func (r *AuditRepo) ListOlderThan(cutoff time.Time, limit int) ([]AuditEntry, error) {
	return r.listOlderThanWhere(cutoff, limit, "", nil)
}

// ListOlderThanForRoute returns audit entries older than the cutoff for a specific route.
func (r *AuditRepo) ListOlderThanForRoute(route string, cutoff time.Time, limit int) ([]AuditEntry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("audit repo is not initialized")
	}
	route = strings.TrimSpace(strings.ToLower(route))
	if route == "" {
		return nil, errors.New("route is required")
	}
	condition := "(LOWER(route_id) = ? OR LOWER(route_name) = ?)"
	return r.listOlderThanWhere(cutoff, limit, condition, []any{route, route})
}

// ListOlderThanExcludingRoutes returns audit entries older than cutoff excluding specified routes.
func (r *AuditRepo) ListOlderThanExcludingRoutes(cutoff time.Time, limit int, routes []string) ([]AuditEntry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("audit repo is not initialized")
	}
	condition, args := buildRouteExclusionCondition(routes)
	return r.listOlderThanWhere(cutoff, limit, condition, args)
}

// DeleteOlderThan deletes audit entries older than the cutoff time in batches.
func (r *AuditRepo) DeleteOlderThan(cutoff time.Time, batchSize int) (int64, error) {
	return r.deleteOlderThanWhere(cutoff, batchSize, "", nil)
}

// DeleteOlderThanForRoute deletes audit entries older than cutoff for a specific route.
func (r *AuditRepo) DeleteOlderThanForRoute(route string, cutoff time.Time, batchSize int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("audit repo is not initialized")
	}
	route = strings.TrimSpace(strings.ToLower(route))
	if route == "" {
		return 0, errors.New("route is required")
	}
	condition := "(LOWER(route_id) = ? OR LOWER(route_name) = ?)"
	return r.deleteOlderThanWhere(cutoff, batchSize, condition, []any{route, route})
}

// DeleteOlderThanExcludingRoutes deletes audit entries older than cutoff excluding specified routes.
func (r *AuditRepo) DeleteOlderThanExcludingRoutes(cutoff time.Time, batchSize int, routes []string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("audit repo is not initialized")
	}
	condition, args := buildRouteExclusionCondition(routes)
	return r.deleteOlderThanWhere(cutoff, batchSize, condition, args)
}

func (r *AuditRepo) listOlderThanWhere(cutoff time.Time, limit int, condition string, args []any) ([]AuditEntry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("audit repo is not initialized")
	}
	if cutoff.IsZero() {
		return nil, errors.New("cutoff is required")
	}
	if limit <= 0 {
		limit = 1000
	}

	cutoffRaw := cutoff.UTC().Format(time.RFC3339Nano)
	whereClause := "created_at < ?"
	queryArgs := make([]any, 0, 1+len(args)+1)
	queryArgs = append(queryArgs, cutoffRaw)
	if strings.TrimSpace(condition) != "" {
		whereClause += " AND (" + condition + ")"
		queryArgs = append(queryArgs, args...)
	}
	queryArgs = append(queryArgs, limit)

	query := auditSelectColumns + `
		 WHERE ` + whereClause + `
		 ORDER BY created_at
		 LIMIT ?`
	return r.queryEntries(query, queryArgs...)
}

func (r *AuditRepo) deleteOlderThanWhere(cutoff time.Time, batchSize int, condition string, args []any) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("audit repo is not initialized")
	}
	if cutoff.IsZero() {
		return 0, errors.New("cutoff is required")
	}
	if batchSize <= 0 {
		batchSize = 1000
	}

	cutoffRaw := cutoff.UTC().Format(time.RFC3339Nano)
	whereClause := "created_at < ?"
	baseArgs := make([]any, 0, 1+len(args))
	baseArgs = append(baseArgs, cutoffRaw)
	if strings.TrimSpace(condition) != "" {
		whereClause += " AND (" + condition + ")"
		baseArgs = append(baseArgs, args...)
	}
	var deletedTotal int64

	for {
		query := `
			DELETE FROM audit_logs
			 WHERE id IN (
			 	SELECT id
			 	  FROM audit_logs
			 	 WHERE ` + whereClause + `
			 	 ORDER BY created_at
			 	 LIMIT ?
			 )
		`
		queryArgs := append(append([]any(nil), baseArgs...), batchSize)
		result, err := r.db.Exec(query, queryArgs...)
		if err != nil {
			return deletedTotal, fmt.Errorf("delete audit logs older than cutoff: %w", err)
		}
		affected, _ := result.RowsAffected()
		if affected <= 0 {
			break
		}
		deletedTotal += affected
		if affected < int64(batchSize) {
			break
		}
	}

	return deletedTotal, nil
}

func buildRouteExclusionCondition(routes []string) (string, []any) {
	normalized := make([]string, 0, len(routes))
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		route = strings.TrimSpace(strings.ToLower(route))
		if route == "" {
			continue
		}
		if _, exists := seen[route]; exists {
			continue
		}
		seen[route] = struct{}{}
		normalized = append(normalized, route)
	}
	if len(normalized) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(normalized))
	args := make([]any, 0, len(normalized)*2)
	for range normalized {
		parts = append(parts, "(LOWER(route_id) = ? OR LOWER(route_name) = ?)")
	}
	for _, route := range normalized {
		args = append(args, route, route)
	}
	return "NOT (" + strings.Join(parts, " OR ") + ")", args
}
