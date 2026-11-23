package useragent

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestIsBrowser(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{
			name:      "Chrome Browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			want:      true,
		},
		{
			name:      "Firefox Browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
			want:      true,
		},
		{
			name:      "Safari Browser",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			want:      true,
		},
		{
			name:      "Edge Browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
			want:      true,
		},
		{
			name:      "Bot User Agent",
			userAgent: "Googlebot/2.1 (+http://www.google.com/bot.html)",
			want:      false,
		},
		{
			name:      "Empty User Agent",
			userAgent: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBrowser(tt.userAgent); got != tt.want {
				t.Errorf("IsBrowser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBrowserInfo(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      BrowserInfo
	}{
		{
			name:      "Chrome Browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			want:      BrowserInfo{IsBrowser: true, Name: "Chrome", Version: "91.0.4472.124"},
		},
		{
			name:      "Firefox Browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
			want:      BrowserInfo{IsBrowser: true, Name: "Firefox", Version: "89.0"},
		},
		{
			name:      "Safari Browser",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			want:      BrowserInfo{IsBrowser: true, Name: "Safari", Version: "14.1.1"},
		},
		{
			name:      "Edge Browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
			want:      BrowserInfo{IsBrowser: true, Name: "Edge", Version: "91.0.864.59"},
		},
		{
			name:      "Bot User Agent",
			userAgent: "Googlebot/2.1 (+http://www.google.com/bot.html)",
			want:      BrowserInfo{IsBrowser: false},
		},
		{
			name:      "Empty User Agent",
			userAgent: "",
			want:      BrowserInfo{IsBrowser: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBrowserInfo(tt.userAgent)
			if got.IsBrowser != tt.want.IsBrowser || got.Name != tt.want.Name || got.Version != tt.want.Version {
				t.Errorf("GetBrowserInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBrowserMatchesGetBrowserInfo(t *testing.T) {
	uas := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 OPR/77.0.4054.172",
		"Mozilla/5.0 (Windows NT 10.0; Trident/7.0; rv:11.0) like Gecko",
		"curl/8.0.1",
		"Googlebot/2.1 (+http://www.google.com/bot.html)",
	}
	for _, ua := range uas {
		if IsBrowser(ua) != GetBrowserInfo(ua).IsBrowser {
			t.Fatalf("IsBrowser mismatch for %q", ua)
		}
	}
}

func TestLRUCache_CapacityAndTTL(t *testing.T) {
	if _, err := newBrowserLRUCache(0, time.Second); !errors.Is(err, errInvalidCacheCapacity) {
		t.Fatalf("capacity 0: %v", err)
	}
	if _, err := newBrowserLRUCache(1, -time.Second); !errors.Is(err, errInvalidCacheTTL) {
		t.Fatalf("negative ttl: %v", err)
	}
	if _, err := newShardedBrowserCache(0, time.Second); !errors.Is(err, errInvalidCacheCapacity) {
		t.Fatalf("sharded capacity 0: %v", err)
	}
	cache, err := newBrowserLRUCache(1, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.put("a", BrowserInfo{IsBrowser: true, Name: "Chrome"}); err != nil {
		t.Fatal(err)
	}
	if err := cache.put("b", BrowserInfo{IsBrowser: true, Name: "Firefox"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.get("a"); ok {
		t.Fatalf("capacity 1 should evict a")
	}
	if got, ok := cache.get("b"); !ok || got.Name != "Firefox" {
		t.Fatalf("expected Firefox hit, got (%+v, %v)", got, ok)
	}
	time.Sleep(30 * time.Millisecond)
	if _, ok := cache.get("b"); ok {
		t.Fatalf("ttl expired")
	}
	var nilCache *browserLRUCache
	if _, ok := nilCache.get("x"); ok {
		t.Fatalf("nil get")
	}
	if err := nilCache.put("x", BrowserInfo{}); !errors.Is(err, errUninitializedCache) {
		t.Fatalf("nil put: %v", err)
	}

	var wg sync.WaitGroup
	live, err := newBrowserLRUCache(8, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = live.put("k", BrowserInfo{IsBrowser: true, Name: "Chrome"})
			_, _ = live.get("k")
		}(i)
	}
	wg.Wait()
}

func BenchmarkIsBrowser(b *testing.B) {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Googlebot/2.1 (+http://www.google.com/bot.html)",
		"",
	}

	for _, ua := range userAgents {
		b.Run(ua, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				IsBrowser(ua)
			}
		})
	}
}

func BenchmarkGetBrowserInfo(b *testing.B) {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
		"Googlebot/2.1 (+http://www.google.com/bot.html)",
		"",
	}

	for _, ua := range userAgents {
		b.Run(ua, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				GetBrowserInfo(ua)
			}
		})
	}
}
