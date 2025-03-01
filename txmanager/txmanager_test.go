package txmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 模拟事务接口
type mockTx struct {
	commitCalled   bool
	rollbackCalled bool
	commitErr      error
	rollbackErr    error
}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return m, nil
}

func (m *mockTx) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return m, nil
}

func (m *mockTx) Commit(ctx context.Context) error {
	m.commitCalled = true
	return m.commitErr
}

func (m *mockTx) Rollback(ctx context.Context) error {
	m.rollbackCalled = true
	return m.rollbackErr
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
	mockTx *mockTx
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

func TestRunInTransaction_Success(t *testing.T) {
	// 创建模拟对象
	mockTx := &mockTx{}
	mockDB := &mockPool{mockTx: mockTx}
	txManager := NewTxManager(mockDB)

	// 创建成功的事务函数
	func1Called := false
	func2Called := false

	func1 := func(ctx context.Context, tx pgx.Tx) error {
		func1Called = true
		return nil
	}

	func2 := func(ctx context.Context, tx pgx.Tx) error {
		func2Called = true
		return nil
	}

	// 执行事务
	err := txManager.RunInTransaction(context.Background(), func1, func2)

	// 验证结果
	if err != nil {
		t.Errorf("期望无错误，但得到: %v", err)
	}

	if !func1Called {
		t.Error("第一个事务函数未被调用")
	}

	if !func2Called {
		t.Error("第二个事务函数未被调用")
	}

	if !mockTx.commitCalled {
		t.Error("事务未被提交")
	}

	if mockTx.rollbackCalled {
		t.Error("事务被错误地回滚")
	}
}

func TestRunInTransaction_FirstFuncFails(t *testing.T) {
	// 创建模拟对象
	mockTx := &mockTx{}
	mockDB := &mockPool{mockTx: mockTx}
	txManager := NewTxManager(mockDB)

	// 创建事务函数，第一个会失败
	func1Called := false
	func2Called := false
	expectedErr := errors.New("第一个函数失败")

	func1 := func(ctx context.Context, tx pgx.Tx) error {
		func1Called = true
		return expectedErr
	}

	func2 := func(ctx context.Context, tx pgx.Tx) error {
		func2Called = true
		return nil
	}

	// 执行事务
	err := txManager.RunInTransaction(context.Background(), func1, func2)

	// 验证结果
	if err == nil {
		t.Error("期望有错误，但没有得到错误")
	}

	if !func1Called {
		t.Error("第一个事务函数未被调用")
	}

	if func2Called {
		t.Error("第二个事务函数不应该被调用")
	}

	if mockTx.commitCalled {
		t.Error("事务不应该被提交")
	}

	if !mockTx.rollbackCalled {
		t.Error("事务应该被回滚")
	}
}

func TestRunInTransaction_SecondFuncFails(t *testing.T) {
	// 创建模拟对象
	mockTx := &mockTx{}
	mockDB := &mockPool{mockTx: mockTx}
	txManager := NewTxManager(mockDB)

	// 创建事务函数，第二个会失败
	func1Called := false
	func2Called := false
	expectedErr := errors.New("第二个函数失败")

	func1 := func(ctx context.Context, tx pgx.Tx) error {
		func1Called = true
		return nil
	}

	func2 := func(ctx context.Context, tx pgx.Tx) error {
		func2Called = true
		return expectedErr
	}

	// 执行事务
	err := txManager.RunInTransaction(context.Background(), func1, func2)

	// 验证结果
	if err == nil {
		t.Error("期望有错误，但没有得到错误")
	}

	if !func1Called {
		t.Error("第一个事务函数未被调用")
	}

	if !func2Called {
		t.Error("第二个事务函数未被调用")
	}

	if mockTx.commitCalled {
		t.Error("事务不应该被提交")
	}

	if !mockTx.rollbackCalled {
		t.Error("事务应该被回滚")
	}
}

func TestRunInTransaction_BeginFails(t *testing.T) {
	// 创建模拟对象，Begin会失败
	expectedErr := errors.New("开始事务失败")
	mockDB := &mockPool{beginErr: expectedErr}
	txManager := NewTxManager(mockDB)

	// 创建事务函数
	funcCalled := false
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		funcCalled = true
		return nil
	}

	// 执行事务
	err := txManager.RunInTransaction(context.Background(), txFunc)

	// 验证结果
	if err == nil {
		t.Error("期望有错误，但没有得到错误")
	}

	if funcCalled {
		t.Error("事务函数不应该被调用")
	}
}

func TestRunInTransaction_CommitFails(t *testing.T) {
	// 创建模拟对象，Commit会失败
	expectedErr := errors.New("提交事务失败")
	mockTx := &mockTx{commitErr: expectedErr}
	mockDB := &mockPool{mockTx: mockTx}
	txManager := NewTxManager(mockDB)

	// 创建事务函数
	funcCalled := false
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		funcCalled = true
		return nil
	}

	// 执行事务
	err := txManager.RunInTransaction(context.Background(), txFunc)

	// 验证结果
	if err == nil {
		t.Error("期望有错误，但没有得到错误")
	}

	if !funcCalled {
		t.Error("事务函数应该被调用")
	}

	if !mockTx.commitCalled {
		t.Error("应该尝试提交事务")
	}

	if mockTx.rollbackCalled {
		t.Error("不应该尝试回滚事务")
	}
}

func TestRunInTransactionWithOptions(t *testing.T) {
	// 创建模拟对象
	mockTx := &mockTx{}
	mockDB := &mockPool{mockTx: mockTx}
	txManager := NewTxManager(mockDB)

	// 创建事务函数
	funcCalled := false
	txFunc := func(ctx context.Context, tx pgx.Tx) error {
		funcCalled = true
		return nil
	}

	// 创建事务选项
	opts := pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	}

	// 执行事务
	err := txManager.RunInTransactionWithOptions(context.Background(), opts, txFunc)

	// 验证结果
	if err != nil {
		t.Errorf("期望无错误，但得到: %v", err)
	}

	if !funcCalled {
		t.Error("事务函数未被调用")
	}

	if !mockTx.commitCalled {
		t.Error("事务未被提交")
	}

	if mockTx.rollbackCalled {
		t.Error("事务被错误地回滚")
	}
}

func TestRunInTransaction_NoFuncs(t *testing.T) {
	// 创建模拟对象
	mockTx := &mockTx{}
	mockDB := &mockPool{mockTx: mockTx}
	txManager := NewTxManager(mockDB)

	// 执行事务，不提供任何函数
	err := txManager.RunInTransaction(context.Background())

	// 验证结果
	if err == nil {
		t.Error("期望有错误，但没有得到错误")
	}

	if mockTx.commitCalled {
		t.Error("事务不应该被提交")
	}

	if mockTx.rollbackCalled {
		t.Error("事务不应该被回滚")
	}
}