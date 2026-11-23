package useragent

import (
	"errors"
	"strings"
	"testing"
)

func TestRepro_IEVersionExtractionAndNilShardedCache(t *testing.T) {
	ie10 := GetBrowserInfo("Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.1; Trident/6.0)")
	if !ie10.IsBrowser || ie10.Name != "Internet Explorer" || ie10.Version != "10.0" {
		t.Fatalf("BUG: MSIE 10.0 parsed as %+v", ie10)
	}

	ie11 := GetBrowserInfo("Mozilla/5.0 (Windows NT 10.0; Trident/7.0; rv:11.0) like Gecko")
	if !ie11.IsBrowser || ie11.Name != "Internet Explorer" || ie11.Version != "11.0" {
		t.Fatalf("BUG: IE 11 parsed as %+v", ie11)
	}

	var sc *shardedBrowserCache
	if _, ok := sc.get("k"); ok {
		t.Fatalf("nil shardedBrowserCache.get should return false")
	}
	if err := sc.put("k", BrowserInfo{}); !errors.Is(err, errUninitializedCache) {
		t.Fatalf("nil shardedBrowserCache.put should return errUninitializedCache, got %v", err)
	}

	hugeUA := "Mozilla/5.0 Chrome/120.0.0.0 " + strings.Repeat("X", 64*1024)
	_ = GetBrowserInfo(hugeUA)
	if _, cached := getBrowserInfoCache().get(hugeUA); cached {
		t.Fatalf("SECURITY BUG: 64KB User-Agent was stored raw in global cache")
	}
}
