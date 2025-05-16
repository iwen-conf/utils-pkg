package txmanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// 上下文键类型，用于在上下文中存取值
type contextKey string

// 上下文键常量
const (
	// ActiveTxKey 用于存储当前活动事务
	ActiveTxKey contextKey = "active_tx"
	// LoggerKey 用于存储日志记录器
	LoggerKey contextKey = "tx_logger"
	// MetricsKey 用于存储指标收集器
	MetricsKey contextKey = "tx_metrics"
)

// TxFunc 定义在事务中执行的函数类型
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// DBConn 定义数据库连接接口
type DBConn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// Logger 定义日志接口
type Logger interface {
	Error(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// Metrics 定义指标收集接口
type Metrics interface {
	RecordTransactionDuration(duration time.Duration)
	IncrementTransactionCount()
	IncrementFailedTransactionCount()
	IncrementRetryCount()
}

// TxOptions 扩展事务选项
type TxOptions struct {
	pgx.TxOptions
	MaxRetries   int
	RetryBackoff time.Duration
}

// DefaultTxOptions 返回默认事务选项
func DefaultTxOptions() TxOptions {
	return TxOptions{
		TxOptions:    pgx.TxOptions{IsoLevel: pgx.ReadCommitted},
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,
	}
}

// TxManager 事务管理器
type TxManager struct {
	db      DBConn
	logger  Logger
	metrics Metrics
}

// NewTxManager 创建一个新的事务管理器
func NewTxManager(db DBConn) *TxManager {
	return &TxManager{db: db}
}

// WithLogger 设置日志记录器
func (tm *TxManager) WithLogger(logger Logger) *TxManager {
	tm.logger = logger
	return tm
}

// WithMetrics 设置指标收集器
func (tm *TxManager) WithMetrics(metrics Metrics) *TxManager {
	tm.metrics = metrics
	return tm
}

// 获取上下文中的活动事务
func GetActiveTx(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(ActiveTxKey).(pgx.Tx)
	return tx
}

// 创建带有活动事务的上下文
func withActiveTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, ActiveTxKey, tx)
}

// 记录错误
func (tm *TxManager) logError(ctx context.Context, msg string, err error, additionalFields ...interface{}) {
	if tm.logger != nil {
		// 添加上下文中的信息（如果有）
		fields := []interface{}{"error", err}

		// 如果上下文包含日志记录器，可以使用它
		if ctx != nil {
			if logger, ok := ctx.Value(LoggerKey).(Logger); ok && logger != nil {
				fields = append(fields, "context_logger", "present")
			}
		}

		// 添加其他字段
		if len(additionalFields) > 0 {
			fields = append(fields, additionalFields...)
		}

		tm.logger.Error(msg, fields...)
	}
}

// 记录信息
func (tm *TxManager) logInfo(ctx context.Context, msg string, fields ...interface{}) {
	if tm.logger != nil {
		// 添加上下文中的信息（如果有）
		extraFields := []interface{}{}

		// 如果上下文包含日志记录器，可以使用它
		if ctx != nil {
			if logger, ok := ctx.Value(LoggerKey).(Logger); ok && logger != nil {
				extraFields = append(extraFields, "context_logger", "present")
			}
		}

		allFields := append(extraFields, fields...)
		tm.logger.Info(msg, allFields...)
	}
}

// 记录调试信息
func (tm *TxManager) logDebug(ctx context.Context, msg string, fields ...interface{}) {
	if tm.logger != nil {
		// 添加上下文中的信息（如果有）
		extraFields := []interface{}{}

		// 如果上下文包含日志记录器，可以使用它
		if ctx != nil {
			if logger, ok := ctx.Value(LoggerKey).(Logger); ok && logger != nil {
				extraFields = append(extraFields, "context_logger", "present")
			}
		}

		allFields := append(extraFields, fields...)
		tm.logger.Debug(msg, allFields...)
	}
}

// 记录指标
func (tm *TxManager) recordMetrics(duration time.Duration, err error) {
	if tm.metrics == nil {
		return
	}

	tm.metrics.RecordTransactionDuration(duration)
	tm.metrics.IncrementTransactionCount()

	if err != nil {
		tm.metrics.IncrementFailedTransactionCount()
	}
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	// 目前只处理死锁和序列化失败
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 40001: serialization_failure, 40P01: deadlock_detected
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}

// RunInTransaction 在单个事务中执行多个函数
// 如果任何一个函数返回错误，事务将被回滚
// 如果所有函数成功执行，事务将被提交
func (tm *TxManager) RunInTransaction(ctx context.Context, txFuncs ...TxFunc) error {
	return tm.RunInTransactionWithOptions(ctx, pgx.TxOptions{}, txFuncs...)
}

// RunInTransactionWithOptions 在单个事务中执行多个函数，支持自定义事务选项
func (tm *TxManager) RunInTransactionWithOptions(ctx context.Context, opts pgx.TxOptions, txFuncs ...TxFunc) error {
	startTime := time.Now()
	var err error
	defer func() {
		tm.recordMetrics(time.Since(startTime), err)
	}()

	if len(txFuncs) == 0 {
		err = errors.New("没有提供事务函数")
		tm.logError(ctx, "事务执行失败", err)
		return err
	}

	// 检查上下文中是否已有活动事务
	if tx := GetActiveTx(ctx); tx != nil {
		tm.logDebug(ctx, "使用已存在的事务")
		// 直接使用已存在的事务
		for i, txFunc := range txFuncs {
			if err = txFunc(ctx, tx); err != nil {
				err = fmt.Errorf("事务函数 %d 执行失败: %w", i+1, err)
				tm.logError(ctx, "事务嵌套执行失败", err, "function_index", i+1)
				return err
			}
		}
		return nil
	}

	// 开始新事务
	tm.logDebug(ctx, "开始新事务", "isolation_level", opts.IsoLevel)
	tx, txErr := tm.db.BeginTx(ctx, opts)
	if txErr != nil {
		err = fmt.Errorf("开始事务失败: %w", txErr)
		tm.logError(ctx, "开始事务失败", txErr)
		return err
	}

	// 创建带事务的上下文，供嵌套事务使用
	txCtx := withActiveTx(ctx, tx)

	// 确保事务最终会被提交或回滚
	var committed bool
	defer func() {
		if !committed && tx != nil {
			// 如果没有提交，则尝试回滚
			rErr := tx.Rollback(ctx)
			if rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
				tm.logError(ctx, "事务回滚失败", rErr, "original_error", err)
			}
		}
	}()

	// 依次执行所有事务函数
	for i, txFunc := range txFuncs {
		if err = txFunc(txCtx, tx); err != nil {
			err = fmt.Errorf("事务函数 %d 执行失败: %w", i+1, err)
			tm.logError(ctx, "事务函数执行失败", err, "function_index", i+1)
			return err
		}
	}

	// 提交事务
	if err = tx.Commit(ctx); err != nil {
		err = fmt.Errorf("提交事务失败: %w", err)
		tm.logError(ctx, "提交事务失败", err)
		return err
	}

	committed = true
	tm.logDebug(ctx, "事务成功提交")
	return nil
}

// RunInTransactionWithRetry 执行带重试机制的事务
func (tm *TxManager) RunInTransactionWithRetry(ctx context.Context, opts TxOptions, txFuncs ...TxFunc) error {
	var lastErr error

	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		if attempt > 0 {
			tm.logInfo(ctx, "重试事务", "attempt", attempt, "max_retries", opts.MaxRetries)
			if tm.metrics != nil {
				tm.metrics.IncrementRetryCount()
			}

			// 计算退避时间
			backoff := opts.RetryBackoff * time.Duration(1<<uint(attempt-1))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// 继续重试
			}
		}

		err := tm.RunInTransactionWithOptions(ctx, opts.TxOptions, txFuncs...)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否是可重试的错误
		if !isRetryableError(err) {
			tm.logInfo(ctx, "事务错误不可重试", "error", err)
			return err
		}

		tm.logInfo(ctx, "检测到可重试的事务错误", "error", err, "attempt", attempt+1)
	}

	return fmt.Errorf("事务重试耗尽 (%d 次尝试): %w", opts.MaxRetries+1, lastErr)
}

// RunInTransactionWithTimeout 在设定超时的事务中执行函数
func (tm *TxManager) RunInTransactionWithTimeout(ctx context.Context, timeout time.Duration, txFuncs ...TxFunc) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return tm.RunInTransaction(ctx, txFuncs...)
}

// TxBuilder 事务构建器，提供链式API
type TxBuilder struct {
	manager *TxManager
	ctx     context.Context
	options TxOptions
	txFuncs []TxFunc
}

// Begin 开始构建事务
func (tm *TxManager) Begin() *TxBuilder {
	return &TxBuilder{
		manager: tm,
		ctx:     context.Background(),
		options: DefaultTxOptions(),
	}
}

// WithContext 设置上下文
func (b *TxBuilder) WithContext(ctx context.Context) *TxBuilder {
	b.ctx = ctx
	return b
}

// WithIsolation 设置隔离级别
func (b *TxBuilder) WithIsolation(level pgx.TxIsoLevel) *TxBuilder {
	b.options.IsoLevel = level
	return b
}

// WithAccessMode 设置访问模式
func (b *TxBuilder) WithAccessMode(mode pgx.TxAccessMode) *TxBuilder {
	b.options.AccessMode = mode
	return b
}

// WithDeferrable 设置是否可延迟
func (b *TxBuilder) WithDeferrable(deferrable bool) *TxBuilder {
	// 注意：当前pgx/v5中TxOptions没有Deferrable字段，此方法仅为接口完整性保留
	// 实际使用前请检查pgx版本是否支持
	return b
}

// WithRetry 设置重试次数
func (b *TxBuilder) WithRetry(count int) *TxBuilder {
	b.options.MaxRetries = count
	return b
}

// WithRetryBackoff 设置重试退避时间
func (b *TxBuilder) WithRetryBackoff(backoff time.Duration) *TxBuilder {
	b.options.RetryBackoff = backoff
	return b
}

// WithTimeout 设置超时时间
func (b *TxBuilder) WithTimeout(timeout time.Duration) *TxBuilder {
	ctx, cancel := context.WithTimeout(b.ctx, timeout)
	// 将cancel函数保存到上下文中，以便后续调用
	b.ctx = context.WithValue(ctx, contextKey("cancel_func"), cancel)
	return b
}

// AddFunc 添加事务函数
func (b *TxBuilder) AddFunc(txFunc TxFunc) *TxBuilder {
	b.txFuncs = append(b.txFuncs, txFunc)
	return b
}

// Run 执行事务
func (b *TxBuilder) Run(txFuncs ...TxFunc) error {
	// 调用之前保存的cancel函数(如果有)
	if cancel, ok := b.ctx.Value(contextKey("cancel_func")).(context.CancelFunc); ok && cancel != nil {
		defer cancel()
	}

	allFuncs := append(b.txFuncs, txFuncs...)

	if b.options.MaxRetries > 0 {
		return b.manager.RunInTransactionWithRetry(b.ctx, b.options, allFuncs...)
	}

	return b.manager.RunInTransactionWithOptions(b.ctx, b.options.TxOptions, allFuncs...)
}
