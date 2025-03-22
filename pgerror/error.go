package pgerror

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
)

// 定义数据库错误码
const (
	// 完整性约束错误 (23xxx)
	ForeignKeyViolation = "23503" // 外键约束错误
	UniqueViolation    = "23505" // 唯一约束错误
	CheckViolation     = "23514" // 检查约束错误
	NotNullViolation   = "23502" // 非空约束错误
	
	// 权限错误 (42xxx)
	InsufficientPrivilege = "42501" // 权限不足
	UndefinedTable       = "42P01" // 表不存在
	UndefinedColumn      = "42703" // 列不存在
	DuplicateTable       = "42P07" // 表已存在
	
	// 连接错误 (08xxx)
	ConnectionException      = "08000" // 连接异常
	ConnectionDoesNotExist  = "08003" // 连接不存在
	ConnectionFailure       = "08006" // 连接失败
	
	// 数据异常 (22xxx)
	DataException           = "22000" // 数据异常
	NumericValueOutOfRange = "22003" // 数值超出范围
	InvalidDatetimeFormat  = "22007" // 日期时间格式无效
	DivisionByZero        = "22012" // 除以零
	
	// 事务错误 (25xxx)
	InvalidTransactionState = "25000" // 无效的事务状态
	ActiveTransactionState = "25001" // 活动事务状态
	BranchTransactionState = "25002" // 分支事务状态
	
	// 系统错误 (53xxx, 54xxx, 58xxx)
	InsufficientResources  = "53000" // 资源不足
	ProgramLimitExceeded  = "54000" // 程序限制超出
	SystemError           = "58000" // 系统错误
)

// ErrorCategory 错误类别
type ErrorCategory string

const (
	CategoryIntegrityConstraint ErrorCategory = "完整性约束错误"
	CategoryPermission         ErrorCategory = "权限错误"
	CategoryConnection        ErrorCategory = "连接错误"
	CategoryData             ErrorCategory = "数据错误"
	CategoryTransaction      ErrorCategory = "事务错误"
	CategorySystem          ErrorCategory = "系统错误"
	CategoryUnknown         ErrorCategory = "未知错误"
)

// DBError 数据库错误结构体
type DBError struct {
	Code     string        // 错误码
	Message  string        // 错误消息
	Detail   string        // 详细信息
	Hint     string        // 提示信息
	Category ErrorCategory // 错误类别
	Schema   string        // 模式名
	Table    string        // 表名
	Column   string        // 列名
	Raw      error        // 原始错误
}

// Error 实现error接口
func (e *DBError) Error() string {
	return e.Message
}

// Unwrap 实现errors.Unwrap接口
func (e *DBError) Unwrap() error {
	return e.Raw
}

// Is 实现errors.Is接口
func (e *DBError) Is(target error) bool {
	t, ok := target.(*DBError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// GetCategory 根据错误码获取错误类别
func GetCategory(code string) ErrorCategory {
	switch {
	case strings.HasPrefix(code, "23"):
		return CategoryIntegrityConstraint
	case strings.HasPrefix(code, "42"):
		return CategoryPermission
	case strings.HasPrefix(code, "08"):
		return CategoryConnection
	case strings.HasPrefix(code, "22"):
		return CategoryData
	case strings.HasPrefix(code, "25"):
		return CategoryTransaction
	case strings.HasPrefix(code, "53"), strings.HasPrefix(code, "54"), strings.HasPrefix(code, "58"):
		return CategorySystem
	default:
		return CategoryUnknown
	}
}

// WrapDBError 包装数据库错误
func WrapDBError(err error) error {
	if err == nil {
		return nil
	}

	// 尝试将错误转换为PgError
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		dbErr := &DBError{
			Code:     pgErr.Code,
			Detail:   pgErr.Detail,
			Hint:     pgErr.Hint,
			Schema:   pgErr.SchemaName,
			Table:    pgErr.TableName,
			Column:   pgErr.ColumnName,
			Category: GetCategory(pgErr.Code),
			Raw:      err,
		}

		// 根据错误码处理不同类型的错误
		switch pgErr.Code {
		case ForeignKeyViolation:
			return handleForeignKeyViolation(dbErr, pgErr)
		case UniqueViolation:
			return handleUniqueViolation(dbErr, pgErr)
		case CheckViolation:
			return handleCheckViolation(dbErr, pgErr)
		case NotNullViolation:
			return handleNotNullViolation(dbErr, pgErr)
		case InsufficientPrivilege:
			return handleInsufficientPrivilege(dbErr, pgErr)
		case UndefinedTable:
			return handleUndefinedTable(dbErr, pgErr)
		case UndefinedColumn:
			return handleUndefinedColumn(dbErr, pgErr)
		case ConnectionException, ConnectionDoesNotExist, ConnectionFailure:
			return handleConnectionError(dbErr, pgErr)
		case DataException, NumericValueOutOfRange, InvalidDatetimeFormat, DivisionByZero:
			return handleDataError(dbErr, pgErr)
		case InvalidTransactionState, ActiveTransactionState, BranchTransactionState:
			return handleTransactionError(dbErr, pgErr)
		case InsufficientResources, ProgramLimitExceeded, SystemError:
			return handleSystemError(dbErr, pgErr)
		default:
			dbErr.Message = fmt.Sprintf("数据库错误：%s", pgErr.Message)
			return dbErr
		}
	}

	// 如果不是pg错误，返回通用错误
	return &DBError{
		Code:     "UNKNOWN",
		Message:  err.Error(),
		Category: CategoryUnknown,
		Raw:      err,
	}
}

// 处理外键约束错误
func handleForeignKeyViolation(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.TableName)
	referencedTable := extractReferencedTable(pgErr.Detail)
	constraintName := pgErr.ConstraintName

	dbErr.Message = fmt.Sprintf(
		"数据关联错误：无法在%s中创建或更新记录，因为在%s中找不到关联的记录（约束：%s）",
		tableName,
		referencedTable,
		constraintName,
	)
	
	if hint := extractForeignKeyValues(pgErr.Detail); hint != "" {
		dbErr.Hint = fmt.Sprintf("请检查关联数据是否存在，关联值：%s", hint)
	}

	return dbErr
}

// 处理唯一约束错误
func handleUniqueViolation(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.TableName)
	columnName := extractColumnName(pgErr.Detail)
	value := extractUniqueValue(pgErr.Detail)

	dbErr.Message = fmt.Sprintf(
		"数据重复错误：在%s中已存在相同的%s记录",
		tableName,
		columnName,
	)

	if value != "" {
		dbErr.Hint = fmt.Sprintf("重复的值：%s", value)
	}

	return dbErr
}

// 处理检查约束错误
func handleCheckViolation(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.TableName)
	constraintName := pgErr.ConstraintName
	condition := extractCheckCondition(pgErr.Detail)

	dbErr.Message = fmt.Sprintf(
		"数据验证错误：%s中的数据不满足%s约束条件",
		tableName,
		constraintName,
	)

	if condition != "" {
		dbErr.Hint = fmt.Sprintf("验证条件：%s", condition)
	}

	return dbErr
}

// 处理非空约束错误
func handleNotNullViolation(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.TableName)
	columnName := pgErr.ColumnName

	dbErr.Message = fmt.Sprintf(
		"数据完整性错误：%s的%s字段不能为空",
		tableName,
		columnName,
	)

	dbErr.Hint = "请提供必要的数据值"
	return dbErr
}

// 处理权限不足错误
func handleInsufficientPrivilege(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	operation := extractOperation(pgErr.Message)
	object := extractObject(pgErr.Message)

	dbErr.Message = fmt.Sprintf(
		"权限错误：当前用户没有权限执行%s操作（对象：%s）",
		operation,
		object,
	)

	dbErr.Hint = "请联系数据库管理员获取必要权限"
	return dbErr
}

// 处理表不存在错误
func handleUndefinedTable(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.Message)

	dbErr.Message = fmt.Sprintf(
		"表不存在错误：数据表%s不存在",
		tableName,
	)

	dbErr.Hint = "请检查表名是否正确，或者确认表是否已经创建"
	return dbErr
}

// 处理列不存在错误
func handleUndefinedColumn(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	columnName := extractColumnFromMessage(pgErr.Message)
	tableName := extractTableFromMessage(pgErr.Message)

	dbErr.Message = fmt.Sprintf(
		"列不存在错误：数据表%s中不存在列%s",
		tableName,
		columnName,
	)

	dbErr.Hint = "请检查列名是否正确，或者确认列是否已经添加到表中"
	return dbErr
}

// 处理连接错误
func handleConnectionError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"数据库连接错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "请检查数据库连接配置和网络状态"
	return dbErr
}

// 处理数据错误
func handleDataError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"数据错误：%s",
		pgErr.Message,
	)

	switch pgErr.Code {
	case NumericValueOutOfRange:
		dbErr.Hint = "请检查数值是否在允许的范围内"
	case InvalidDatetimeFormat:
		dbErr.Hint = "请检查日期时间格式是否正确"
	case DivisionByZero:
		dbErr.Hint = "计算过程中出现除以零的操作"
	default:
		dbErr.Hint = "请检查数据格式是否正确"
	}

	return dbErr
}

// 处理事务错误
func handleTransactionError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"事务错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "请检查事务状态和操作顺序"
	return dbErr
}

// 处理系统错误
func handleSystemError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"系统错误：%s",
		pgErr.Message,
	)

	switch pgErr.Code {
	case InsufficientResources:
		dbErr.Hint = "系统资源不足，请稍后重试或联系管理员"
	case ProgramLimitExceeded:
		dbErr.Hint = "超出程序限制，请检查配置或联系管理员"
	default:
		dbErr.Hint = "系统发生错误，请联系管理员"
	}

	return dbErr
}

// 辅助函数

// 从错误信息中提取表名
func extractTableName(tableName string) string {
	if idx := strings.LastIndex(tableName, "."); idx != -1 {
		return tableName[idx+1:]
	}
	return tableName
}

// 从错误信息中提取列名
func extractColumnName(detail string) string {
	if idx := strings.Index(detail, "Key ("); idx != -1 {
		if endIdx := strings.Index(detail[idx:], ")"); endIdx != -1 {
			return detail[idx+5 : idx+endIdx]
		}
	}
	return "未知字段"
}

// 从错误信息中提取被引用的表名
func extractReferencedTable(detail string) string {
	if idx := strings.Index(detail, "referenced table \""); idx != -1 {
		if endIdx := strings.Index(detail[idx+18:], "\""); endIdx != -1 {
			return detail[idx+18 : idx+18+endIdx]
		}
	}
	return "未知表"
}

// 从错误信息中提取外键值
func extractForeignKeyValues(detail string) string {
	if idx := strings.Index(detail, "Key ("); idx != -1 {
		if endIdx := strings.Index(detail[idx:], ")"); endIdx != -1 {
			return detail[idx+5 : idx+endIdx]
		}
	}
	return ""
}

// 从错误信息中提取唯一值
func extractUniqueValue(detail string) string {
	if idx := strings.Index(detail, "="); idx != -1 {
		return strings.TrimSpace(detail[idx+1:])
	}
	return ""
}

// 从错误信息中提取检查条件
func extractCheckCondition(detail string) string {
	if idx := strings.Index(detail, "check constraint \""); idx != -1 {
		if endIdx := strings.Index(detail[idx+18:], "\""); endIdx != -1 {
			return detail[idx+18 : idx+18+endIdx]
		}
	}
	return ""
}

// 从错误信息中提取操作类型
func extractOperation(message string) string {
	operations := map[string]string{
		"SELECT": "查询",
		"INSERT": "插入",
		"UPDATE": "更新",
		"DELETE": "删除",
		"CREATE": "创建",
		"ALTER":  "修改",
		"DROP":   "删除",
	}

	for eng, chn := range operations {
		if strings.Contains(message, eng) {
			return chn
		}
	}
	return "未知操作"
}

// 从错误信息中提取对象
func extractObject(message string) string {
	if idx := strings.Index(message, "on"); idx != -1 {
		return strings.TrimSpace(message[idx+2:])
	}
	return "未知对象"
}

// 从错误信息中提取列名
func extractColumnFromMessage(message string) string {
	if idx := strings.Index(message, "column \""); idx != -1 {
		if endIdx := strings.Index(message[idx+8:], "\""); endIdx != -1 {
			return message[idx+8 : idx+8+endIdx]
		}
	}
	return "未知列"
}

// 从错误信息中提取表名
func extractTableFromMessage(message string) string {
	if idx := strings.Index(message, "table \""); idx != -1 {
		if endIdx := strings.Index(message[idx+7:], "\""); endIdx != -1 {
			return message[idx+7 : idx+7+endIdx]
		}
	}
	return "未知表"
} 