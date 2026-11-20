package jwt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	// ErrTokenRevoked 表示令牌已被提前吊销。
	ErrTokenRevoked = errors.New("token has been revoked")
	// ErrSessionRevoked 表示令牌所属会话已被提前吊销。
	ErrSessionRevoked = errors.New("session has been revoked")
	// ErrSessionIDMissing 表示令牌或调用参数不包含可吊销的会话 ID。
	ErrSessionIDMissing = errors.New("token session id is missing")
)

const opportunisticSweepThreshold = 1024
const opportunisticSweepBudget = 16

// RevocationStore 保存令牌和会话的提前吊销状态。
//
// key 由 TokenManager 生成且不包含原始令牌，外部实现只需按 expiresAt
// 保存并查询该不透明 key。缓存或数据库实现都不应自行解析 JWT。
type RevocationStore interface {
	Revoke(ctx context.Context, key string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, key string) (bool, error)
}

// RevocationStoreCleaner 是支持主动清理过期记录的可选能力。
// 带原生 TTL 的外部存储无需实现此接口。
type RevocationStoreCleaner interface {
	CleanExpired(ctx context.Context, now time.Time) (int, error)
}

// RevocationStoreSizer 是支持统计当前吊销记录数的可选能力。
type RevocationStoreSizer interface {
	Size(ctx context.Context) (int, error)
}

// MemoryRevocationStore 提供单进程默认吊销存储。
// 需要跨进程或重启后保留吊销状态时，应注入外部 RevocationStore。
type MemoryRevocationStore struct {
	mu      sync.RWMutex
	entries map[string]time.Time
	nowFunc func() time.Time
}

var (
	_ RevocationStore        = (*MemoryRevocationStore)(nil)
	_ RevocationStoreCleaner = (*MemoryRevocationStore)(nil)
	_ RevocationStoreSizer   = (*MemoryRevocationStore)(nil)
)

// NewMemoryRevocationStore 创建内存吊销存储。
func NewMemoryRevocationStore() *MemoryRevocationStore {
	return NewMemoryRevocationStoreWithClock(nil)
}

// NewMemoryRevocationStoreWithClock 创建带可控时钟的内存吊销存储。
func NewMemoryRevocationStoreWithClock(clock func() time.Time) *MemoryRevocationStore {
	if clock == nil {
		clock = time.Now
	}
	return &MemoryRevocationStore{
		entries: make(map[string]time.Time),
		nowFunc: clock,
	}
}

func (s *MemoryRevocationStore) now() time.Time {
	if s != nil && s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

// Revoke 写入或延长一条吊销记录。
func (s *MemoryRevocationStore) Revoke(ctx context.Context, key string, expiresAt time.Time) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if key == "" {
		return errors.New("revocation key cannot be empty")
	}

	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[string]time.Time)
	}
	if !expiresAt.After(now) {
		delete(s.entries, key)
		return nil
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
	if current, ok := s.entries[key]; !ok || expiresAt.After(current) {
		s.entries[key] = expiresAt
	}
	return nil
}

// IsRevoked 查询仍在有效期内的吊销记录。
func (s *MemoryRevocationStore) IsRevoked(ctx context.Context, key string) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	if key == "" {
		return false, nil
	}

	now := s.now()
	s.mu.RLock()
	expiresAt, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if expiresAt.After(now) {
		return true, nil
	}

	s.mu.Lock()
	if current, exists := s.entries[key]; exists && !current.After(now) {
		delete(s.entries, key)
	}
	s.mu.Unlock()
	return false, nil
}

// CleanExpired 清理过期吊销记录。
func (s *MemoryRevocationStore) CleanExpired(ctx context.Context, now time.Time) (int, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}

	cleaned := 0
	s.mu.Lock()
	for key, expiresAt := range s.entries {
		if !expiresAt.After(now) {
			delete(s.entries, key)
			cleaned++
		}
	}
	s.mu.Unlock()
	return cleaned, nil
}

// Size 返回当前存储的未过期吊销记录数。
func (s *MemoryRevocationStore) Size(ctx context.Context) (int, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries), nil
}

func tokenRevocationKey(token string) string {
	return revocationKey("token:", token)
}

func sessionRevocationKey(sessionID string) string {
	return revocationKey("session:", sessionID)
}

func revocationKey(prefix string, value string) string {
	return prefix + hashRevocationValue(value)
}

func tokenLogID(token string) string {
	digest := hashRevocationValue(token)
	return digest[:12]
}

func hashRevocationValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
