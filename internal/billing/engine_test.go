package billing

import (
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/store"
)

func TestEngineCalculateCost(t *testing.T) {
	t.Parallel()

	e := &Engine{
		cfg: config.BillingConfig{
			Enabled:     true,
			DefaultCost: 1,
			RouteCosts: map[string]int64{
				"route-1": 5,
			},
			MethodMultipliers: map[string]float64{
				"POST": 2.0,
			},
		},
	}

	getCost := e.CalculateCost(RequestMeta{
		Route:  &config.Route{ID: "route-1"},
		Method: "GET",
	})
	if getCost != 5 {
		t.Fatalf("expected GET cost=5 got %d", getCost)
	}

	postCost := e.CalculateCost(RequestMeta{
		Route:  &config.Route{ID: "route-1"},
		Method: "POST",
	})
	if postCost != 10 {
		t.Fatalf("expected POST cost=10 got %d", postCost)
	}

	defaultCost := e.CalculateCost(RequestMeta{
		Route:  &config.Route{ID: "route-x"},
		Method: "GET",
	})
	if defaultCost != 1 {
		t.Fatalf("expected default route cost=1 got %d", defaultCost)
	}
}

func TestEnginePreCheckZeroBalanceAndTestKeyBypass(t *testing.T) {
	t.Parallel()

	st := openBillingStore(t)
	defer st.Close()

	user := createBillingUser(t, st, "low-balance@example.com", 3)
	consumer := &config.Consumer{ID: user.ID, Name: user.Name}

	rejectEngine := NewEngine(st, config.BillingConfig{
		Enabled:           true,
		DefaultCost:       5,
		ZeroBalanceAction: "reject",
		TestModeEnabled:   true,
	})
	_, err := rejectEngine.PreCheck(RequestMeta{
		Consumer: consumer,
		Route:    &config.Route{ID: "route-1"},
		Method:   "GET",
	})
	if err != store.ErrInsufficientCredits {
		t.Fatalf("expected ErrInsufficientCredits got %v", err)
	}

	allowEngine := NewEngine(st, config.BillingConfig{
		Enabled:           true,
		DefaultCost:       5,
		ZeroBalanceAction: "allow_with_flag",
		TestModeEnabled:   true,
	})
	result, err := allowEngine.PreCheck(RequestMeta{
		Consumer: consumer,
		Route:    &config.Route{ID: "route-1"},
		Method:   "GET",
	})
	if err != nil {
		t.Fatalf("PreCheck allow_with_flag error: %v", err)
	}
	if !result.ZeroBalance || result.ShouldDeduct {
		t.Fatalf("expected zero-balance allow result, got %#v", result)
	}

	bypass, err := allowEngine.PreCheck(RequestMeta{
		Consumer:  consumer,
		Route:     &config.Route{ID: "route-1"},
		Method:    "GET",
		RawAPIKey: "ck_test_abc123",
	})
	if err != nil {
		t.Fatalf("PreCheck test key bypass error: %v", err)
	}
	if bypass.Cost != 0 || bypass.ShouldDeduct {
		t.Fatalf("expected bypass result with zero deduction, got %#v", bypass)
	}
}

func TestEngineDeductCreatesTransaction(t *testing.T) {
	t.Parallel()

	st := openBillingStore(t)
	defer st.Close()

	user := createBillingUser(t, st, "deduct@example.com", 20)
	engine := NewEngine(st, config.BillingConfig{
		Enabled:           true,
		DefaultCost:       4,
		ZeroBalanceAction: "reject",
		TestModeEnabled:   true,
	})
	consumer := &config.Consumer{ID: user.ID, Name: user.Name}

	check, err := engine.PreCheck(RequestMeta{
		Consumer: consumer,
		Route:    &config.Route{ID: "route-deduct"},
		Method:   "GET",
	})
	if err != nil {
		t.Fatalf("PreCheck error: %v", err)
	}
	if !check.ShouldDeduct || check.Cost != 4 {
		t.Fatalf("unexpected precheck result: %#v", check)
	}

	newBalance, err := engine.Deduct(check, "req-1", "route-deduct")
	if err != nil {
		t.Fatalf("Deduct error: %v", err)
	}
	if newBalance != 16 {
		t.Fatalf("expected new balance 16 got %d", newBalance)
	}

	list, err := st.Credits().ListByUser(user.ID, store.CreditListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListByUser credit tx error: %v", err)
	}
	if list.Total != 1 || len(list.Transactions) != 1 {
		t.Fatalf("expected one credit transaction, got total=%d len=%d", list.Total, len(list.Transactions))
	}
	tx := list.Transactions[0]
	if tx.Amount != -4 || tx.BalanceAfter != 16 || tx.RequestID != "req-1" {
		t.Fatalf("unexpected credit transaction: %#v", tx)
	}
}

func openBillingStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(&config.Config{
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	})
	if err != nil {
		t.Fatalf("open store error: %v", err)
	}
	return st
}

func createBillingUser(t *testing.T, st *store.Store, email string, balance int64) *store.User {
	t.Helper()
	pw, err := store.HashPassword("pw")
	if err != nil {
		t.Fatalf("hash password error: %v", err)
	}
	user := &store.User{
		Email:         email,
		Name:          "Billing User",
		PasswordHash:  pw,
		Role:          "user",
		Status:        "active",
		CreditBalance: balance,
	}
	if err := st.Users().Create(user); err != nil {
		t.Fatalf("create billing user error: %v", err)
	}
	return user
}

// Test Enabled function
func TestEngineEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		engine   *Engine
		expected bool
	}{
		{
			name:     "nil engine",
			engine:   nil,
			expected: false,
		},
		{
			name: "enabled config",
			engine: &Engine{
				cfg: config.BillingConfig{Enabled: true},
			},
			expected: true,
		},
		{
			name: "disabled config",
			engine: &Engine{
				cfg: config.BillingConfig{Enabled: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.engine.Enabled()
			if result != tt.expected {
				t.Errorf("Enabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test Enabled with store integration
func TestEngineEnabled_Integration(t *testing.T) {
	t.Parallel()

	st := openBillingStore(t)
	defer st.Close()

	// Enabled engine
	enabledEngine := NewEngine(st, config.BillingConfig{Enabled: true})
	if !enabledEngine.Enabled() {
		t.Error("Expected enabled engine to return true")
	}

	// Disabled engine
	disabledEngine := NewEngine(st, config.BillingConfig{Enabled: false})
	if disabledEngine.Enabled() {
		t.Error("Expected disabled engine to return false")
	}

	// Nil engine
	var nilEngine *Engine
	if nilEngine.Enabled() {
		t.Error("Expected nil engine to return false")
	}
}

