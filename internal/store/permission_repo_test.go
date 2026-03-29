package store

import (
	"database/sql"
	"testing"
	"time"
)

func TestPermissionRepoCRUDAndFindList(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()

	user := createPermissionTestUser(t, s, "perm1@example.com")
	repo := s.Permissions()

	cost := int64(7)
	validFrom := time.Now().Add(-time.Hour).UTC()
	validUntil := time.Now().Add(time.Hour).UTC()
	permission := &EndpointPermission{
		UserID:       user.ID,
		RouteID:      "route-users",
		Methods:      []string{"GET", "POST"},
		Allowed:      true,
		RateLimits:   map[string]any{"algorithm": "fixed_window", "limit": 2, "window": "1s"},
		CreditCost:   &cost,
		ValidFrom:    &validFrom,
		ValidUntil:   &validUntil,
		AllowedDays:  []int{1, 2, 3, 4, 5},
		AllowedHours: []string{"09:00-18:00"},
	}
	if err := repo.Create(permission); err != nil {
		t.Fatalf("Create permission error: %v", err)
	}
	if permission.ID == "" {
		t.Fatalf("expected created permission id")
	}

	found, err := repo.FindByUserAndRoute(user.ID, "route-users")
	if err != nil {
		t.Fatalf("FindByUserAndRoute error: %v", err)
	}
	if found == nil {
		t.Fatalf("expected permission to be found")
	}
	if found.CreditCost == nil || *found.CreditCost != 7 {
		t.Fatalf("unexpected credit cost: %#v", found.CreditCost)
	}

	found.Allowed = false
	found.Methods = []string{"GET"}
	if err := repo.Update(found); err != nil {
		t.Fatalf("Update permission error: %v", err)
	}

	list, err := repo.ListByUser(user.ID)
	if err != nil {
		t.Fatalf("ListByUser permissions error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one permission in list, got %d", len(list))
	}
	if list[0].Allowed {
		t.Fatalf("expected updated allowed=false")
	}
	if len(list[0].Methods) != 1 || list[0].Methods[0] != "GET" {
		t.Fatalf("unexpected updated methods: %#v", list[0].Methods)
	}

	if err := repo.Delete(found.ID); err != nil {
		t.Fatalf("Delete permission error: %v", err)
	}
	missing, err := repo.FindByUserAndRoute(user.ID, "route-users")
	if err != nil {
		t.Fatalf("FindByUserAndRoute after delete error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected no permission after delete, got %#v", missing)
	}
}

func TestPermissionRepoBulkAssign(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()

	user := createPermissionTestUser(t, s, "perm2@example.com")
	repo := s.Permissions()

	cost := int64(3)
	err := repo.BulkAssign(user.ID, []EndpointPermission{
		{
			RouteID:      "route-a",
			Methods:      []string{"GET"},
			Allowed:      true,
			AllowedDays:  []int{1},
			AllowedHours: []string{"08:00-12:00"},
		},
		{
			RouteID:      "route-b",
			Methods:      []string{"POST"},
			Allowed:      true,
			CreditCost:   &cost,
			RateLimits:   map[string]any{"algorithm": "fixed_window", "limit": 1, "window": "1s"},
			AllowedDays:  []int{2},
			AllowedHours: []string{"09:00-18:00"},
		},
	})
	if err != nil {
		t.Fatalf("BulkAssign error: %v", err)
	}

	list, err := repo.ListByUser(user.ID)
	if err != nil {
		t.Fatalf("ListByUser after bulk assign error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(list))
	}

	err = repo.BulkAssign(user.ID, []EndpointPermission{
		{
			RouteID: "route-a",
			Methods: []string{"GET", "POST"},
			Allowed: true,
		},
	})
	if err != nil {
		t.Fatalf("BulkAssign upsert error: %v", err)
	}
	found, err := repo.FindByUserAndRoute(user.ID, "route-a")
	if err != nil {
		t.Fatalf("FindByUserAndRoute route-a error: %v", err)
	}
	if found == nil || len(found.Methods) != 2 {
		t.Fatalf("expected upserted methods on route-a, got %#v", found)
	}
}

func TestPermissionRepoValidationAndErrors(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()
	repo := s.Permissions()

	err := repo.Create(&EndpointPermission{})
	if err == nil {
		t.Fatalf("expected create validation error")
	}

	err = repo.Delete("missing")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for delete missing, got %v", err)
	}
}

func createPermissionTestUser(t *testing.T, s *Store, email string) *User {
	t.Helper()
	pw, err := HashPassword("pw")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	user := &User{
		Email:        email,
		Name:         "Permission User",
		PasswordHash: pw,
		Role:         "user",
		Status:       "active",
	}
	if err := s.Users().Create(user); err != nil {
		t.Fatalf("Create user error: %v", err)
	}
	return user
}
