package cli

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func runCredit(args []string) error {
	if len(args) == 0 {
		return errors.New("missing credit subcommand (expected: overview|balance|topup|deduct|transactions)")
	}
	switch args[0] {
	case "overview":
		return runCreditOverview(args[1:])
	case "balance":
		return runCreditBalance(args[1:])
	case "topup":
		return runCreditAdjust(args[1:], true)
	case "deduct":
		return runCreditAdjust(args[1:], false)
	case "transactions":
		return runCreditTransactions(args[1:])
	default:
		return fmt.Errorf("unknown credit subcommand %q", args[0])
	}
}

func runCreditOverview(args []string) error {
	fs := flag.NewFlagSet("credit overview", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodGet, "/admin/api/v1/credits/overview", nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}

	payload := asMap(result)
	if payload == nil {
		return printJSON(result)
	}
	totalDistributed := firstString(payload, "total_distributed", "TotalDistributed")
	totalConsumed := firstString(payload, "total_consumed", "TotalConsumed")
	fmt.Printf("Total Distributed: %s\n", totalDistributed)
	fmt.Printf("Total Consumed   : %s\n", totalConsumed)

	topRaw, _ := findFirst(payload, "top_consumers", "TopConsumers")
	items := asSlice(topRaw)
	if len(items) == 0 {
		fmt.Println("\nNo top consumer data.")
		return nil
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		consumer := asMap(item)
		rows = append(rows, []string{
			firstString(consumer, "user_id", "UserID"),
			firstString(consumer, "email", "Email"),
			firstString(consumer, "name", "Name"),
			firstString(consumer, "consumed", "Consumed"),
		})
	}
	fmt.Println()
	printTable([]string{"USER ID", "EMAIL", "NAME", "CONSUMED"}, rows)
	return nil
}

func runCreditBalance(args []string) error {
	fs := flag.NewFlagSet("credit balance", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodGet, "/admin/api/v1/users/"+url.PathEscape(id)+"/credits/balance", nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	payload := asMap(result)
	rows := [][]string{{
		firstString(payload, "user_id", "UserID"),
		firstString(payload, "credit_balance", "CreditBalance", "balance"),
	}}
	printTable([]string{"USER ID", "BALANCE"}, rows)
	return nil
}

func runCreditAdjust(args []string, topup bool) error {
	action := "topup"
	pathAction := "topup"
	if !topup {
		action = "deduct"
		pathAction = "deduct"
	}

	fs := flag.NewFlagSet("credit "+action, flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	amount := fs.Int("amount", 0, "amount")
	reason := fs.String("reason", "", "reason")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}
	if _, err := requireInt(*amount, "amount"); err != nil {
		return err
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"amount": *amount,
		"reason": strings.TrimSpace(*reason),
	}
	result, err := client.call(http.MethodPost, "/admin/api/v1/users/"+url.PathEscape(id)+"/credits/"+pathAction, nil, payload)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	printMapAsKeyValues(asMap(result))
	return nil
}

func runCreditTransactions(args []string) error {
	fs := flag.NewFlagSet("credit transactions", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	txnType := fs.String("type", "", "transaction type filter")
	limit := fs.Int("limit", 50, "result limit")
	offset := fs.Int("offset", 0, "result offset")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*txnType) != "" {
		query.Set("type", strings.TrimSpace(*txnType))
	}
	if *limit > 0 {
		query.Set("limit", asString(*limit))
	}
	if *offset > 0 {
		query.Set("offset", asString(*offset))
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodGet, "/admin/api/v1/users/"+url.PathEscape(id)+"/credits/transactions", query, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	payload := asMap(result)
	itemsRaw, _ := findFirst(payload, "transactions", "Transactions")
	items := asSlice(itemsRaw)
	if len(items) == 0 {
		fmt.Println("No credit transactions found.")
		return nil
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		txn := asMap(item)
		rows = append(rows, []string{
			firstString(txn, "id", "ID"),
			firstString(txn, "type", "Type"),
			firstString(txn, "amount", "Amount"),
			firstString(txn, "balance_after", "BalanceAfter"),
			firstString(txn, "description", "Description"),
			firstString(txn, "created_at", "CreatedAt"),
		})
	}
	printTable([]string{"ID", "TYPE", "AMOUNT", "BALANCE", "DESCRIPTION", "CREATED"}, rows)
	if totalRaw, ok := findFirst(payload, "total", "Total"); ok {
		fmt.Printf("\nTotal: %d\n", asInt(totalRaw, len(items)))
	}
	return nil
}
