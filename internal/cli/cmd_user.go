package cli

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type adminCommandFlags struct {
	configPath *string
	adminURL   *string
	adminKey   *string
	output     *string
}

func addAdminCommandFlags(fs *flag.FlagSet) *adminCommandFlags {
	return &adminCommandFlags{
		configPath: fs.String("config", "apicerberus.yaml", "path to config file for admin connection defaults"),
		adminURL:   fs.String("admin-url", "", "admin API base URL (e.g. http://127.0.0.1:9876)"),
		adminKey:   fs.String("admin-key", "", "admin API key (defaults from config)"),
		output:     fs.String("output", "table", "output format: table or json"),
	}
}

func resolveAdminCommand(flags *adminCommandFlags) (*adminClient, string, error) {
	mode, err := normalizeOutputMode(*flags.output)
	if err != nil {
		return nil, "", err
	}
	client, err := newAdminClient(*flags.configPath, *flags.adminURL, *flags.adminKey)
	if err != nil {
		return nil, "", err
	}
	return client, mode, nil
}

func runUser(args []string) error {
	if len(args) == 0 {
		return errors.New("missing user subcommand (expected: list|create|get|update|suspend|activate|apikey|permission|ip)")
	}
	switch args[0] {
	case "list":
		return runUserList(args[1:])
	case "create":
		return runUserCreate(args[1:])
	case "get":
		return runUserGet(args[1:])
	case "update":
		return runUserUpdate(args[1:])
	case "suspend":
		return runUserStatus(args[1:], "suspend")
	case "activate":
		return runUserStatus(args[1:], "activate")
	case "apikey":
		return runUserAPIKey(args[1:])
	case "permission":
		return runUserPermission(args[1:])
	case "ip":
		return runUserIP(args[1:])
	default:
		return fmt.Errorf("unknown user subcommand %q", args[0])
	}
}

func runUserList(args []string) error {
	fs := flag.NewFlagSet("user list", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	search := fs.String("search", "", "search by email/name/company")
	status := fs.String("status", "", "status filter")
	role := fs.String("role", "", "role filter")
	sortBy := fs.String("sort", "", "sort field")
	desc := fs.Bool("desc", false, "sort descending")
	limit := fs.Int("limit", 50, "result limit")
	offset := fs.Int("offset", 0, "result offset")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	query := url.Values{}
	if strings.TrimSpace(*search) != "" {
		query.Set("search", strings.TrimSpace(*search))
	}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*role) != "" {
		query.Set("role", strings.TrimSpace(*role))
	}
	if strings.TrimSpace(*sortBy) != "" {
		query.Set("sort", strings.TrimSpace(*sortBy))
	}
	if *desc {
		query.Set("desc", "true")
	}
	if *limit > 0 {
		query.Set("limit", asString(*limit))
	}
	if *offset > 0 {
		query.Set("offset", asString(*offset))
	}

	result, err := client.call(http.MethodGet, "/admin/api/v1/users", query, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}

	payload := asMap(result)
	usersRaw, _ := findFirst(payload, "users")
	items := asSlice(usersRaw)
	if len(items) == 0 {
		fmt.Println("No users found.")
		return nil
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		user := asMap(item)
		id, _ := findString(user, "id", "ID")
		email, _ := findString(user, "email", "Email")
		name, _ := findString(user, "name", "Name")
		roleValue, _ := findString(user, "role", "Role")
		statusValue, _ := findString(user, "status", "Status")
		credits, _ := findString(user, "credit_balance", "CreditBalance")
		rows = append(rows, []string{id, email, name, roleValue, statusValue, credits})
	}
	printTable([]string{"ID", "EMAIL", "NAME", "ROLE", "STATUS", "CREDITS"}, rows)
	if totalRaw, ok := findFirst(payload, "total", "Total"); ok {
		fmt.Printf("\nTotal: %d\n", asInt(totalRaw, len(items)))
	}
	return nil
}

func runUserCreate(args []string) error {
	fs := flag.NewFlagSet("user create", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	email := fs.String("email", "", "user email")
	name := fs.String("name", "", "user name")
	company := fs.String("company", "", "company")
	role := fs.String("role", "user", "role")
	status := fs.String("status", "active", "status")
	password := fs.String("password", "", "password")
	credits := fs.Int("credits", 0, "initial credits")
	if err := fs.Parse(args); err != nil {
		return err
	}

	emailValue, err := requireArg(*email, "email")
	if err != nil {
		return err
	}
	nameValue, err := requireArg(*name, "name")
	if err != nil {
		return err
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"email":           emailValue,
		"name":            nameValue,
		"company":         strings.TrimSpace(*company),
		"role":            strings.TrimSpace(*role),
		"status":          strings.TrimSpace(*status),
		"initial_credits": *credits,
	}
	if strings.TrimSpace(*password) != "" {
		payload["password"] = strings.TrimSpace(*password)
	}
	result, err := client.call(http.MethodPost, "/admin/api/v1/users", nil, payload)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	user := asMap(result)
	rows := [][]string{{
		firstString(user, "id", "ID"),
		firstString(user, "email", "Email"),
		firstString(user, "name", "Name"),
		firstString(user, "role", "Role"),
		firstString(user, "status", "Status"),
		firstString(user, "credit_balance", "CreditBalance"),
	}}
	printTable([]string{"ID", "EMAIL", "NAME", "ROLE", "STATUS", "CREDITS"}, rows)
	return nil
}

func runUserGet(args []string) error {
	fs := flag.NewFlagSet("user get", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	id := fs.String("id", "", "user id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	userID := strings.TrimSpace(*id)
	if userID == "" && fs.NArg() > 0 {
		userID = strings.TrimSpace(fs.Arg(0))
	}
	if userID == "" {
		return errors.New("user id is required (use --id or positional <id>)")
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodGet, "/admin/api/v1/users/"+url.PathEscape(userID), nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	user := asMap(result)
	printMapAsKeyValues(user)
	return nil
}

func runUserUpdate(args []string) error {
	fs := flag.NewFlagSet("user update", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	id := fs.String("id", "", "user id")
	email := fs.String("email", "", "user email")
	name := fs.String("name", "", "user name")
	company := fs.String("company", "", "company")
	role := fs.String("role", "", "role")
	status := fs.String("status", "", "status")
	password := fs.String("password", "", "password")
	credits := fs.Int("credits", -1, "credit balance override")
	rateLimitRPS := fs.Int("rate-limit-rps", -1, "requests_per_second rate limit override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	userID := strings.TrimSpace(*id)
	if userID == "" && fs.NArg() > 0 {
		userID = strings.TrimSpace(fs.Arg(0))
	}
	if userID == "" {
		return errors.New("user id is required (use --id or positional <id>)")
	}

	payload := map[string]any{}
	if strings.TrimSpace(*email) != "" {
		payload["email"] = strings.TrimSpace(*email)
	}
	if strings.TrimSpace(*name) != "" {
		payload["name"] = strings.TrimSpace(*name)
	}
	if strings.TrimSpace(*company) != "" {
		payload["company"] = strings.TrimSpace(*company)
	}
	if strings.TrimSpace(*role) != "" {
		payload["role"] = strings.TrimSpace(*role)
	}
	if strings.TrimSpace(*status) != "" {
		payload["status"] = strings.TrimSpace(*status)
	}
	if strings.TrimSpace(*password) != "" {
		payload["password"] = strings.TrimSpace(*password)
	}
	if *credits >= 0 {
		payload["credit_balance"] = *credits
	}
	if *rateLimitRPS >= 0 {
		payload["rate_limits"] = map[string]any{
			"requests_per_second": *rateLimitRPS,
		}
	}
	if len(payload) == 0 {
		return errors.New("no update fields provided")
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodPut, "/admin/api/v1/users/"+url.PathEscape(userID), nil, payload)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	user := asMap(result)
	rows := [][]string{{
		firstString(user, "id", "ID"),
		firstString(user, "email", "Email"),
		firstString(user, "name", "Name"),
		firstString(user, "role", "Role"),
		firstString(user, "status", "Status"),
		firstString(user, "credit_balance", "CreditBalance"),
	}}
	printTable([]string{"ID", "EMAIL", "NAME", "ROLE", "STATUS", "CREDITS"}, rows)
	return nil
}

func runUserStatus(args []string, statusAction string) error {
	fs := flag.NewFlagSet("user "+statusAction, flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	id := fs.String("id", "", "user id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	userID := strings.TrimSpace(*id)
	if userID == "" && fs.NArg() > 0 {
		userID = strings.TrimSpace(fs.Arg(0))
	}
	if userID == "" {
		return errors.New("user id is required (use --id or positional <id>)")
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodPost, "/admin/api/v1/users/"+url.PathEscape(userID)+"/"+statusAction, nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	fmt.Printf("User %s %s.\n", userID, statusAction+"d")
	return nil
}

func runUserAPIKey(args []string) error {
	if len(args) == 0 {
		return errors.New("missing user apikey subcommand (expected: list|create|revoke)")
	}
	switch args[0] {
	case "list":
		return runUserAPIKeyList(args[1:])
	case "create":
		return runUserAPIKeyCreate(args[1:])
	case "revoke":
		return runUserAPIKeyRevoke(args[1:])
	default:
		return fmt.Errorf("unknown user apikey subcommand %q", args[0])
	}
}

func runUserAPIKeyList(args []string) error {
	fs := flag.NewFlagSet("user apikey list", flag.ContinueOnError)
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
	result, err := client.call(http.MethodGet, "/admin/api/v1/users/"+url.PathEscape(id)+"/api-keys", nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	items := asSlice(result)
	if len(items) == 0 {
		fmt.Println("No API keys found.")
		return nil
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		key := asMap(item)
		rows = append(rows, []string{
			firstString(key, "id", "ID"),
			firstString(key, "key_prefix", "KeyPrefix"),
			firstString(key, "name", "Name"),
			firstString(key, "status", "Status"),
			firstString(key, "last_used_at", "LastUsedAt"),
			firstString(key, "created_at", "CreatedAt"),
		})
	}
	printTable([]string{"ID", "PREFIX", "NAME", "STATUS", "LAST USED", "CREATED"}, rows)
	return nil
}

func runUserAPIKeyCreate(args []string) error {
	fs := flag.NewFlagSet("user apikey create", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	name := fs.String("name", "default", "key name")
	modeName := fs.String("mode", "live", "key mode: live|test")
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
	result, err := client.call(http.MethodPost, "/admin/api/v1/users/"+url.PathEscape(id)+"/api-keys", nil, map[string]any{
		"name": strings.TrimSpace(*name),
		"mode": strings.TrimSpace(*modeName),
	})
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	resp := asMap(result)
	if fullKey, ok := findString(resp, "key", "full_key"); ok && fullKey != "" {
		fmt.Printf("API key created: %s\n", fullKey)
	}
	if keyRaw, ok := findFirst(resp, "api_key", "key_info", "key"); ok {
		if keyMap := asMap(keyRaw); keyMap != nil {
			rows := [][]string{{
				firstString(keyMap, "id", "ID"),
				firstString(keyMap, "key_prefix", "KeyPrefix"),
				firstString(keyMap, "name", "Name"),
				firstString(keyMap, "status", "Status"),
			}}
			printTable([]string{"ID", "PREFIX", "NAME", "STATUS"}, rows)
			return nil
		}
	}
	printMapAsKeyValues(resp)
	return nil
}

func runUserAPIKeyRevoke(args []string) error {
	fs := flag.NewFlagSet("user apikey revoke", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	keyID := fs.String("key", "", "api key id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}
	kid, err := requireArg(*keyID, "key")
	if err != nil {
		return err
	}
	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	_, err = client.call(http.MethodDelete, "/admin/api/v1/users/"+url.PathEscape(id)+"/api-keys/"+url.PathEscape(kid), nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(map[string]any{"revoked": true, "user_id": id, "key_id": kid})
	}
	fmt.Printf("API key revoked: user=%s key=%s\n", id, kid)
	return nil
}

func runUserPermission(args []string) error {
	if len(args) == 0 {
		return errors.New("missing user permission subcommand (expected: list|grant|revoke)")
	}
	switch args[0] {
	case "list":
		return runUserPermissionList(args[1:])
	case "grant":
		return runUserPermissionGrant(args[1:])
	case "revoke":
		return runUserPermissionRevoke(args[1:])
	default:
		return fmt.Errorf("unknown user permission subcommand %q", args[0])
	}
}

func runUserPermissionList(args []string) error {
	fs := flag.NewFlagSet("user permission list", flag.ContinueOnError)
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
	result, err := client.call(http.MethodGet, "/admin/api/v1/users/"+url.PathEscape(id)+"/permissions", nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	items := asSlice(result)
	if len(items) == 0 {
		fmt.Println("No permissions found.")
		return nil
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		permission := asMap(item)
		rows = append(rows, []string{
			firstString(permission, "id", "ID"),
			firstString(permission, "route_id", "RouteID"),
			firstString(permission, "methods", "Methods"),
			firstString(permission, "allowed", "Allowed"),
			firstString(permission, "credit_cost", "CreditCost"),
		})
	}
	printTable([]string{"ID", "ROUTE", "METHODS", "ALLOWED", "CREDIT COST"}, rows)
	return nil
}

func runUserPermissionGrant(args []string) error {
	fs := flag.NewFlagSet("user permission grant", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	routeID := fs.String("route", "", "route id")
	methods := fs.String("methods", "", "comma-separated methods")
	allowed := fs.Bool("allow", true, "allow or deny")
	creditCost := fs.Int("credit-cost", -1, "optional credit cost override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}
	route, err := requireArg(*routeID, "route")
	if err != nil {
		return err
	}

	payload := map[string]any{
		"route_id": route,
		"allowed":  *allowed,
	}
	if strings.TrimSpace(*methods) != "" {
		payload["methods"] = splitCSV(*methods)
	}
	if *creditCost >= 0 {
		payload["credit_cost"] = *creditCost
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodPost, "/admin/api/v1/users/"+url.PathEscape(id)+"/permissions", nil, payload)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	printMapAsKeyValues(asMap(result))
	return nil
}

func runUserPermissionRevoke(args []string) error {
	fs := flag.NewFlagSet("user permission revoke", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	permissionID := fs.String("permission", "", "permission id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}
	pid, err := requireArg(*permissionID, "permission")
	if err != nil {
		return err
	}
	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	_, err = client.call(http.MethodDelete, "/admin/api/v1/users/"+url.PathEscape(id)+"/permissions/"+url.PathEscape(pid), nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(map[string]any{"revoked": true, "user_id": id, "permission_id": pid})
	}
	fmt.Printf("Permission revoked: user=%s permission=%s\n", id, pid)
	return nil
}

func runUserIP(args []string) error {
	if len(args) == 0 {
		return errors.New("missing user ip subcommand (expected: list|add|remove)")
	}
	switch args[0] {
	case "list":
		return runUserIPList(args[1:])
	case "add":
		return runUserIPAdd(args[1:])
	case "remove":
		return runUserIPRemove(args[1:])
	default:
		return fmt.Errorf("unknown user ip subcommand %q", args[0])
	}
}

func runUserIPList(args []string) error {
	fs := flag.NewFlagSet("user ip list", flag.ContinueOnError)
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
	result, err := client.call(http.MethodGet, "/admin/api/v1/users/"+url.PathEscape(id)+"/ip-whitelist", nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	payload := asMap(result)
	rawIPs, _ := findFirst(payload, "ip_whitelist", "IPWhitelist")
	ips := asSlice(rawIPs)
	if len(ips) == 0 {
		fmt.Println("No IP whitelist entries.")
		return nil
	}
	rows := make([][]string, 0, len(ips))
	for _, ip := range ips {
		rows = append(rows, []string{asString(ip)})
	}
	printTable([]string{"IP"}, rows)
	return nil
}

func runUserIPAdd(args []string) error {
	fs := flag.NewFlagSet("user ip add", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	ip := fs.String("ip", "", "ip or cidr")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}
	ipValue, err := requireArg(*ip, "ip")
	if err != nil {
		return err
	}
	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodPost, "/admin/api/v1/users/"+url.PathEscape(id)+"/ip-whitelist", nil, map[string]any{"ip": ipValue})
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	fmt.Printf("IP whitelist added: user=%s ip=%s\n", id, ipValue)
	return nil
}

func runUserIPRemove(args []string) error {
	fs := flag.NewFlagSet("user ip remove", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	userID := fs.String("user", "", "user id")
	ip := fs.String("ip", "", "ip or cidr")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := requireArg(*userID, "user")
	if err != nil {
		return err
	}
	ipValue, err := requireArg(*ip, "ip")
	if err != nil {
		return err
	}
	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	_, err = client.call(http.MethodDelete, "/admin/api/v1/users/"+url.PathEscape(id)+"/ip-whitelist/"+url.PathEscape(ipValue), nil, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(map[string]any{"removed": true, "user_id": id, "ip": ipValue})
	}
	fmt.Printf("IP whitelist removed: user=%s ip=%s\n", id, ipValue)
	return nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func firstString(m map[string]any, keys ...string) string {
	value, _ := findString(m, keys...)
	return value
}
