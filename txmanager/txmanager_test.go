package txmanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 模拟事务接口
type mockTx struct {
	mock.Mock
}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *mockTx) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	args := m.Called(ctx, txOptions)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *mockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *mockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (m *mockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (m *mockTx) Conn() *pgx.Conn {
	return nil
}

func (m *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return &pgconn.StatementDescription{}, nil
}

// 模拟数据库连接池
type mockPool struct {
	mockTx   *mockTx
	beginErr error
}

func (m *mockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	return m.mockTx, nil
}

func (m *mockPool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	return m.mockTx, nil
}

func (m *mockPool) Close() {}

func (m *mockPool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	return nil, nil
}

func (m *mockPool) AcquireAllIdle(ctx context.Context) []*pgxpool.Conn {
	return nil
}

func (m *mockPool) AcquireFunc(ctx context.Context, f func(*pgxpool.Conn) error) error {
	return nil
}

func (m *mockPool) Config() *pgxpool.Config {
	return nil
}

func (m *mockPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockPool) Ping(ctx context.Context) error {
	return nil
}

func (m *mockPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (m *mockPool) Reset() {}

func (m *mockPool) Stat() *pgxpool.Stat {
	return nil
}

// 模拟Logger接口
type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// 模拟Metrics接口
type mockMetrics struct {
	mock.Mock
}

func (m *mockMetrics) RecordTransactionDuration(duration time.Duration) {
	m.Called(duration)
}

func (m *mockMetrics) IncrementTransactionCount() {
	m.Called()
}

func (m *mockMetrics) IncrementFailedTransactionCount() {
	m.Called()
}

func (m *mockMetrics) IncrementRetryCount() {
	m.Called()
}

// 测试基本事务功能
func TestRunInTransaction(t *testing.T) {
	// 创建模拟对象
	mockDB := new(mockTx)
	mockTx := new(mockTx)

	// 设置模拟行为
	mockDB.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
	mockTx.On("Commit", mock.Anything).Return(nil)

	// 创建事务管理器
	txManager := NewTxManager(mockDB)

	// 定义一个简单的事务函数
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		// 验证传递的事务是否正确
		assert.Equal(t, mockTx, tx)
		return nil
	}

	// 运行事务
	err := txManager.RunInTransaction(context.Background(), txFunc)

	// 验证结果
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

// 测试事务回滚
func TestTransactionRollback(t *testing.T) {
	// 创建模拟对象
	mockDB := new(mockTx)
	mockTx := new(mockTx)

	// 设置模拟行为
	mockDB.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	// 创建事务管理器
	txManager := NewTxManager(mockDB)

	// 定义一个会失败的事务函数
	expectedErr := errors.New("测试错误")
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		return expectedErr
	}

	// 运行事务
	err := txManager.RunInTransaction(context.Background(), txFunc)

	// 验证结果
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

// 测试事务嵌套
func TestNestedTransactions(t *testing.T) {
	// 创建模拟对象
	mockDB := new(mockTx)
	mockTx := new(mockTx)

	// 设置模拟行为 - 只应该调用一次BeginTx
	mockDB.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil).Once()
	mockTx.On("Commit", mock.Anything).Return(nil).Once()

	// 创建事务管理器
	txManager := NewTxManager(mockDB)

	// 定义外层事务函数
	outerFunc := func(ctx context.Context, tx pgx.Tx) error {
		// 嵌套事务函数
		innerFunc := func(ctx context.Context, tx pgx.Tx) error {
			return nil
		}

		// 在外层事务中运行嵌套事务
		return txManager.RunInTransaction(ctx, innerFunc)
	}

	// 运行事务
	err := txManager.RunInTransaction(context.Background(), outerFunc)

	// 验证结果
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

// 测试事务重试机制
func TestTransactionRetry(t *testing.T) {
	// 创建模拟对象
	mockDB := new(mockTx)
	mockTx1 := new(mockTx)
	mockTx2 := new(mockTx)
	mockLogger := new(mockLogger)
	mockMetrics := new(mockMetrics)

	// 设置死锁错误
	deadlockErr := &pgconn.PgError{
		Code: "40P01", // 死锁错误码
	}

	// 设置模拟行为 - 第一次失败，第二次成功
	mockDB.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx1, nil).Once()
	mockTx1.On("Rollback", mock.Anything).Return(nil).Once()
	mockDB.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx2, nil).Once()
	mockTx2.On("Commit", mock.Anything).Return(nil).Once()

	// 设置日志和指标记录
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Return(nil)
	mockMetrics.On("RecordTransactionDuration", mock.Anything).Return(nil)
	mockMetrics.On("IncrementTransactionCount").Return(nil)
	mockMetrics.On("IncrementFailedTransactionCount").Return(nil)
	mockMetrics.On("IncrementRetryCount").Return(nil)

	// 创建事务管理器
	txManager := NewTxManager(mockDB).
		WithLogger(mockLogger).
		WithMetrics(mockMetrics)

	// 定义事务函数 - 第一次失败，第二次成功
	var attempt int
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		attempt++
		if attempt == 1 {
			return deadlockErr
		}
		return nil
	}

	// 运行事务
	opts := TxOptions{
		MaxRetries:   3,
		RetryBackoff: 10 * time.Millisecond,
	}
	err := txManager.RunInTransactionWithRetry(context.Background(), opts, txFunc)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, 2, attempt)
	mockDB.AssertExpectations(t)
	mockTx1.AssertExpectations(t)
	mockTx2.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

// 测试链式API
func TestChainAPI(t *testing.T) {
	// 创建模拟对象
	mockDB := new(mockTx)
	mockTx := new(mockTx)

	// 设置模拟行为
	mockDB.On("BeginTx", mock.Anything, mock.MatchedBy(func(opts pgx.TxOptions) bool {
		return opts.IsoLevel == pgx.Serializable && opts.AccessMode == pgx.ReadWrite
	})).Return(mockTx, nil)
	mockTx.On("Commit", mock.Anything).Return(nil)

	// 创建事务管理器
	txManager := NewTxManager(mockDB)

	// 定义事务函数
	var executed bool
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		executed = true
		return nil
	}

	// 使用链式API运行事务
	err := txManager.Begin().
		WithContext(context.Background()).
		WithIsolation(pgx.Serializable).
		WithAccessMode(pgx.ReadWrite).
		WithRetry(2).
		AddFunc(txFunc).
		Run()

	// 验证结果
	assert.NoError(t, err)
	assert.True(t, executed)
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

// 测试事务超时
func TestTransactionTimeout(t *testing.T) {
	// 创建模拟对象
	mockDB := new(mockTx)
	mockTx := new(mockTx)

	// 设置模拟行为 - 使用匹配任何context和options的参数
	mockDB.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	// 创建事务管理器
	txManager := NewTxManager(mockDB)

	// 定义一个长时间运行的事务函数
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// 运行事务，设置很短的超时时间
	ctx := context.Background()
	err := txManager.Begin().
		WithContext(ctx).
		WithTimeout(10 * time.Millisecond).
		Run(txFunc)

	// 验证结果 - 应该超时
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

// 测试无事务函数
func TestNoTransactionFunctions(t *testing.T) {
	// 创建模拟对象
	mockDB := new(mockTx)
	mockLogger := new(mockLogger)

	// 设置日志预期
	mockLogger.On("Error", mock.Anything, mock.Anything).Return(nil)
	// 注意：此处不需要预期Debug调用，因为空事务函数情况下只会记录错误

	// 创建事务管理器
	txManager := NewTxManager(mockDB).WithLogger(mockLogger)

	// 运行没有事务函数的事务
	err := txManager.RunInTransaction(context.Background())

	// 验证结果
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "没有提供事务函数")
	mockLogger.AssertExpectations(t)
}
