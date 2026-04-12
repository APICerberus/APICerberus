package store

import "time"

const auditSelectColumns = `
	SELECT id, request_id, route_id, route_name, service_name,
	       user_id, consumer_name, method, host, path, query,
	       status_code, latency_ms, bytes_in, bytes_out,
	       client_ip, user_agent, blocked, block_reason,
	       request_headers, request_body, response_headers, response_body,
	       error_message, created_at
	  FROM audit_logs`

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID              string         `json:"id"`
	RequestID       string         `json:"request_id"`
	RouteID         string         `json:"route_id"`
	RouteName       string         `json:"route_name"`
	ServiceName     string         `json:"service_name"`
	UserID          string         `json:"user_id"`
	ConsumerName    string         `json:"consumer_name"`
	Method          string         `json:"method"`
	Host            string         `json:"host"`
	Path            string         `json:"path"`
	Query           string         `json:"query"`
	StatusCode      int            `json:"status_code"`
	LatencyMS       int64          `json:"latency_ms"`
	BytesIn         int64          `json:"bytes_in"`
	BytesOut        int64          `json:"bytes_out"`
	ClientIP        string         `json:"client_ip"`
	UserAgent       string         `json:"user_agent"`
	Blocked         bool           `json:"blocked"`
	BlockReason     string         `json:"block_reason"`
	RequestHeaders  map[string]any `json:"request_headers"`
	RequestBody     string         `json:"request_body"`
	ResponseHeaders map[string]any `json:"response_headers"`
	ResponseBody    string         `json:"response_body"`
	ErrorMessage    string         `json:"error_message"`
	CreatedAt       time.Time      `json:"created_at"`
}

// AuditListOptions provides basic pagination and filtering for audit listing.
type AuditListOptions struct {
	UserID    string
	RouteID   string
	Method    string
	StatusMin int
	StatusMax int
	Limit     int
	Offset    int
}

// AuditSearchFilters provides advanced filtering for audit search.
type AuditSearchFilters struct {
	UserID       string
	APIKeyPrefix string
	Route        string
	Method       string
	StatusMin    int
	StatusMax    int
	ClientIP     string
	Blocked      *bool
	BlockReason  string
	DateFrom     *time.Time
	DateTo       *time.Time
	MinLatencyMS int64
	FullText     string
	Limit        int
	Offset       int
}

// AuditListResult is the paginated result of an audit list/search query.
type AuditListResult struct {
	Entries []AuditEntry `json:"entries"`
	Total   int          `json:"total"`
}

// AuditRouteStat represents a single row in the top-routes aggregation.
type AuditRouteStat struct {
	RouteID   string `json:"route_id"`
	RouteName string `json:"route_name"`
	Count     int64  `json:"count"`
}

// AuditUserStat represents a single row in the top-users aggregation.
type AuditUserStat struct {
	UserID       string `json:"user_id"`
	ConsumerName string `json:"consumer_name"`
	Count        int64  `json:"count"`
}

// AuditStats holds aggregated audit statistics.
type AuditStats struct {
	TotalRequests int64            `json:"total_requests"`
	ErrorRequests int64            `json:"error_requests"`
	ErrorRate     float64          `json:"error_rate"`
	AvgLatencyMS  float64          `json:"avg_latency_ms"`
	TopRoutes     []AuditRouteStat `json:"top_routes"`
	TopUsers      []AuditUserStat  `json:"top_users"`
}
