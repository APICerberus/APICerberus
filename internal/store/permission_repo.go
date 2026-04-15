package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/APICerberus/APICerebrus/internal/pkg/uuid"
)

type EndpointPermission struct {
	ID           string
	UserID       string
	RouteID      string
	Methods      []string
	Allowed      bool
	RateLimits   map[string]any
	CreditCost   *int64
	ValidFrom    *time.Time
	ValidUntil   *time.Time
	AllowedDays  []int
	AllowedHours []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PermissionRepo struct {
	db  DB
	now func() time.Time
}

func (s *Store) Permissions() *PermissionRepo {
	if s == nil || s.db == nil {
		return nil
	}
	return &PermissionRepo{
		db:  s.db,
		now: time.Now,
	}
}

func (r *PermissionRepo) Create(permission *EndpointPermission) error {
	if r == nil || r.db == nil {
		return errors.New("permission repo is not initialized")
	}
	if permission == nil {
		return errors.New("permission is nil")
	}
	if err := validatePermissionInput(permission); err != nil {
		return err
	}
	if strings.TrimSpace(permission.ID) == "" {
		id, err := uuid.NewString()
		if err != nil {
			return err
		}
		permission.ID = id
	}
	now := r.now().UTC()
	if permission.CreatedAt.IsZero() {
		permission.CreatedAt = now
	}
	permission.UpdatedAt = now

	methodsJSON, err := marshalJSON(normalizeMethods(permission.Methods), "[]")
	if err != nil {
		return err
	}
	rateLimitsJSON, err := marshalJSON(permission.RateLimits, "{}")
	if err != nil {
		return err
	}
	allowedDaysJSON, err := marshalJSON(permission.AllowedDays, "[]")
	if err != nil {
		return err
	}
	allowedHoursJSON, err := marshalJSON(permission.AllowedHours, "[]")
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO endpoint_permissions(
			id, user_id, route_id, methods, allowed, rate_limits, credit_cost,
			valid_from, valid_until, allowed_days, allowed_hours, created_at, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		permission.ID,
		strings.TrimSpace(permission.UserID),
		strings.TrimSpace(permission.RouteID),
		methodsJSON,
		boolToInt(permission.Allowed),
		rateLimitsJSON,
		creditCostToRaw(permission.CreditCost),
		timePtrToRaw(permission.ValidFrom),
		timePtrToRaw(permission.ValidUntil),
		allowedDaysJSON,
		allowedHoursJSON,
		permission.CreatedAt.UTC().Format(time.RFC3339Nano),
		permission.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert endpoint permission: %w", err)
	}
	return nil
}

func (r *PermissionRepo) Update(permission *EndpointPermission) error {
	if r == nil || r.db == nil {
		return errors.New("permission repo is not initialized")
	}
	if permission == nil {
		return errors.New("permission is nil")
	}
	permission.ID = strings.TrimSpace(permission.ID)
	if permission.ID == "" {
		return errors.New("permission id is required")
	}
	if err := validatePermissionInput(permission); err != nil {
		return err
	}
	permission.UpdatedAt = r.now().UTC()

	methodsJSON, err := marshalJSON(normalizeMethods(permission.Methods), "[]")
	if err != nil {
		return err
	}
	rateLimitsJSON, err := marshalJSON(permission.RateLimits, "{}")
	if err != nil {
		return err
	}
	allowedDaysJSON, err := marshalJSON(permission.AllowedDays, "[]")
	if err != nil {
		return err
	}
	allowedHoursJSON, err := marshalJSON(permission.AllowedHours, "[]")
	if err != nil {
		return err
	}

	result, err := r.db.Exec(`
		UPDATE endpoint_permissions
		   SET user_id = ?, route_id = ?, methods = ?, allowed = ?, rate_limits = ?, credit_cost = ?,
		       valid_from = ?, valid_until = ?, allowed_days = ?, allowed_hours = ?, updated_at = ?
		 WHERE id = ?
	`,
		strings.TrimSpace(permission.UserID),
		strings.TrimSpace(permission.RouteID),
		methodsJSON,
		boolToInt(permission.Allowed),
		rateLimitsJSON,
		creditCostToRaw(permission.CreditCost),
		timePtrToRaw(permission.ValidFrom),
		timePtrToRaw(permission.ValidUntil),
		allowedDaysJSON,
		allowedHoursJSON,
		permission.UpdatedAt.UTC().Format(time.RFC3339Nano),
		permission.ID,
	)
	if err != nil {
		return fmt.Errorf("update endpoint permission: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PermissionRepo) Delete(id string) error {
	if r == nil || r.db == nil {
		return errors.New("permission repo is not initialized")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("permission id is required")
	}
	result, err := r.db.Exec(`DELETE FROM endpoint_permissions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete endpoint permission: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PermissionRepo) FindByUserAndRoute(userID, routeID string) (*EndpointPermission, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("permission repo is not initialized")
	}
	userID = strings.TrimSpace(userID)
	routeID = strings.TrimSpace(routeID)
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if routeID == "" {
		return nil, errors.New("route id is required")
	}

	row := r.db.QueryRow(`
		SELECT id, user_id, route_id, methods, allowed, rate_limits, credit_cost,
		       valid_from, valid_until, allowed_days, allowed_hours, created_at, updated_at
		  FROM endpoint_permissions
		 WHERE user_id = ? AND route_id = ?
		 LIMIT 1
	`, userID, routeID)
	permission, err := scanPermission(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return permission, nil
}

func (r *PermissionRepo) ListByUser(userID string) ([]EndpointPermission, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("permission repo is not initialized")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	rows, err := r.db.Query(`
		SELECT id, user_id, route_id, methods, allowed, rate_limits, credit_cost,
		       valid_from, valid_until, allowed_days, allowed_hours, created_at, updated_at
		  FROM endpoint_permissions
		 WHERE user_id = ?
		 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list endpoint permissions: %w", err)
	}
	defer rows.Close()

	out := make([]EndpointPermission, 0)
	for rows.Next() {
		permission, err := scanPermissionRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate endpoint permissions: %w", err)
	}
	return out, nil
}

func (r *PermissionRepo) BulkAssign(userID string, permissions []EndpointPermission) error {
	if r == nil || r.db == nil {
		return errors.New("permission repo is not initialized")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user id is required")
	}

	return r.withTx(context.Background(), func(tx *sql.Tx) error {
		for _, permission := range permissions {
			permission.UserID = userID
			if err := validatePermissionInput(&permission); err != nil {
				return err
			}
			if strings.TrimSpace(permission.ID) == "" {
				id, err := uuid.NewString()
				if err != nil {
					return err
				}
				permission.ID = id
			}
			now := r.now().UTC()
			if permission.CreatedAt.IsZero() {
				permission.CreatedAt = now
			}
			permission.UpdatedAt = now

			methodsJSON, err := marshalJSON(normalizeMethods(permission.Methods), "[]")
			if err != nil {
				return err
			}
			rateLimitsJSON, err := marshalJSON(permission.RateLimits, "{}")
			if err != nil {
				return err
			}
			allowedDaysJSON, err := marshalJSON(permission.AllowedDays, "[]")
			if err != nil {
				return err
			}
			allowedHoursJSON, err := marshalJSON(permission.AllowedHours, "[]")
			if err != nil {
				return err
			}

			_, err = tx.Exec(`
				INSERT INTO endpoint_permissions(
					id, user_id, route_id, methods, allowed, rate_limits, credit_cost,
					valid_from, valid_until, allowed_days, allowed_hours, created_at, updated_at
				) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(user_id, route_id) DO UPDATE SET
					methods = excluded.methods,
					allowed = excluded.allowed,
					rate_limits = excluded.rate_limits,
					credit_cost = excluded.credit_cost,
					valid_from = excluded.valid_from,
					valid_until = excluded.valid_until,
					allowed_days = excluded.allowed_days,
					allowed_hours = excluded.allowed_hours,
					updated_at = excluded.updated_at
			`,
				permission.ID,
				userID,
				strings.TrimSpace(permission.RouteID),
				methodsJSON,
				boolToInt(permission.Allowed),
				rateLimitsJSON,
				creditCostToRaw(permission.CreditCost),
				timePtrToRaw(permission.ValidFrom),
				timePtrToRaw(permission.ValidUntil),
				allowedDaysJSON,
				allowedHoursJSON,
				permission.CreatedAt.UTC().Format(time.RFC3339Nano),
				permission.UpdatedAt.UTC().Format(time.RFC3339Nano),
			)
			if err != nil {
				return fmt.Errorf("bulk assign endpoint permission: %w", err)
			}
		}
		return nil
	})
}

func (r *PermissionRepo) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func validatePermissionInput(permission *EndpointPermission) error {
	if permission == nil {
		return errors.New("permission is nil")
	}
	if strings.TrimSpace(permission.UserID) == "" {
		return errors.New("permission user id is required")
	}
	if strings.TrimSpace(permission.RouteID) == "" {
		return errors.New("permission route id is required")
	}
	return nil
}

func normalizeMethods(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, item := range in {
		method := strings.ToUpper(strings.TrimSpace(item))
		if method == "" {
			continue
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	return out
}

func scanPermission(row *sql.Row) (*EndpointPermission, error) {
	var (
		permission                              EndpointPermission
		methodsRaw, rateLimitsRaw               string
		creditCostRaw, validFromRaw, validToRaw string
		allowedDaysRaw, allowedHoursRaw         string
		createdRaw, updatedRaw                  string
		allowedInt                              int
	)
	if err := row.Scan(
		&permission.ID,
		&permission.UserID,
		&permission.RouteID,
		&methodsRaw,
		&allowedInt,
		&rateLimitsRaw,
		&creditCostRaw,
		&validFromRaw,
		&validToRaw,
		&allowedDaysRaw,
		&allowedHoursRaw,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return nil, err
	}
	permission.Allowed = allowedInt != 0
	if err := decodePermissionFields(&permission, methodsRaw, rateLimitsRaw, creditCostRaw, validFromRaw, validToRaw, allowedDaysRaw, allowedHoursRaw, createdRaw, updatedRaw); err != nil {
		return nil, err
	}
	return &permission, nil
}

func scanPermissionRows(rows *sql.Rows) (*EndpointPermission, error) {
	var (
		permission                              EndpointPermission
		methodsRaw, rateLimitsRaw               string
		creditCostRaw, validFromRaw, validToRaw string
		allowedDaysRaw, allowedHoursRaw         string
		createdRaw, updatedRaw                  string
		allowedInt                              int
	)
	if err := rows.Scan(
		&permission.ID,
		&permission.UserID,
		&permission.RouteID,
		&methodsRaw,
		&allowedInt,
		&rateLimitsRaw,
		&creditCostRaw,
		&validFromRaw,
		&validToRaw,
		&allowedDaysRaw,
		&allowedHoursRaw,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return nil, err
	}
	permission.Allowed = allowedInt != 0
	if err := decodePermissionFields(&permission, methodsRaw, rateLimitsRaw, creditCostRaw, validFromRaw, validToRaw, allowedDaysRaw, allowedHoursRaw, createdRaw, updatedRaw); err != nil {
		return nil, err
	}
	return &permission, nil
}

func decodePermissionFields(permission *EndpointPermission, methodsRaw, rateLimitsRaw, creditCostRaw, validFromRaw, validToRaw, allowedDaysRaw, allowedHoursRaw, createdRaw, updatedRaw string) error {
	if permission == nil {
		return errors.New("permission is nil")
	}

	if strings.TrimSpace(methodsRaw) == "" {
		methodsRaw = "[]"
	}
	permission.Methods = []string{}
	if err := json.Unmarshal([]byte(methodsRaw), &permission.Methods); err != nil {
		return fmt.Errorf("decode permission methods: %w", err)
	}
	permission.Methods = normalizeMethods(permission.Methods)

	if strings.TrimSpace(rateLimitsRaw) == "" {
		rateLimitsRaw = "{}"
	}
	permission.RateLimits = map[string]any{}
	if err := json.Unmarshal([]byte(rateLimitsRaw), &permission.RateLimits); err != nil {
		return fmt.Errorf("decode permission rate_limits: %w", err)
	}

	if strings.TrimSpace(creditCostRaw) != "" {
		value, err := strconv.ParseInt(strings.TrimSpace(creditCostRaw), 10, 64)
		if err != nil {
			return fmt.Errorf("decode permission credit_cost: %w", err)
		}
		permission.CreditCost = &value
	}

	if strings.TrimSpace(validFromRaw) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, validFromRaw)
		if err != nil {
			return fmt.Errorf("decode permission valid_from: %w", err)
		}
		permission.ValidFrom = &parsed
	}
	if strings.TrimSpace(validToRaw) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, validToRaw)
		if err != nil {
			return fmt.Errorf("decode permission valid_until: %w", err)
		}
		permission.ValidUntil = &parsed
	}

	if strings.TrimSpace(allowedDaysRaw) == "" {
		allowedDaysRaw = "[]"
	}
	permission.AllowedDays = []int{}
	if err := json.Unmarshal([]byte(allowedDaysRaw), &permission.AllowedDays); err != nil {
		return fmt.Errorf("decode permission allowed_days: %w", err)
	}
	if strings.TrimSpace(allowedHoursRaw) == "" {
		allowedHoursRaw = "[]"
	}
	permission.AllowedHours = []string{}
	if err := json.Unmarshal([]byte(allowedHoursRaw), &permission.AllowedHours); err != nil {
		return fmt.Errorf("decode permission allowed_hours: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return fmt.Errorf("decode permission created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return fmt.Errorf("decode permission updated_at: %w", err)
	}
	permission.CreatedAt = createdAt
	permission.UpdatedAt = updatedAt

	return nil
}

func creditCostToRaw(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func timePtrToRaw(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
