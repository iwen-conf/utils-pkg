package useragent

import (
	"os"
	"strings"
	"testing"
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
		"MaxUserAgentLength",
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
		"NewLRUCache",
		"NewShardedCache",
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
