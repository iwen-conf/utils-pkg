package jwt

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryRefreshTokenUseStore_ConsumeAndExpire(t *testing.T) {
	store := NewMemoryRefreshTokenUseStore()
	exp := time.Now().Add(time.Minute)

	first, err := store.Consume(context.Background(), "jti-1", exp)
	if err != nil || !first {
		t.Fatalf("first consume: first=%v err=%v", first, err)
	}
	second, err := store.Consume(context.Background(), "jti-1", exp)
	if err != nil || second {
		t.Fatalf("second consume should not be first: first=%v err=%v", second, err)
	}

	consumed, err := store.IsConsumed(context.Background(), "jti-1")
	if err != nil || !consumed {
		t.Fatalf("IsConsumed: consumed=%v err=%v", consumed, err)
	}

	store.entries["expired"] = time.Now().Add(-time.Second)
	first, err = store.Consume(context.Background(), "expired", time.Now().Add(time.Minute))
	if err != nil || !first {
		t.Fatalf("expired jti should be reusable: first=%v err=%v", first, err)
	}
}

func TestMemoryRefreshTokenUseStore_ConcurrentSingleFirstUse(t *testing.T) {
	store := NewMemoryRefreshTokenUseStore()
	exp := time.Now().Add(time.Minute)
	var firstCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			first, err := store.Consume(context.Background(), "same", exp)
			if err != nil {
				t.Errorf("Consume: %v", err)
				return
			}
			if first {
				firstCount.Add(1)
			}
		}()
	}
	wg.Wait()
	if firstCount.Load() != 1 {
		t.Fatalf("expected exactly one first use, got %d", firstCount.Load())
	}
}

func TestMemoryRefreshTokenUseStore_InvalidInput(t *testing.T) {
	store := NewMemoryRefreshTokenUseStore()
	_, err := store.Consume(context.Background(), "", time.Now().Add(time.Minute))
	if !errors.Is(err, ErrEmptyRefreshJTI) {
		t.Fatalf("empty jti: %v", err)
	}
	_, err = store.IsConsumed(context.Background(), "")
	if !errors.Is(err, ErrEmptyRefreshJTI) {
		t.Fatalf("IsConsumed empty jti: %v", err)
	}
	_, err = store.Consume(context.Background(), "jti", time.Now().Add(-time.Second))
	if !errors.Is(err, ErrRefreshJTINotFuture) {
		t.Fatalf("past expiry: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.Consume(ctx, "jti", time.Now().Add(time.Minute))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context: %v", err)
	}
}

func TestRefreshToken_SharedStoreSingleFirstUse(t *testing.T) {
	store := NewMemoryRefreshTokenUseStore()
	revocations := NewMemoryRevocationStore()
	opts1 := DefaultJWTOptions()
	opts1.RefreshTokenUseStore = store
	opts1.RevocationStore = revocations
	opts2 := DefaultJWTOptions()
	opts2.RefreshTokenUseStore = store
	opts2.RevocationStore = revocations

	m1 := MustNewTokenManager(testSecretKey, opts1)
	m2 := MustNewTokenManager(testSecretKey, opts2)
	defer m1.Shutdown()
	defer m2.Shutdown()

	refresh, err := m1.GenerateToken("u-1", &TokenOptions{TokenType: RefreshToken, SessionID: "sid-1"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	type result struct {
		err error
	}
	results := make(chan result, 2)
	go func() {
		_, _, err := m1.RefreshToken(refresh)
		results <- result{err: err}
	}()
	go func() {
		_, _, err := m2.RefreshToken(refresh)
		results <- result{err: err}
	}()

	var successes, reuses int
	for i := 0; i < 2; i++ {
		got := <-results
		switch {
		case got.err == nil:
			successes++
		case errors.Is(got.err, ErrRefreshReuseDetected):
			reuses++
		default:
			t.Fatalf("unexpected refresh error: %v", got.err)
		}
	}
	if successes != 1 || reuses != 1 {
		t.Fatalf("expected 1 success and 1 reuse, got success=%d reuse=%d", successes, reuses)
	}

	if _, err := m1.ValidateRefreshToken(context.Background(), refresh); !errors.Is(err, ErrTokenRevoked) && !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("old refresh should be revoked, got %v", err)
	}
}

func TestRefreshToken_StoreErrorDoesNotIssueTokens(t *testing.T) {
	store := &failingRefreshStore{consumeErr: errors.New("store down")}
	opts := DefaultJWTOptions()
	opts.RefreshTokenUseStore = store
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	refresh, err := tm.GenerateToken("u-1", &TokenOptions{TokenType: RefreshToken, SessionID: "sid-1"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	access, newRefresh, err := tm.RefreshToken(refresh)
	if err == nil || access != "" || newRefresh != "" {
		t.Fatalf("store error must not issue tokens: access=%q refresh=%q err=%v", access, newRefresh, err)
	}
	if !errors.Is(err, store.consumeErr) {
		t.Fatalf("expected store error in chain, got %v", err)
	}
}

type failingRefreshStore struct {
	consumeErr error
	isErr      error
}

func (s *failingRefreshStore) Consume(ctx context.Context, jti string, expiresAt time.Time) (bool, error) {
	return false, s.consumeErr
}

func (s *failingRefreshStore) IsConsumed(ctx context.Context, jti string) (bool, error) {
	return false, s.isErr
}
