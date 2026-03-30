package cli

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func runAnalytics(args []string) error {
	if len(args) == 0 {
		return errors.New("missing analytics subcommand (expected: overview|requests|latency)")
	}
	switch args[0] {
	case "overview":
		return runAnalyticsOverview(args[1:])
	case "requests":
		return runAnalyticsRequests(args[1:])
	case "latency":
		return runAnalyticsLatency(args[1:])
	default:
		return fmt.Errorf("unknown analytics subcommand %q", args[0])
	}
}

func runAnalyticsOverview(args []string) error {
	query, common, err := parseAnalyticsCommonFlags("analytics overview", args)
	if err != nil {
		return err
	}
	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodGet, "/admin/api/v1/analytics/overview", query, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	payload := asMap(result)
	printMapAsKeyValues(payload)
	return nil
}

func runAnalyticsRequests(args []string) error {
	fs := flag.NewFlagSet("analytics requests", flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	from := fs.String("from", "", "RFC3339 start time")
	to := fs.String("to", "", "RFC3339 end time")
	window := fs.String("window", "", "time window duration (e.g. 1h)")
	granularity := fs.String("granularity", "1m", "bucket duration")
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*from) != "" {
		query.Set("from", strings.TrimSpace(*from))
	}
	if strings.TrimSpace(*to) != "" {
		query.Set("to", strings.TrimSpace(*to))
	}
	if strings.TrimSpace(*window) != "" {
		query.Set("window", strings.TrimSpace(*window))
	}
	if strings.TrimSpace(*granularity) != "" {
		query.Set("granularity", strings.TrimSpace(*granularity))
	}

	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodGet, "/admin/api/v1/analytics/timeseries", query, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	payload := asMap(result)
	itemsRaw, _ := findFirst(payload, "items", "Items")
	items := asSlice(itemsRaw)
	if len(items) == 0 {
		fmt.Println("No analytics request data.")
		return nil
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		point := asMap(item)
		rows = append(rows, []string{
			firstString(point, "timestamp"),
			firstString(point, "requests"),
			firstString(point, "errors"),
			firstString(point, "avg_latency_ms"),
			firstString(point, "p95_latency_ms"),
			firstString(point, "credits_consumed"),
		})
	}
	printTable([]string{"TIMESTAMP", "REQUESTS", "ERRORS", "AVG LAT(ms)", "P95 LAT(ms)", "CREDITS"}, rows)
	return nil
}

func runAnalyticsLatency(args []string) error {
	query, common, err := parseAnalyticsCommonFlags("analytics latency", args)
	if err != nil {
		return err
	}
	client, mode, err := resolveAdminCommand(common)
	if err != nil {
		return err
	}
	result, err := client.call(http.MethodGet, "/admin/api/v1/analytics/latency", query, nil)
	if err != nil {
		return err
	}
	if mode == outputJSON {
		return printJSON(result)
	}
	printMapAsKeyValues(asMap(result))
	return nil
}

func parseAnalyticsCommonFlags(name string, args []string) (url.Values, *adminCommandFlags, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	common := addAdminCommandFlags(fs)
	from := fs.String("from", "", "RFC3339 start time")
	to := fs.String("to", "", "RFC3339 end time")
	window := fs.String("window", "", "time window duration (e.g. 1h)")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	query := url.Values{}
	if strings.TrimSpace(*from) != "" {
		query.Set("from", strings.TrimSpace(*from))
	}
	if strings.TrimSpace(*to) != "" {
		query.Set("to", strings.TrimSpace(*to))
	}
	if strings.TrimSpace(*window) != "" {
		query.Set("window", strings.TrimSpace(*window))
	}
	return query, common, nil
}
