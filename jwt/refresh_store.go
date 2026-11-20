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

// RefreshTokenUseReleaser 是可选的消费回滚接口。
// 当 refresh 轮换在 Consume 成功后因后续吊销存储瞬时故障而中止时，
// 实现此接口的存储可回滚该次未完成的消费记录，使合法客户端重试不会误触重放封禁。
type RefreshTokenUseReleaser interface {
	Release(ctx context.Context, jti string) error
}

// MemoryRefreshTokenUseStore 提供单进程默认 refresh 消费存储。
type MemoryRefreshTokenUseStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
	nowFunc func() time.Time
}

var (
	_ RefreshTokenUseStore        = (*MemoryRefreshTokenUseStore)(nil)
	_ RefreshTokenUseStoreCleaner = (*MemoryRefreshTokenUseStore)(nil)
	_ RefreshTokenUseReleaser     = (*MemoryRefreshTokenUseStore)(nil)
)

// NewMemoryRefreshTokenUseStore 创建内存 refresh 消费存储。
func NewMemoryRefreshTokenUseStore() *MemoryRefreshTokenUseStore {
	return NewMemoryRefreshTokenUseStoreWithClock(nil)
}

// NewMemoryRefreshTokenUseStoreWithClock 创建带可控时钟的内存 refresh 消费存储。
func NewMemoryRefreshTokenUseStoreWithClock(clock func() time.Time) *MemoryRefreshTokenUseStore {
	if clock == nil {
		clock = time.Now
	}
	return &MemoryRefreshTokenUseStore{
		entries: make(map[string]time.Time),
		nowFunc: clock,
	}
}

func (s *MemoryRefreshTokenUseStore) now() time.Time {
	if s != nil && s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

// Consume 在同一临界区内检查未过期记录并写入。首次消费返回 firstUse=true。
func (s *MemoryRefreshTokenUseStore) Consume(ctx context.Context, jti string, expiresAt time.Time) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	if jti == "" {
		return false, ErrEmptyRefreshJTI
	}
	now := s.now()
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
	if len(s.entries) >= opportunisticSweepThreshold {
		swept := 0
		for k, exp := range s.entries {
			if !exp.After(now) {
				delete(s.entries, k)
			}
			swept++
			if swept >= opportunisticSweepBudget {
				break
			}
		}
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

	now := s.now()
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

// Release 回滚尚未完成轮换提交的 jti 消费记录。
func (s *MemoryRefreshTokenUseStore) Release(ctx context.Context, jti string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if jti == "" {
		return ErrEmptyRefreshJTI
	}
	s.mu.Lock()
	delete(s.entries, jti)
	s.mu.Unlock()
	return nil
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
