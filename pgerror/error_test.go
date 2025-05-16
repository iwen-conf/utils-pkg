package pgerror

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// 创建模拟的 PgError
func createMockPgError(code string, message, detail, hint, table, column, constraint string) *pgconn.PgError {
	return &pgconn.PgError{
		Code:           code,
		Message:        message,
		Detail:         detail,
		Hint:           hint,
		TableName:      table,
		ColumnName:     column,
		ConstraintName: constraint,
	}
}

func TestWrapDBError(t *testing.T) {
	t.Run("空错误测试", func(t *testing.T) {
		err := WrapDBError(nil)
		assert.Nil(t, err)
	})

	t.Run("非PG错误测试", func(t *testing.T) {
		originalErr := errors.New("普通错误")
		err := WrapDBError(originalErr)

		dbErr, ok := err.(*DBError)
		assert.True(t, ok)
		assert.Equal(t, "UNKNOWN", dbErr.Code)
		assert.Equal(t, "普通错误", dbErr.Message)
		assert.Equal(t, CategoryUnknown, dbErr.Category)
	})
}

func TestForeignKeyViolation(t *testing.T) {
	pgErr := createMockPgError(
		ForeignKeyViolation,
		"违反外键约束",
		"Key (user_id)=(123) is not present in table \"users\"",
		"",
		"orders",
		"user_id",
		"fk_orders_users",
	)

	err := WrapDBError(pgErr)
	dbErr, ok := err.(*DBError)
	assert.True(t, ok)

	assert.Equal(t, ForeignKeyViolation, dbErr.Code)
	assert.Contains(t, dbErr.Message, "数据关联错误")
	assert.Contains(t, dbErr.Message, "orders")
	assert.Contains(t, dbErr.Message, "users")
	assert.Contains(t, dbErr.Hint, "user_id")
}

func TestUniqueViolation(t *testing.T) {
	pgErr := createMockPgError(
		UniqueViolation,
		"违反唯一约束",
		"Key (email)=(test@example.com) already exists.",
		"",
		"users",
		"email",
		"users_email_key",
	)

	err := WrapDBError(pgErr)
	dbErr, ok := err.(*DBError)
	assert.True(t, ok)

	assert.Equal(t, UniqueViolation, dbErr.Code)
	assert.Contains(t, dbErr.Message, "数据重复错误")
	assert.Contains(t, dbErr.Message, "users")
	assert.Contains(t, dbErr.Message, "email")
	assert.Contains(t, dbErr.Hint, "test@example.com")
}

func TestNotNullViolation(t *testing.T) {
	pgErr := createMockPgError(
		NotNullViolation,
		"违反非空约束",
		"Null value in column \"name\"",
		"",
		"users",
		"name",
		"",
	)

	err := WrapDBError(pgErr)
	dbErr, ok := err.(*DBError)
	assert.True(t, ok)

	assert.Equal(t, NotNullViolation, dbErr.Code)
	assert.Contains(t, dbErr.Message, "数据完整性错误")
	assert.Contains(t, dbErr.Message, "users")
	assert.Contains(t, dbErr.Message, "name")
	assert.Contains(t, dbErr.Hint, "请提供必要的数据值")
}

func TestCheckViolation(t *testing.T) {
	pgErr := createMockPgError(
		CheckViolation,
		"违反检查约束",
		"New row for table \"products\" violates check constraint \"price_positive\"",
		"",
		"products",
		"price",
		"price_positive",
	)

	err := WrapDBError(pgErr)
	dbErr, ok := err.(*DBError)
	assert.True(t, ok)

	assert.Equal(t, CheckViolation, dbErr.Code)
	assert.Contains(t, dbErr.Message, "数据验证错误")
	assert.Contains(t, dbErr.Message, "products")
	assert.Contains(t, dbErr.Message, "price_positive")
}

func TestInsufficientPrivilege(t *testing.T) {
	pgErr := createMockPgError(
		InsufficientPrivilege,
		"permission denied for table users",
		"",
		"",
		"users",
		"",
		"",
	)

	err := WrapDBError(pgErr)
	dbErr, ok := err.(*DBError)
	assert.True(t, ok)

	assert.Equal(t, InsufficientPrivilege, dbErr.Code)
	assert.Contains(t, dbErr.Message, "权限错误")
	assert.Contains(t, dbErr.Message, "user")
	assert.Contains(t, dbErr.Hint, "请联系数据库管理员")
}

func TestUndefinedTable(t *testing.T) {
	pgErr := createMockPgError(
		UndefinedTable,
		"relation \"unknown_table\" does not exist",
		"",
		"",
		"unknown_table",
		"",
		"",
	)

	err := WrapDBError(pgErr)
	dbErr, ok := err.(*DBError)
	assert.True(t, ok)

	assert.Equal(t, UndefinedTable, dbErr.Code)
	assert.Contains(t, dbErr.Message, "表不存在错误")
	assert.Contains(t, dbErr.Message, "unknown_table")
}

func TestConnectionError(t *testing.T) {
	pgErr := createMockPgError(
		ConnectionException,
		"connection refused",
		"",
		"",
		"",
		"",
		"",
	)

	err := WrapDBError(pgErr)
	dbErr, ok := err.(*DBError)
	assert.True(t, ok)

	assert.Equal(t, ConnectionException, dbErr.Code)
	assert.Contains(t, dbErr.Message, "数据库连接错误")
	assert.Contains(t, dbErr.Hint, "请检查数据库连接配置")
}

func TestDataError(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		message  string
		expected string
	}{
		{
			name:     "数值范围错误",
			code:     NumericValueOutOfRange,
			message:  "numeric field overflow",
			expected: "请检查数值是否在允许的范围内",
		},
		{
			name:     "日期格式错误",
			code:     InvalidDatetimeFormat,
			message:  "invalid date format",
			expected: "请检查日期时间格式是否正确",
		},
		{
			name:     "除零错误",
			code:     DivisionByZero,
			message:  "division by zero",
			expected: "计算过程中出现除以零的操作",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgErr := createMockPgError(
				tc.code,
				tc.message,
				"",
				"",
				"",
				"",
				"",
			)

			err := WrapDBError(pgErr)
			dbErr, ok := err.(*DBError)
			assert.True(t, ok)

			assert.Equal(t, tc.code, dbErr.Code)
			assert.Contains(t, dbErr.Message, "数据错误")
			assert.Equal(t, tc.expected, dbErr.Hint)
		})
	}
}

func TestErrorCategory(t *testing.T) {
	testCases := []struct {
		code     string
		expected ErrorCategory
	}{
		{ForeignKeyViolation, CategoryIntegrityConstraint},
		{InsufficientPrivilege, CategoryPermission},
		{ConnectionException, CategoryConnection},
		{DataException, CategoryData},
		{InvalidTransactionState, CategoryTransaction},
		{SystemError, CategorySystem},
		{"99999", CategoryUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			category := GetCategory(tc.code)
			assert.Equal(t, tc.expected, category)
		})
	}
}

func TestErrorInterfaces(t *testing.T) {
	originalErr := errors.New("原始错误")
	dbErr := &DBError{
		Code:     "TEST001",
		Message:  "测试错误",
		Category: CategoryUnknown,
		Raw:      originalErr,
	}

	t.Run("Error接口", func(t *testing.T) {
		assert.Equal(t, "测试错误 | 错误码: TEST001", dbErr.Error())
	})

	t.Run("Unwrap接口", func(t *testing.T) {
		assert.Equal(t, originalErr, errors.Unwrap(dbErr))
	})

	t.Run("Is接口", func(t *testing.T) {
		target := &DBError{Code: "TEST001"}
		assert.True(t, errors.Is(dbErr, target))

		nonMatch := &DBError{Code: "TEST002"}
		assert.False(t, errors.Is(dbErr, nonMatch))
	})
}

func TestExtractFunctions(t *testing.T) {
	t.Run("提取表名", func(t *testing.T) {
		assert.Equal(t, "users", extractTableName("public.users"))
		assert.Equal(t, "orders", extractTableName("orders"))
	})

	t.Run("提取列名", func(t *testing.T) {
		detail := "Key (email)=(test@example.com)"
		assert.Equal(t, "email", extractColumnName(detail))
	})

	t.Run("提取被引用表名", func(t *testing.T) {
		detail := `Key (user_id)=(1) is not present in referenced table "users"`
		assert.Equal(t, "users", extractReferencedTable(detail))
	})

	t.Run("提取外键值", func(t *testing.T) {
		detail := "Key (user_id)=(123)"
		assert.Equal(t, "user_id=123", extractForeignKeyValues(detail))
	})

	t.Run("提取唯一值", func(t *testing.T) {
		detail := "Key (email)=(test@example.com)"
		assert.Equal(t, "email=test@example.com", extractUniqueValue(detail))
	})

	t.Run("提取操作类型", func(t *testing.T) {
		assert.Equal(t, "查询", extractOperation("SELECT from users"))
		assert.Equal(t, "插入", extractOperation("INSERT INTO users"))
		assert.Equal(t, "更新", extractOperation("UPDATE users"))
		assert.Equal(t, "删除", extractOperation("DELETE FROM users"))
	})
}
