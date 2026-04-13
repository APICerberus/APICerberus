package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/APICerberus/APICerebrus/internal/pkg/uuid"
)

type CreditTransaction struct {
	ID            string
	UserID        string
	Type          string
	Amount        int64
	BalanceBefore int64
	BalanceAfter  int64
	Description   string
	RequestID     string
	RouteID       string
	CreatedAt     time.Time
}

type CreditListOptions struct {
	Type   string
	Limit  int
	Offset int
}

type CreditListResult struct {
	Transactions []CreditTransaction
	Total        int
}

type TopConsumer struct {
	UserID   string
	Email    string
	Name     string
	Consumed int64
}

type CreditOverviewStats struct {
	TotalDistributed int64
	TotalConsumed    int64
	TopConsumers     []TopConsumer
}

type CreditRepo struct {
	db  *sql.DB
	now func() time.Time
}

func (s *Store) Credits() *CreditRepo {
	if s == nil || s.db == nil {
		return nil
	}
	return &CreditRepo{
		db:  s.db,
		now: time.Now,
	}
}

func (r *CreditRepo) Create(txn *CreditTransaction) error {
	if r == nil || r.db == nil {
		return errors.New("credit repo is not initialized")
	}
	if txn == nil {
		return errors.New("credit transaction is nil")
	}
	txn.UserID = strings.TrimSpace(txn.UserID)
	if txn.UserID == "" {
		return errors.New("credit transaction user id is required")
	}
	txn.Type = strings.TrimSpace(strings.ToLower(txn.Type))
	if txn.Type == "" {
		txn.Type = "consume"
	}
	if strings.TrimSpace(txn.ID) == "" {
		id, err := uuid.NewString()
		if err != nil {
			return err
		}
		txn.ID = id
	}
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = r.now().UTC()
	}

	_, err := r.db.Exec(`
		INSERT INTO credit_transactions(
			id, user_id, type, amount, balance_before, balance_after,
			description, request_id, route_id, created_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		txn.ID,
		txn.UserID,
		txn.Type,
		txn.Amount,
		txn.BalanceBefore,
		txn.BalanceAfter,
		strings.TrimSpace(txn.Description),
		strings.TrimSpace(txn.RequestID),
		strings.TrimSpace(txn.RouteID),
		txn.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert credit transaction: %w", err)
	}
	return nil
}

// CreateTx creates a credit transaction within an existing transaction.
func (r *CreditRepo) CreateTx(tx *sql.Tx, txn *CreditTransaction) error {
	if r == nil || tx == nil {
		return errors.New("credit repo or transaction is nil")
	}
	if txn == nil {
		return errors.New("credit transaction is nil")
	}
	txn.UserID = strings.TrimSpace(txn.UserID)
	if txn.UserID == "" {
		return errors.New("credit transaction user id is required")
	}
	txn.Type = strings.TrimSpace(strings.ToLower(txn.Type))
	if txn.Type == "" {
		txn.Type = "consume"
	}
	if strings.TrimSpace(txn.ID) == "" {
		id, err := uuid.NewString()
		if err != nil {
			return err
		}
		txn.ID = id
	}
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = r.now().UTC()
	}

	_, err := tx.Exec(`
		INSERT INTO credit_transactions(
			id, user_id, type, amount, balance_before, balance_after,
			description, request_id, route_id, created_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		txn.ID,
		txn.UserID,
		txn.Type,
		txn.Amount,
		txn.BalanceBefore,
		txn.BalanceAfter,
		strings.TrimSpace(txn.Description),
		strings.TrimSpace(txn.RequestID),
		strings.TrimSpace(txn.RouteID),
		txn.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert credit transaction: %w", err)
	}
	return nil
}

func (r *CreditRepo) ListByUser(userID string, opts CreditListOptions) (*CreditListResult, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("credit repo is not initialized")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	where := []string{"user_id = ?"}
	args := []any{userID}

	if txnType := strings.TrimSpace(strings.ToLower(opts.Type)); txnType != "" {
		where = append(where, "type = ?")
		args = append(args, txnType)
	}
	whereSQL := " WHERE " + strings.Join(where, " AND ")

	limit := opts.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	countSQL := "SELECT COUNT(*) FROM credit_transactions" + whereSQL
	var total int
	if err := r.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count credit transactions: %w", err)
	}

	query := `
		SELECT id, user_id, type, amount, balance_before, balance_after,
		       description, request_id, route_id, created_at
		  FROM credit_transactions` + whereSQL + `
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`
	queryArgs := append(append([]any(nil), args...), limit, offset)
	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list credit transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]CreditTransaction, 0, limit)
	for rows.Next() {
		txn, err := scanCreditTransactionRows(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *txn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credit transactions: %w", err)
	}

	return &CreditListResult{
		Transactions: transactions,
		Total:        total,
	}, nil
}

func (r *CreditRepo) OverviewStats() (*CreditOverviewStats, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("credit repo is not initialized")
	}

	stats := &CreditOverviewStats{
		TopConsumers: []TopConsumer{},
	}
	if err := r.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)
		  FROM credit_transactions
	`).Scan(&stats.TotalDistributed, &stats.TotalConsumed); err != nil {
		return nil, fmt.Errorf("query credit overview totals: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT u.id, u.email, u.name, COALESCE(SUM(-c.amount), 0) AS consumed
		  FROM users u
		  JOIN credit_transactions c ON c.user_id = u.id
		 WHERE c.amount < 0
		 GROUP BY u.id, u.email, u.name
		 ORDER BY consumed DESC
		 LIMIT 10
	`)
	if err != nil {
		return nil, fmt.Errorf("query credit overview top consumers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item TopConsumer
		if err := rows.Scan(&item.UserID, &item.Email, &item.Name, &item.Consumed); err != nil {
			return nil, fmt.Errorf("scan top consumer: %w", err)
		}
		stats.TopConsumers = append(stats.TopConsumers, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top consumers: %w", err)
	}

	return stats, nil
}

func scanCreditTransactionRows(rows *sql.Rows) (*CreditTransaction, error) {
	var (
		txn          CreditTransaction
		createdAtRaw string
	)
	if err := rows.Scan(
		&txn.ID,
		&txn.UserID,
		&txn.Type,
		&txn.Amount,
		&txn.BalanceBefore,
		&txn.BalanceAfter,
		&txn.Description,
		&txn.RequestID,
		&txn.RouteID,
		&createdAtRaw,
	); err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("decode credit transaction created_at: %w", err)
	}
	txn.CreatedAt = createdAt
	return &txn, nil
}
