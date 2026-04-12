package store

import "testing"

func TestCreditRepoCreateListAndOverview(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()

	users := s.Users()
	credits := s.Credits()

	pw, err := HashPassword("pw")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	u1 := &User{Email: "u1@example.com", Name: "U1", PasswordHash: pw, Role: "user", Status: "active"}
	u2 := &User{Email: "u2@example.com", Name: "U2", PasswordHash: pw, Role: "user", Status: "active"}
	if err := users.Create(u1); err != nil {
		t.Fatalf("create user1 error: %v", err)
	}
	if err := users.Create(u2); err != nil {
		t.Fatalf("create user2 error: %v", err)
	}

	if err := credits.Create(&CreditTransaction{
		UserID:        u1.ID,
		Type:          "topup",
		Amount:        100,
		BalanceBefore: 0,
		BalanceAfter:  100,
		Description:   "initial topup",
	}); err != nil {
		t.Fatalf("create topup tx error: %v", err)
	}
	if err := credits.Create(&CreditTransaction{
		UserID:        u1.ID,
		Type:          "consume",
		Amount:        -30,
		BalanceBefore: 100,
		BalanceAfter:  70,
		Description:   "call route-1",
		RouteID:       "route-1",
	}); err != nil {
		t.Fatalf("create consume tx u1 error: %v", err)
	}
	if err := credits.Create(&CreditTransaction{
		UserID:        u2.ID,
		Type:          "consume",
		Amount:        -50,
		BalanceBefore: 80,
		BalanceAfter:  30,
		Description:   "call route-2",
		RouteID:       "route-2",
	}); err != nil {
		t.Fatalf("create consume tx u2 error: %v", err)
	}

	list, err := credits.ListByUser(u1.ID, CreditListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListByUser error: %v", err)
	}
	if list.Total != 2 || len(list.Transactions) != 2 {
		t.Fatalf("unexpected list result: total=%d len=%d", list.Total, len(list.Transactions))
	}

	consumes, err := credits.ListByUser(u1.ID, CreditListOptions{Type: "consume", Limit: 10})
	if err != nil {
		t.Fatalf("ListByUser consume filter error: %v", err)
	}
	if consumes.Total != 1 || len(consumes.Transactions) != 1 {
		t.Fatalf("unexpected consume filtered list: total=%d len=%d", consumes.Total, len(consumes.Transactions))
	}
	if consumes.Transactions[0].Amount != -30 {
		t.Fatalf("expected consume amount -30 got %d", consumes.Transactions[0].Amount)
	}

	stats, err := credits.OverviewStats()
	if err != nil {
		t.Fatalf("OverviewStats error: %v", err)
	}
	if stats.TotalDistributed != 100 {
		t.Fatalf("expected TotalDistributed=100 got %d", stats.TotalDistributed)
	}
	if stats.TotalConsumed != 80 {
		t.Fatalf("expected TotalConsumed=80 got %d", stats.TotalConsumed)
	}
	if len(stats.TopConsumers) == 0 || stats.TopConsumers[0].UserID != u2.ID || stats.TopConsumers[0].Consumed != 50 {
		t.Fatalf("unexpected TopConsumers: %#v", stats.TopConsumers)
	}
}

func TestCreditRepoCreateValidation(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)
	defer s.Close()

	err := s.Credits().Create(&CreditTransaction{})
	if err == nil {
		t.Fatalf("expected validation error for empty transaction")
	}
}
