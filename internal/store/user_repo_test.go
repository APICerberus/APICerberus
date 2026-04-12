package store

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestUserRepoCreateFindUpdateDeleteAndStatus(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()

	repo := s.Users()
	if repo == nil {
		t.Fatalf("expected non-nil user repo")
	}

	passwordHash, err := HashPassword("super-secret")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	user := &User{
		Email:         "john@example.com",
		Name:          "John",
		Company:       "Acme",
		PasswordHash:  passwordHash,
		Role:          "user",
		Status:        "active",
		CreditBalance: 50,
		RateLimits:    map[string]any{"rps": 20},
		IPWhitelist:   []string{"203.0.113.0/24"},
		Metadata:      map[string]any{"tier": "pro"},
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	foundByID, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if foundByID == nil || foundByID.Email != "john@example.com" {
		t.Fatalf("unexpected FindByID result: %#v", foundByID)
	}

	foundByEmail, err := repo.FindByEmail("john@example.com")
	if err != nil {
		t.Fatalf("FindByEmail error: %v", err)
	}
	if foundByEmail == nil || foundByEmail.ID != user.ID {
		t.Fatalf("unexpected FindByEmail result: %#v", foundByEmail)
	}

	foundByID.Name = "John Updated"
	foundByID.Status = "suspended"
	foundByID.CreditBalance = 75
	if err := repo.Update(foundByID); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	updated, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after update error: %v", err)
	}
	if updated.Name != "John Updated" || updated.Status != "suspended" || updated.CreditBalance != 75 {
		t.Fatalf("update not persisted: %#v", updated)
	}

	if err := repo.UpdateStatus(user.ID, "active"); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	afterStatus, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after status update error: %v", err)
	}
	if afterStatus.Status != "active" {
		t.Fatalf("expected status active got %q", afterStatus.Status)
	}

	if err := repo.Delete(user.ID); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	deleted, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after delete error: %v", err)
	}
	if deleted == nil || deleted.Status != "deleted" {
		t.Fatalf("expected soft-deleted user, got %#v", deleted)
	}

	if err := repo.HardDelete(user.ID); err != nil {
		t.Fatalf("HardDelete error: %v", err)
	}
	missing, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after hard delete error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil after hard delete, got %#v", missing)
	}
}

func TestUserRepoListWithSearchFilterSortAndPagination(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()
	repo := s.Users()

	users := []User{
		{Email: "anna@example.com", Name: "Anna", Role: "user", Status: "active", CreditBalance: 10},
		{Email: "bob@example.com", Name: "Bob", Role: "user", Status: "suspended", CreditBalance: 20},
		{Email: "carol@example.com", Name: "Carol", Role: "user", Status: "active", CreditBalance: 30},
	}
	for i := range users {
		hash, err := HashPassword("x")
		if err != nil {
			t.Fatalf("HashPassword error: %v", err)
		}
		users[i].PasswordHash = hash
		if err := repo.Create(&users[i]); err != nil {
			t.Fatalf("Create user[%d] error: %v", i, err)
		}
	}

	result, err := repo.List(UserListOptions{
		Search: "a",
		Status: "active",
		Role:   "user",
		SortBy: "email",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if result.Total < 2 {
		t.Fatalf("expected at least 2 active users matching search, got %d", result.Total)
	}
	if len(result.Users) < 2 {
		t.Fatalf("expected user entries in list")
	}

	page, err := repo.List(UserListOptions{
		Role:   "user",
		SortBy: "email",
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("List pagination error: %v", err)
	}
	if len(page.Users) != 1 {
		t.Fatalf("expected one user on paginated result, got %d", len(page.Users))
	}
}

func TestPasswordHashAndVerify(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("abc123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !VerifyPassword(hash, "abc123") {
		t.Fatalf("expected password verification to succeed")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatalf("expected password verification to fail for wrong password")
	}
}

func TestInitialAdminUserCreatedOnOpen(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bootstrap-admin.db")
	cfg := &config.Config{
		Store: config.StoreConfig{
			Path:        path,
			BusyTimeout: time.Second,
			JournalMode: "WAL",
			ForeignKeys: true,
		},
	}
	s, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer s.Close()

	repo := s.Users()
	admin, err := repo.FindByEmail("admin@apicerberus.local")
	if err != nil {
		t.Fatalf("FindByEmail admin error: %v", err)
	}
	if admin == nil {
		t.Fatalf("expected initial admin user to exist")
	}
	if admin.Role != "admin" {
		t.Fatalf("expected initial admin role=admin got %q", admin.Role)
	}
	if admin.Status != "active" {
		t.Fatalf("expected initial admin status=active got %q", admin.Status)
	}
}

func TestUserRepoUpdateCreditBalance(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()

	repo := s.Users()
	hash, err := HashPassword("pw")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	user := &User{
		Email:         "credits@example.com",
		Name:          "Credits",
		PasswordHash:  hash,
		Role:          "user",
		Status:        "active",
		CreditBalance: 10,
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	balance, err := repo.UpdateCreditBalance(user.ID, -4)
	if err != nil {
		t.Fatalf("UpdateCreditBalance deduct error: %v", err)
	}
	if balance != 6 {
		t.Fatalf("expected balance 6 got %d", balance)
	}

	balance, err = repo.UpdateCreditBalance(user.ID, 9)
	if err != nil {
		t.Fatalf("UpdateCreditBalance topup error: %v", err)
	}
	if balance != 15 {
		t.Fatalf("expected balance 15 got %d", balance)
	}

	_, err = repo.UpdateCreditBalance(user.ID, -20)
	if err != ErrInsufficientCredits {
		t.Fatalf("expected ErrInsufficientCredits got %v", err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	cfg := &config.Config{
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	}
	s, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open store: %v", err)
	}
	return s
}

func TestDeleteMissingUserReturnsNoRows(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()

	err := s.Users().Delete("missing-id")
	if !errorsIsNoRows(err) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func errorsIsNoRows(err error) bool {
	return err == sql.ErrNoRows
}
