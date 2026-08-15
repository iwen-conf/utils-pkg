package jwt

import (
	"context"
	"sync"
	"time"
)

// RefreshTokenUseStore 原子消费 refresh token 的 jti。
// 默认内存实现只保证单进程；多实例必须注入真正原子的外部实现
// （SETNX / 唯一键 / 事务），禁止 Get-then-Set。
type RefreshTokenUseStore interface {
	Consume(ctx context.Context, jti string, expiresAt time.Time) (firstUse bool, err error)
	IsConsumed(ctx context.Context, jti string) (bool, error)
}

// RefreshTokenUseStoreCleaner 是支持主动清理过期消费记录的可选能力。
type RefreshTokenUseStoreCleaner interface {
	CleanExpired(ctx context.Context, now time.Time) (int, error)
}

// MemoryRefreshTokenUseStore 提供单进程默认 refresh 消费存储。
type MemoryRefreshTokenUseStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

var (
	_ RefreshTokenUseStore        = (*MemoryRefreshTokenUseStore)(nil)
	_ RefreshTokenUseStoreCleaner = (*MemoryRefreshTokenUseStore)(nil)
)

// NewMemoryRefreshTokenUseStore 创建内存 refresh 消费存储。
func NewMemoryRefreshTokenUseStore() *MemoryRefreshTokenUseStore {
	return &MemoryRefreshTokenUseStore{entries: make(map[string]time.Time)}
}

// Consume 在同一临界区内检查未过期记录并写入。首次消费返回 firstUse=true。
func (s *MemoryRefreshTokenUseStore) Consume(ctx context.Context, jti string, expiresAt time.Time) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	if jti == "" {
		return false, ErrEmptyRefreshJTI
	}
	now := time.Now()
	if !expiresAt.After(now) {
		return false, ErrRefreshJTINotFuture
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[string]time.Time)
	}
	if existing, ok := s.entries[jti]; ok && existing.After(now) {
		return false, nil
	}
	s.entries[jti] = expiresAt
	return true, nil
}

// IsConsumed 查询未过期的消费记录；过期记录会被删除并视为未消费。
func (s *MemoryRefreshTokenUseStore) IsConsumed(ctx context.Context, jti string) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	if jti == "" {
		return false, ErrEmptyRefreshJTI
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.entries[jti]
	if !ok {
		return false, nil
	}
	if existing.After(now) {
		return true, nil
	}
	delete(s.entries, jti)
	return false, nil
}

// CleanExpired 删除过期消费记录。
func (s *MemoryRefreshTokenUseStore) CleanExpired(ctx context.Context, now time.Time) (int, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}

	cleaned := 0
	s.mu.Lock()
	for jti, expiresAt := range s.entries {
		if !expiresAt.After(now) {
			delete(s.entries, jti)
			cleaned++
		}
	}
	s.mu.Unlock()
	return cleaned, nil
}
