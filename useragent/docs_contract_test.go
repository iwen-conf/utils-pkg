package useragent

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestUsageDocIsHeuristicNotSecurityVerdict(t *testing.T) {
	data, err := os.ReadFile("使用说明.md")
	if err != nil {
		t.Fatalf("read 使用说明.md: %v", err)
	}
	text := string(data)

	required := []string{
		"启发式",
		"不是安全判定",
		"不得用于",
		"time.Duration",
		"NewLRUCache(1000, 30*time.Minute)",
		"NewShardedCache(500, time.Hour)",
		"if err != nil",
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Errorf("使用说明.md missing %q", want)
		}
	}

	forbidden := []string{
		"- 爬虫（机器人）检测",
		"检测到爬虫:",
		"### 检测爬虫",
		"cache := useragent.NewLRUCache(1000, 1800)",
		"NewShardedCache(500, 3600)",
		"NewShardedCache(1000, 3600)",
		`"is_bot":`,
		"进行浏览器/爬虫区分",
		"ttl         int64",
	}
	for _, bad := range forbidden {
		if strings.Contains(text, bad) {
			t.Errorf("使用说明.md still contains %q", bad)
		}
	}
}

func TestDocsCacheConstructorExample(t *testing.T) {
	cache, err := NewLRUCache(1000, 30*time.Minute)
	if err != nil {
		t.Fatalf("NewLRUCache(1000, 30*time.Minute): %v", err)
	}
	if err := cache.Put("key1", "value1"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	value, found := cache.Get("key1")
	if !found || value != "value1" {
		t.Fatalf("Get after Put: found=%v value=%v", found, value)
	}

	sharded, err := NewShardedCache(500, time.Hour)
	if err != nil {
		t.Fatalf("NewShardedCache(500, time.Hour): %v", err)
	}
	if err := sharded.Put("ua-1", BrowserInfo{Name: "Chrome"}); err != nil {
		t.Fatalf("sharded Put: %v", err)
	}

	if _, err := NewLRUCache(0, time.Minute); !errors.Is(err, ErrInvalidCacheCapacity) {
		t.Fatalf("capacity 0: %v", err)
	}
	if _, err := NewShardedCache(500, -time.Second); !errors.Is(err, ErrInvalidCacheTTL) {
		t.Fatalf("negative ttl: %v", err)
	}
}
