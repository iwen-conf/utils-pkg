package useragent

import (
	"testing"
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
			want:      BrowserInfo{IsBrowser: true, Name: "Safari", Version: "605.1.15"},
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