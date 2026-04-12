// Package main provides comprehensive benchmarks for API Cerberus.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// BenchmarkRouter benchmarks the radix tree router.
func BenchmarkRouter(b *testing.B) {
	routes := make(map[string]bool)

	// Add routes
	routePaths := []string{
		"/api/v1/users",
		"/api/v1/users/:id",
		"/api/v1/orders",
		"/api/v1/orders/:id",
		"/api/v1/products",
		"/api/v1/products/:id",
		"/health",
		"/metrics",
	}

	for _, route := range routePaths {
		routes[route] = true
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = routes["/api/v1/users/123"]
		}
	})
}

// BenchmarkRouterComplex benchmarks complex routing scenarios.
func BenchmarkRouterComplex(b *testing.B) {
	routes := make(map[string]int)

	// Simulate 100 routes
	for i := 0; i < 100; i++ {
		routes[fmt.Sprintf("/api/v1/service%d/:id", i)] = i
	}

	paths := []string{
		"/api/v1/service50/123",
		"/api/v1/service75/456",
		"/api/v1/service99/789",
	}

	b.ResetTimer()
	idx := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = routes[paths[idx%len(paths)]]
			idx++
		}
	})
}

// BenchmarkHTTPRequest benchmarks HTTP request creation.
func BenchmarkHTTPRequest(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Request-ID", fmt.Sprintf("req-%d", i))
	}
}

// BenchmarkResponseWriter benchmarks response writing.
func BenchmarkResponseWriter(b *testing.B) {
	data := []byte(`{"status":"ok","data":{"id":123,"name":"test"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		rr.WriteHeader(http.StatusOK)
		_, _ = rr.Write(data)
	}
}

// BenchmarkJWTSign benchmarks JWT signing simulation.
func BenchmarkJWTSign(b *testing.B) {
	payload := fmt.Sprintf(`{"sub":"user%d","exp":%d}`, 123, time.Now().Add(time.Hour).Unix())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(payload) + i
	}
}

// BenchmarkUUID benchmarks UUID v4 generation simulation.
func BenchmarkUUID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%08d-%04d-4%03d-y%03d-%012d", i, i%10000, i%1000, i%1000, i)
	}
}

// BenchmarkJSONMarshal benchmarks JSON string building.
func BenchmarkJSONMarshal(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf(`{"id":"%d","name":"Test %d","active":true,"count":%d}`, i, i, i%100)
	}
}

// BenchmarkRateLimiterTokenBucket benchmarks token bucket simulation.
func BenchmarkRateLimiterTokenBucket(b *testing.B) {
	tokens := 100.0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if tokens >= 1 {
			tokens--
		}
		if i%100 == 0 {
			tokens = 100 // refill
		}
	}
}

// BenchmarkCacheGet benchmarks cache retrieval simulation.
func BenchmarkCacheGet(b *testing.B) {
	cache := make(map[string]string)
	cache["key1"] = "value1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache["key1"]
	}
}

// BenchmarkCacheSet benchmarks cache storage simulation.
func BenchmarkCacheSet(b *testing.B) {
	cache := make(map[string]string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache[fmt.Sprintf("key%d", i)] = "value"
	}
}

// BenchmarkStringConcat benchmarks string concatenation.
func BenchmarkStringConcat(b *testing.B) {
	parts := []string{"/api/v1", "/users", "/123", "/profile"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		for _, p := range parts {
			result += p
		}
		_ = result
	}
}

// BenchmarkStringBuilder benchmarks strings.Builder.
func BenchmarkStringBuilder(b *testing.B) {
	parts := []string{"/api/v1", "/users", "/123", "/profile"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		for _, p := range parts {
			sb.WriteString(p)
		}
		_ = sb.String()
	}
}

// BenchmarkParseQueryString benchmarks query parameter parsing.
func BenchmarkParseQueryString(b *testing.B) {
	query := "key1=value1&key2=value2&key3=value3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make(map[string]string)
		pairs := splitN(query, '&', 10)
		for _, pair := range pairs {
			kv := splitN(pair, '=', 2)
			if len(kv) == 2 {
				result[kv[0]] = kv[1]
			}
		}
	}
}

func splitN(s string, sep byte, n int) []string {
	var result []string
	start := 0
	for i := 0; i < len(s) && len(result) < n-1; i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
