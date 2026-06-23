package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"KorisPanel/panel/internal/cache"
)

func TestCacheStats_GET(t *testing.T) {
	tests := []struct {
		name       string
		cache      *cache.QueryCache
		wantHits   int64
		wantMisses int64
		wantSize   int
	}{
		{
			name:       "nil cache returns zeros",
			cache:      nil,
			wantHits:   0,
			wantMisses: 0,
			wantSize:   0,
		},
		{
			name:       "empty cache returns zeros",
			cache:      cache.NewQueryCache(100, 5*time.Minute),
			wantHits:   0,
			wantMisses: 0,
			wantSize:   0,
		},
		{
			name: "cache with entries returns correct stats",
			cache: func() *cache.QueryCache {
				c := cache.NewQueryCache(100, 5*time.Minute)
				c.Set("key1", "value1")
				c.Set("key2", "value2")
				c.Get("key1") // hit
				c.Get("key3") // miss
				return c
			}(),
			wantHits:   1,
			wantMisses: 1,
			wantSize:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{Cache: tt.cache}
			req := httptest.NewRequest(http.MethodGet, "/api/admin/cache-stats", nil)
			rec := httptest.NewRecorder()

			s.cacheStats(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			var resp map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["ok"] != true {
				t.Errorf("ok = %v, want true", resp["ok"])
			}

			cacheData, ok := resp["cache"].(map[string]any)
			if !ok {
				t.Fatal("missing or invalid 'cache' field in response")
			}

			if got := int64(cacheData["hits"].(float64)); got != tt.wantHits {
				t.Errorf("hits = %d, want %d", got, tt.wantHits)
			}
			if got := int64(cacheData["misses"].(float64)); got != tt.wantMisses {
				t.Errorf("misses = %d, want %d", got, tt.wantMisses)
			}
			if got := int(cacheData["size"].(float64)); got != tt.wantSize {
				t.Errorf("size = %d, want %d", got, tt.wantSize)
			}
		})
	}
}

func TestCacheStats_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/cache-stats", nil)
	rec := httptest.NewRecorder()

	s.cacheStats(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
