package txmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxFunc 定义在事务中执行的函数类型
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// DBConn 定义数据库连接接口
type DBConn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// TxManager 事务管理器
type TxManager struct {
	db DBConn
}

// NewTxManager 创建一个新的事务管理器
func NewTxManager(db DBConn) *TxManager {
	return &TxManager{db: db}
}

// RunInTransaction 在单个事务中执行多个函数
// 如果任何一个函数返回错误，事务将被回滚
// 如果所有函数成功执行，事务将被提交
func (tm *TxManager) RunInTransaction(ctx context.Context, txFuncs ...TxFunc) error {
	if len(txFuncs) == 0 {
		return errors.New("没有提供事务函数")
	}

	// 开始事务
	tx, err := tm.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	// 依次执行所有事务函数
	for i, txFunc := range txFuncs {
		if err = txFunc(ctx, tx); err != nil {
			// 如果函数执行失败，回滚事务
			rErr := tx.Rollback(ctx)
			if rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
				// 记录回滚错误，但不覆盖原始错误
				fmt.Printf("事务回滚失败: %v\n", rErr)
			}
			return fmt.Errorf("事务函数 %d 执行失败: %w", i+1, err)
		}
	}

	// 提交事务
	if cErr := tx.Commit(ctx); cErr != nil {
		return fmt.Errorf("提交事务失败: %w", cErr)
	}

	return nil
}

// RunInTransactionWithOptions 在单个事务中执行多个函数，支持自定义事务选项
func (tm *TxManager) RunInTransactionWithOptions(ctx context.Context, opts pgx.TxOptions, txFuncs ...TxFunc) error {
	if len(txFuncs) == 0 {
		return errors.New("没有提供事务函数")
	}

	// 开始事务，使用自定义选项
	tx, err := tm.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	// 确保事务最终会被提交或回滚
	defer func() {
		if err != nil {
			// 如果有错误发生，回滚事务
			rErr := tx.Rollback(ctx)
			if rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
				// 记录回滚错误，但不覆盖原始错误
				fmt.Printf("事务回滚失败: %v\n", rErr)
			}
		} else {
			// 如果没有错误，提交事务
			cErr := tx.Commit(ctx)
			if cErr != nil {
				err = fmt.Errorf("提交事务失败: %w", cErr)
			}
		}
	}()

	// 依次执行所有事务函数
	for i, txFunc := range txFuncs {
		if err = txFunc(ctx, tx); err != nil {
			return fmt.Errorf("事务函数 %d 执行失败: %w", i+1, err)
		}
	}

	return nil
}