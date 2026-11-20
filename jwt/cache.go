package jwt

import (
	"container/list"
	"sync"
	"time"
)

type cachedClaimsEntry struct {
	key        string
	claims     *StandardClaims
	createdAt  time.Time
	validUntil time.Time
}

// validationCache 是严格受限的 O(1) LRU 正向解析缓存。
//
// 安全不变量：
//  1. 仅缓存已验证通过的令牌（绝不缓存失败结果，避免延长 nbf 未生效或瞬时故障，并防止随机无效 token 挤占缓存）。
//  2. validUntil = min(now + cacheTTL, claims.ExpiresAt)，绝不越过令牌自身 exp。
//  3. 写入和读取均执行深拷贝，隔离外部可变状态。
//  4. 容量严格按 LRU 淘汰，即使 capacity == 1 也不会越界增长。
type validationCache struct {
	mu    sync.Mutex
	items map[string]*list.Element
	order *list.List
}

func newValidationCache() *validationCache {
	return &validationCache{
		items: make(map[string]*list.Element),
		order: list.New(),
	}
}

func (c *validationCache) get(token string, now time.Time) (*StandardClaims, error, bool) {
	c.mu.Lock()
	el, exists := c.items[token]
	if !exists {
		c.mu.Unlock()
		return nil, nil, false
	}
	entry := el.Value.(*cachedClaimsEntry)
	if !now.Before(entry.validUntil) {
		c.removeElementLocked(el)
		c.mu.Unlock()
		return nil, nil, false
	}
	c.order.MoveToFront(el)
	stored := entry.claims
	c.mu.Unlock()

	cloned, err := cloneStandardClaims(stored)
	if err != nil {
		return nil, err, true
	}
	return cloned, nil, true
}

func (c *validationCache) put(token string, claims *StandardClaims, now time.Time, ttl time.Duration, capacity int) error {
	if capacity <= 0 || ttl <= 0 {
		return nil
	}
	if claims == nil || claims.ExpiresAt == nil || !claims.ExpiresAt.Time.After(now) {
		return nil
	}

	cloned, err := cloneStandardClaims(claims)
	if err != nil {
		return err
	}

	validUntil := now.Add(ttl)
	if claims.ExpiresAt.Time.Before(validUntil) {
		validUntil = claims.ExpiresAt.Time
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.items[token]; ok {
		entry := existing.Value.(*cachedClaimsEntry)
		entry.claims = cloned
		entry.createdAt = now
		entry.validUntil = validUntil
		c.order.MoveToFront(existing)
		return nil
	}

	for c.order.Len() >= capacity {
		back := c.order.Back()
		if back == nil {
			break
		}
		c.removeElementLocked(back)
	}

	el := c.order.PushFront(&cachedClaimsEntry{
		key:        token,
		claims:     cloned,
		createdAt:  now,
		validUntil: validUntil,
	})
	c.items[token] = el
	return nil
}

func (c *validationCache) delete(token string) {
	c.mu.Lock()
	if el, ok := c.items[token]; ok {
		c.removeElementLocked(el)
	}
	c.mu.Unlock()
}

func (c *validationCache) clear() {
	c.mu.Lock()
	c.items = make(map[string]*list.Element)
	c.order.Init()
	c.mu.Unlock()
}

func (c *validationCache) resize(capacity int) {
	if capacity < 0 {
		capacity = 0
	}
	c.mu.Lock()
	for c.order.Len() > capacity {
		back := c.order.Back()
		if back == nil {
			break
		}
		c.removeElementLocked(back)
	}
	c.mu.Unlock()
}

func (c *validationCache) clampTTL(now time.Time, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for el := c.order.Front(); el != nil; {
		next := el.Next()
		entry := el.Value.(*cachedClaimsEntry)
		maxValidUntil := entry.createdAt.Add(ttl)
		if maxValidUntil.Before(entry.validUntil) {
			entry.validUntil = maxValidUntil
		}
		if !now.Before(entry.validUntil) {
			c.removeElementLocked(el)
		}
		el = next
	}
}

func (c *validationCache) cleanExpired(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cleaned := 0
	for el := c.order.Front(); el != nil; {
		next := el.Next()
		entry := el.Value.(*cachedClaimsEntry)
		if !now.Before(entry.validUntil) {
			c.removeElementLocked(el)
			cleaned++
		}
		el = next
	}
	return cleaned
}

func (c *validationCache) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

func (c *validationCache) removeElementLocked(el *list.Element) {
	entry := el.Value.(*cachedClaimsEntry)
	delete(c.items, entry.key)
	c.order.Remove(el)
}
