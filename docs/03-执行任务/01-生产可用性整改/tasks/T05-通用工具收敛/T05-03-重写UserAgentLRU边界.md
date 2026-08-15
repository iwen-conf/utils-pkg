# T05-03 修复 F-UTIL-001 重写 User-Agent LRU 边界

状态：`[ ]`

## 来源

1. finding_id：F-UTIL-001（User-Agent 部分）。
2. Handoff：[自研 LRU 对无效容量和并发更新不安全](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-util-001--自研-lru-对无效容量和并发更新不安全)。
3. 权限类型：service。
4. 数据产量：availability capability；无效容量 panic 和并发过期删除可破坏新值。

## 目标

User-Agent 缓存使用 `container/list + map` 的 O(1) LRU，在单个锁内完成读取、过期检查、移动和删除。constructor 对无效容量/TTL 返回 error，调用方不能获得半可用对象。

## 允许改动

1. `useragent/useragent.go` 中 CacheEntry/LRUCache/ShardedCache 和默认缓存初始化。
2. `useragent/useragent_test.go`，或新增 `useragent/cache_test.go`。
3. `useragent/使用说明.md` 中公开缓存 API 示例。

## 禁止改动

1. 不在本任务改变浏览器识别规则；留给 T05-04。
2. 不引入第三方缓存依赖、后台 goroutine 或定时清理器。
3. 不保留 capacity<=0 时“等 Put 再 panic”的行为。
4. 不用切片扫描实现 move/evict。

<Context-Queries>

```bash
arc-idx search 'f:^useragent/ CacheEntry|LRUCache|NewLRUCache|Get\(|Put\(|moveToFront|cleanup|ShardedCache' --repo . --context 5 --max 220 --format json
arc-idx search 'f:^useragent/ isBrowserCache|browserInfoCache|capacity|ttl|expiration|cleanupTime' --repo . --context 4 --max 200 --format json
arc-idx ast --repo . --lang go 'func ($C *LRUCache) Get($KEY string) (interface{}, bool) { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. constructor 返回 `*LRUCache`，不验证 capacity/ttl。
2. capacity 0/负数时 Put 访问 `keys[len(keys)-1]` panic。
3. Get 在读锁下取得旧 entry，释放锁后检查过期，再加写锁删除；期间并发 Put 的新 entry 可能被旧 Get 删除。
4. move/remove 使用 keys slice 线性扫描和头插，Get/Put 为 O(n)。
5. cleanup 只在命中 Get 时每 5 分钟触发，语义隐蔽。

## 固定公开 API

```go
func NewLRUCache(capacity int, ttl time.Duration) (*LRUCache, error)
func NewShardedCache(shardCapacity int, ttl time.Duration) (*ShardedCache, error)
func (c *LRUCache) Get(key string) (any, bool)
func (c *LRUCache) Put(key string, value any) error
func (sc *ShardedCache) Get(key string) (any, bool)
func (sc *ShardedCache) Put(key string, value any) error
```

1. capacity <= 0 返回 `ErrInvalidCacheCapacity` 和 nil。
2. ttl < 0 返回 `ErrInvalidCacheTTL` 和 nil；ttl == 0 表示不过期。
3. nil receiver 的 Get 返回 `(nil, false)`；Put 返回 `ErrUninitializedCache`，不得 no-op 成功。
4. T06-04 必须记录 ttl 从秒数 int64 改为 `time.Duration` 的 v0.4.0 迁移。

## 固定内部结构

```go
type cacheItem struct {
    key       string
    value     any
    expiresAt time.Time
}

type LRUCache struct {
    mu       sync.Mutex
    capacity int
    ttl      time.Duration
    items    map[string]*list.Element
    order    *list.List
    now      func() time.Time
}
```

1. list front 为最近使用，back 为最久未使用。
2. `now` 是私有测试时钟，constructor 默认 `time.Now`；不提供公开时钟配置。
3. Get 从加锁到过期删除/MoveToFront 全程单锁，返回 element 当前值。
4. Put 更新现有 element 或新增 front；超容量从 back 精确删除。
5. 删除定期 cleanupTime；过期项按访问/淘汰处理，不启动 goroutine。

## 执行清单

- [ ] 定义两个哨兵错误和 constructor 校验，失败返回 nil 对象。
- [ ] 用 list+map 替换 CacheEntry map + keys slice；把导出 `CacheEntry` 收敛为私有实现类型。
- [ ] 重写 Get 为单锁事务，消除 stale read 删除新值窗口。
- [ ] 按固定新签名重写 Put 的 update/insert/evict，保证 map/list 一一对应并传播 nil/uninitialized error。
- [ ] 重写 ShardedCache constructor 传播错误；16 个分片全部成功后才返回对象。
- [ ] 默认全局缓存使用内部已验证 constants 初始化；不忽略 constructor error。
- [ ] 测试通过私有 `now` 推进时间，不使用多秒 sleep。
- [ ] 更新使用说明 constructor 签名、time.Duration、错误处理和零 TTL 语义。

## 正向测试

1. Put/Get 命中，更新相同 key 后返回新值并移动到 front。
2. 容量 2 时访问 A 后插入 C，应淘汰 B。
3. ttl=0 的 entry 不过期；正 ttl 在测试时钟越过 expiresAt 后 miss 并从 map/list 同时删除。
4. ShardedCache 对同 key 稳定落同 shard，多 key 并发通过 race。
5. 并发 Get/Put 同 key 后最终最新写入仍可取，不被过期旧读删除。

## 反向测试

1. capacity 0/负数、ttl 负数 constructor 返回 nil 和对应错误。
2. nil receiver/未初始化内部结构不 panic，行为按选定 API 契约测试。
3. 连续更新 key 不产生重复 list element，map/list 长度始终相同且不超过 capacity。
4. 过期删除后再次 Put 同 key 能正常插入。

## 验证命令

```bash
gofmt -w useragent/useragent.go useragent/*_test.go
go test -count=1 ./useragent -run 'LRU|Cache|Capacity|TTL|Expire|Concurrent|Shard'
go test -race -count=1 ./useragent
go test -count=1 ./...
go vet ./...
rg -n 'keys \[\]string|moveToFront|removeKey|cleanupTime' useragent || true
git diff --check
```

## 不要做

1. 不在释放读锁后基于旧 entry 删除当前 map 值。
2. 不用 slice 头插/扫描冒充 O(1) LRU。
3. 不启动清理 goroutine，避免 Shutdown 生命周期扩张。
4. 不忽略 NewShardedCache 的任一 shard 构造错误。
5. 不使用真实长 sleep 制造过期测试。

## 完成标准

1. 无效配置 fail-fast，map/list 操作 O(1) 且单锁一致。
2. 旧 stale-read 并发窗口有回归覆盖，race 通过。
3. TTL、LRU 次序、更新、分片和容量边界测试完整。
4. 文档使用新签名，进度表同步迁移事项。
