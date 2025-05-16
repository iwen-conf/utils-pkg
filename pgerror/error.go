package pgerror

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// 定义数据库错误码
const (
	// 完整性约束错误 (23xxx)
	ForeignKeyViolation = "23503" // 外键约束错误
	UniqueViolation     = "23505" // 唯一约束错误
	CheckViolation      = "23514" // 检查约束错误
	NotNullViolation    = "23502" // 非空约束错误
	ExclusionViolation  = "23P01" // 排除约束错误
	RestrictViolation   = "23001" // 限制违规错误

	// 权限错误 (42xxx)
	InsufficientPrivilege = "42501" // 权限不足
	UndefinedTable        = "42P01" // 表不存在
	UndefinedColumn       = "42703" // 列不存在
	DuplicateTable        = "42P07" // 表已存在
	DuplicateColumn       = "42701" // 列已存在
	UndefinedFunction     = "42883" // 函数不存在
	UndefinedObject       = "42704" // 对象不存在
	SyntaxError           = "42601" // 语法错误

	// 连接错误 (08xxx)
	ConnectionException                  = "08000" // 连接异常
	ConnectionDoesNotExist               = "08003" // 连接不存在
	ConnectionFailure                    = "08006" // 连接失败
	SQLClientUnableToEstablishConnection = "08001" // 客户端无法建立连接
	ConnectionRejection                  = "08004" // 连接被拒绝

	// 数据异常 (22xxx)
	DataException             = "22000" // 数据异常
	NumericValueOutOfRange    = "22003" // 数值超出范围
	InvalidDatetimeFormat     = "22007" // 日期时间格式无效
	DivisionByZero            = "22012" // 除以零
	IntervalFieldOverflow     = "22015" // 间隔字段溢出
	InvalidParameterValue     = "22023" // 无效的参数值
	CharacterNotInRepertoire  = "22021" // 字符不在字符集中
	StringDataRightTruncation = "22001" // 字符串数据右截断

	// 事务错误 (25xxx)
	InvalidTransactionState  = "25000" // 无效的事务状态
	ActiveTransactionState   = "25001" // 活动事务状态
	BranchTransactionState   = "25002" // 分支事务状态
	NoActiveTransaction      = "25P01" // 没有活动事务
	InFailedTransactionState = "25P02" // 在失败的事务状态中

	// 系统错误 (53xxx, 54xxx, 58xxx)
	InsufficientResources      = "53000" // 资源不足
	DiskFull                   = "53100" // 磁盘已满
	OutOfMemory                = "53200" // 内存不足
	TooManyConnections         = "53300" // 连接过多
	ConfigurationLimitExceeded = "53400" // 配置限制超出
	ProgramLimitExceeded       = "54000" // 程序限制超出
	StatementTooComplex        = "54001" // 语句过于复杂
	TooManyColumns             = "54011" // 列过多
	TooManyArguments           = "54023" // 参数过多
	SystemError                = "58000" // 系统错误
	IOError                    = "58030" // 输入输出错误

	// 操作符干预错误 (57xxx)
	OperatorIntervention = "57000" // 操作员干预
	QueryCanceled        = "57014" // 查询取消
	AdminShutdown        = "57P01" // 管理员关闭
	CrashShutdown        = "57P02" // 崩溃关闭
	DatabaseDropped      = "57P03" // 数据库已删除

	// 恢复错误 (XX xxx)
	DeadlockDetected = "40P01" // 检测到死锁

	// 类错误 (P0xxx)
	PlPgSQLError   = "P0000" // PL/pgSQL错误
	RaiseException = "P0001" // 抛出异常
	NoDataFound    = "P0002" // 未找到数据
	TooManyRows    = "P0003" // 行过多
)

// ErrorCategory 错误类别
type ErrorCategory string

const (
	CategoryIntegrityConstraint ErrorCategory = "完整性约束错误"
	CategoryPermission          ErrorCategory = "权限错误"
	CategoryConnection          ErrorCategory = "连接错误"
	CategoryData                ErrorCategory = "数据错误"
	CategoryTransaction         ErrorCategory = "事务错误"
	CategorySystem              ErrorCategory = "系统错误"
	CategoryOperator            ErrorCategory = "操作错误"
	CategoryPlpgsql             ErrorCategory = "PL/pgSQL错误"
	CategoryRecovery            ErrorCategory = "恢复错误"
	CategorySyntax              ErrorCategory = "语法错误"
	CategoryUnknown             ErrorCategory = "未知错误"
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
	Position string        // 错误位置
	Query    string        // 导致错误的查询（如果可用）
	Where    string        // 错误上下文位置
	Raw      error         // 原始错误
	Time     time.Time     // 错误发生时间
}

// Error 实现error接口
func (e *DBError) Error() string {
	var parts []string

	// 基本错误信息
	parts = append(parts, e.Message)

	// 添加错误码
	if e.Code != "" && e.Code != "UNKNOWN" {
		parts = append(parts, fmt.Sprintf("错误码: %s", e.Code))
	}

	// 添加详情
	if e.Detail != "" {
		parts = append(parts, fmt.Sprintf("详情: %s", e.Detail))
	}

	// 添加提示
	if e.Hint != "" {
		parts = append(parts, fmt.Sprintf("提示: %s", e.Hint))
	}

	// 添加位置信息
	var location []string
	if e.Schema != "" {
		location = append(location, fmt.Sprintf("模式: %s", e.Schema))
	}
	if e.Table != "" {
		location = append(location, fmt.Sprintf("表: %s", e.Table))
	}
	if e.Column != "" {
		location = append(location, fmt.Sprintf("列: %s", e.Column))
	}
	if len(location) > 0 {
		parts = append(parts, fmt.Sprintf("位置: [%s]", strings.Join(location, ", ")))
	}

	return strings.Join(parts, " | ")
}

// FormatError 返回格式化的错误信息
func (e *DBError) FormatError(includeDetails bool) string {
	var result strings.Builder

	// 主要错误信息
	result.WriteString(e.Message)

	if includeDetails {
		result.WriteString("\n")

		// 分类和错误码
		result.WriteString(fmt.Sprintf("分类: %s (错误码: %s)\n", e.Category, e.Code))

		// 详情和提示
		if e.Detail != "" {
			result.WriteString(fmt.Sprintf("详情: %s\n", e.Detail))
		}
		if e.Hint != "" {
			result.WriteString(fmt.Sprintf("提示: %s\n", e.Hint))
		}

		// 位置信息
		if e.Schema != "" || e.Table != "" || e.Column != "" {
			result.WriteString("位置信息:\n")
			if e.Schema != "" {
				result.WriteString(fmt.Sprintf("  模式: %s\n", e.Schema))
			}
			if e.Table != "" {
				result.WriteString(fmt.Sprintf("  表: %s\n", e.Table))
			}
			if e.Column != "" {
				result.WriteString(fmt.Sprintf("  列: %s\n", e.Column))
			}
		}

		// 错误位置
		if e.Position != "" {
			result.WriteString(fmt.Sprintf("错误位置: %s\n", e.Position))
		}
		if e.Where != "" {
			result.WriteString(fmt.Sprintf("上下文: %s\n", e.Where))
		}

		// 查询信息
		if e.Query != "" {
			// 限制显示的查询长度，避免输出过长
			query := e.Query
			if len(query) > 200 {
				query = query[:200] + "..."
			}
			result.WriteString(fmt.Sprintf("SQL查询: %s\n", query))
		}

		// 时间
		if !e.Time.IsZero() {
			result.WriteString(fmt.Sprintf("发生时间: %s\n", e.Time.Format("2006-01-02 15:04:05")))
		}
	}

	return result.String()
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
	case strings.HasPrefix(code, "57"):
		return CategoryOperator
	case strings.HasPrefix(code, "P0"):
		return CategoryPlpgsql
	case strings.HasPrefix(code, "40"):
		return CategoryRecovery
	case code == SyntaxError:
		return CategorySyntax
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
			Position: fmt.Sprintf("%d", pgErr.Position),
			Where:    pgErr.Where,
			Category: GetCategory(pgErr.Code),
			Raw:      err,
			Time:     time.Now(),
		}

		// 尝试获取原始SQL查询（如果可用）
		if query := extractQueryFromError(err); query != "" {
			dbErr.Query = query
		}

		// 根据错误码处理不同类型的错误
		switch pgErr.Code {
		// 完整性约束错误
		case ForeignKeyViolation:
			return handleForeignKeyViolation(dbErr, pgErr)
		case UniqueViolation:
			return handleUniqueViolation(dbErr, pgErr)
		case CheckViolation:
			return handleCheckViolation(dbErr, pgErr)
		case NotNullViolation:
			return handleNotNullViolation(dbErr, pgErr)
		case ExclusionViolation:
			return handleExclusionViolation(dbErr, pgErr)
		case RestrictViolation:
			return handleRestrictViolation(dbErr, pgErr)

		// 权限错误
		case InsufficientPrivilege:
			return handleInsufficientPrivilege(dbErr, pgErr)
		case UndefinedTable:
			return handleUndefinedTable(dbErr, pgErr)
		case UndefinedColumn:
			return handleUndefinedColumn(dbErr, pgErr)
		case DuplicateTable:
			return handleDuplicateTable(dbErr, pgErr)
		case DuplicateColumn:
			return handleDuplicateColumn(dbErr, pgErr)
		case UndefinedFunction:
			return handleUndefinedFunction(dbErr, pgErr)
		case UndefinedObject:
			return handleUndefinedObject(dbErr, pgErr)
		case SyntaxError:
			return handleSyntaxError(dbErr, pgErr)

		// 连接错误
		case ConnectionException, ConnectionDoesNotExist, ConnectionFailure,
			SQLClientUnableToEstablishConnection, ConnectionRejection:
			return handleConnectionError(dbErr, pgErr)

		// 数据错误
		case DataException, NumericValueOutOfRange, InvalidDatetimeFormat,
			DivisionByZero, IntervalFieldOverflow, InvalidParameterValue,
			CharacterNotInRepertoire, StringDataRightTruncation:
			return handleDataError(dbErr, pgErr)

		// 事务错误
		case InvalidTransactionState, ActiveTransactionState, BranchTransactionState,
			NoActiveTransaction, InFailedTransactionState:
			return handleTransactionError(dbErr, pgErr)

		// 系统错误
		case InsufficientResources, DiskFull, OutOfMemory, TooManyConnections,
			ConfigurationLimitExceeded, ProgramLimitExceeded, StatementTooComplex,
			TooManyColumns, TooManyArguments, SystemError, IOError:
			return handleSystemError(dbErr, pgErr)

		// 操作干预错误
		case OperatorIntervention, QueryCanceled, AdminShutdown, CrashShutdown, DatabaseDropped:
			return handleOperatorInterventionError(dbErr, pgErr)

		// 恢复错误
		case DeadlockDetected:
			return handleDeadlockError(dbErr)

		// PL/pgSQL错误
		case PlPgSQLError, RaiseException, NoDataFound, TooManyRows:
			return handlePlPgSQLError(dbErr, pgErr)

		default:
			// 处理未明确定义的错误码
			// 尝试根据错误码前缀猜测错误类型
			switch {
			case strings.HasPrefix(pgErr.Code, "23"):
				return handleGenericIntegrityConstraintError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "42"):
				return handleGenericPermissionError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "08"):
				return handleGenericConnectionError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "22"):
				return handleGenericDataError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "25"):
				return handleGenericTransactionError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "53"), strings.HasPrefix(pgErr.Code, "54"),
				strings.HasPrefix(pgErr.Code, "58"):
				return handleGenericSystemError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "57"):
				return handleGenericOperatorError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "P0"):
				return handleGenericPlPgSQLError(dbErr, pgErr)
			case strings.HasPrefix(pgErr.Code, "40"):
				return handleGenericRecoveryError(dbErr, pgErr)
			default:
				// 最后的兜底
				dbErr.Message = fmt.Sprintf("数据库错误 [%s]：%s", pgErr.Code, pgErr.Message)
				return dbErr
			}
		}
	}

	// 如果不是pg错误，返回通用错误
	return &DBError{
		Code:     "UNKNOWN",
		Message:  err.Error(),
		Category: CategoryUnknown,
		Raw:      err,
		Time:     time.Now(),
	}
}

// WrapDBErrorWithQuery 包装数据库错误，并附加SQL查询信息
func WrapDBErrorWithQuery(err error, query string) error {
	if err == nil {
		return nil
	}

	dbErr := WrapDBError(err)
	if wrappedErr, ok := dbErr.(*DBError); ok {
		wrappedErr.Query = query
		// 如果存在语法错误并且有Position信息，为错误位置提供上下文
		if wrappedErr.Code == SyntaxError && wrappedErr.Position != "" {
			pos, parseErr := strconv.Atoi(wrappedErr.Position)
			if parseErr == nil && pos > 0 && pos < len(query) {
				// 提供错误位置前后的上下文
				start := pos - 20
				if start < 0 {
					start = 0
				}
				end := pos + 20
				if end > len(query) {
					end = len(query)
				}

				context := query[start:end]
				marker := strings.Repeat(" ", pos-start) + "^"
				wrappedErr.Hint = fmt.Sprintf("%s\n查询上下文: %s\n%s",
					wrappedErr.Hint, context, marker)
			}
		}
		return wrappedErr
	}

	return dbErr
}

// 从错误信息中提取SQL查询（如果可能）
func extractQueryFromError(err error) string {
	errStr := err.Error()

	// 尝试提取常见的查询前缀模式
	queryPrefixes := []string{
		"ERROR: syntax error in query: ",
		"query failed: ",
		"executing statement: ",
		"query: ",
	}

	for _, prefix := range queryPrefixes {
		if idx := strings.Index(errStr, prefix); idx != -1 {
			query := errStr[idx+len(prefix):]
			// 提取到下一个明显的分隔符
			for _, sep := range []string{"\n", ": ", ". "} {
				if sepIdx := strings.Index(query, sep); sepIdx != -1 {
					query = query[:sepIdx]
				}
			}
			return strings.TrimSpace(query)
		}
	}

	// 尝试匹配常见的SQL语句开始
	sqlPattern := regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|TRUNCATE|BEGIN|COMMIT|ROLLBACK|GRANT|REVOKE)\s+.+`)
	matches := sqlPattern.FindString(errStr)
	if matches != "" {
		return matches
	}

	return ""
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
	tableName := pgErr.TableName

	dbErr.Message = fmt.Sprintf(
		"权限错误：当前用户没有权限执行%s操作（对象：%s，表：%s）",
		operation,
		object,
		tableName,
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
	if tableName == "" {
		return "未知表"
	}

	// 处理schema.table格式
	if idx := strings.LastIndex(tableName, "."); idx != -1 {
		return tableName[idx+1:]
	}

	// 处理引号包裹的表名
	tableName = strings.Trim(tableName, "\"")

	return tableName
}

// 从错误信息中提取列名
func extractColumnName(detail string) string {
	if detail == "" {
		return "未知字段"
	}

	// 尝试匹配 "Key (column)"
	if idx := strings.Index(detail, "Key ("); idx != -1 {
		if endIdx := strings.Index(detail[idx:], ")"); endIdx != -1 {
			columnName := detail[idx+5 : idx+endIdx]
			// 处理多列情况，如 "Key (col1, col2)"
			if strings.Contains(columnName, ", ") {
				return columnName // 返回多列名
			}
			return strings.Trim(columnName, "\"") // 去除可能的引号
		}
	}

	// 尝试匹配 "column: xxx"
	re := regexp.MustCompile(`column[:\s]+["']?([^"',\s]+)["']?`)
	matches := re.FindStringSubmatch(detail)
	if len(matches) > 1 {
		return matches[1]
	}

	return "未知字段"
}

// 从错误信息中提取被引用的表名
func extractReferencedTable(detail string) string {
	if detail == "" {
		return "未知表"
	}

	// 尝试匹配 "referenced table "xxx"" 模式
	if idx := strings.Index(detail, "referenced table \""); idx != -1 {
		if endIdx := strings.Index(detail[idx+18:], "\""); endIdx != -1 {
			return detail[idx+18 : idx+18+endIdx]
		}
	}

	// 尝试匹配 "references xxx" 模式
	re := regexp.MustCompile(`references\s+([^\s"]+|"[^"]+")\s`)
	matches := re.FindStringSubmatch(detail)
	if len(matches) > 1 {
		return strings.Trim(matches[1], "\"")
	}

	return "未知表"
}

// 从错误信息中提取外键值
func extractForeignKeyValues(detail string) string {
	if detail == "" {
		return ""
	}

	// 提取 "Key (xxx)=(yyy)" 格式的内容
	keyValuePattern := regexp.MustCompile(`Key\s*\(([^)]+)\)=\(([^)]+)\)`)
	matches := keyValuePattern.FindStringSubmatch(detail)
	if len(matches) > 2 {
		// 返回 "列名=值" 的格式
		columns := strings.Split(matches[1], ", ")
		values := strings.Split(matches[2], ", ")

		var pairs []string
		for i := 0; i < len(columns) && i < len(values); i++ {
			pairs = append(pairs, fmt.Sprintf("%s=%s", columns[i], values[i]))
		}

		return strings.Join(pairs, ", ")
	}

	// 如果上面的模式没有匹配，尝试只提取键名
	if idx := strings.Index(detail, "Key ("); idx != -1 {
		if endIdx := strings.Index(detail[idx:], ")"); endIdx != -1 {
			return detail[idx+5 : idx+endIdx]
		}
	}

	return ""
}

// 从错误信息中提取唯一值
func extractUniqueValue(detail string) string {
	if detail == "" {
		return ""
	}

	// 尝试匹配 "Key (xxx)=(yyy)" 格式
	valuePattern := regexp.MustCompile(`Key\s*\(([^)]+)\)=\(([^)]+)\)`)
	matches := valuePattern.FindStringSubmatch(detail)
	if len(matches) > 2 {
		// 返回 "列名=值" 的格式
		columns := strings.Split(matches[1], ", ")
		values := strings.Split(matches[2], ", ")

		var pairs []string
		for i := 0; i < len(columns) && i < len(values); i++ {
			pairs = append(pairs, fmt.Sprintf("%s=%s", columns[i], values[i]))
		}

		return strings.Join(pairs, ", ")
	}

	// 简单模式，提取 "=" 后面的内容
	if idx := strings.Index(detail, "="); idx != -1 {
		value := strings.TrimSpace(detail[idx+1:])
		// 去除可能的括号
		value = strings.Trim(value, "()")
		return value
	}

	return ""
}

// 从错误信息中提取检查条件
func extractCheckCondition(detail string) string {
	if detail == "" {
		return ""
	}

	// 尝试匹配 "check constraint "xxx"" 模式
	if idx := strings.Index(detail, "check constraint \""); idx != -1 {
		if endIdx := strings.Index(detail[idx+18:], "\""); endIdx != -1 {
			return detail[idx+18 : idx+18+endIdx]
		}
	}

	// 尝试提取括号中的条件表达式
	re := regexp.MustCompile(`\(([^)]+)\)`)
	matches := re.FindStringSubmatch(detail)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// 从错误信息中提取操作类型
func extractOperation(message string) string {
	if message == "" {
		return "未知操作"
	}

	operations := map[string]string{
		"SELECT":   "查询",
		"INSERT":   "插入",
		"UPDATE":   "更新",
		"DELETE":   "删除",
		"CREATE":   "创建",
		"ALTER":    "修改",
		"DROP":     "删除",
		"TRUNCATE": "截断",
		"GRANT":    "授权",
		"REVOKE":   "撤销",
		"BEGIN":    "开始事务",
		"COMMIT":   "提交事务",
		"ROLLBACK": "回滚事务",
		"ANALYZE":  "分析",
		"VACUUM":   "清理",
		"EXPLAIN":  "执行计划",
		"COPY":     "复制",
	}

	// 先尝试匹配完整的操作词
	message = strings.ToUpper(message)
	for eng, chn := range operations {
		if strings.Contains(message, eng) {
			return chn
		}
	}

	return "未知操作"
}

// 从错误信息中提取对象
func extractObject(message string) string {
	if message == "" {
		return "未知对象"
	}

	// 尝试匹配 "on xxx" 模式
	if idx := strings.Index(message, " on "); idx != -1 {
		object := strings.TrimSpace(message[idx+4:])

		// 如果对象后面还有内容，尝试提取到下一个空格或结束
		if spaceIdx := strings.Index(object, " "); spaceIdx != -1 {
			object = object[:spaceIdx]
		}

		// 移除末尾的标点符号
		object = strings.TrimRight(object, ".,;:")
		return object
	}

	// 尝试从引号中提取对象名
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}

	return "未知对象"
}

// 从错误信息中提取列名
func extractColumnFromMessage(message string) string {
	if message == "" {
		return "未知列"
	}

	// 尝试匹配 "column "xxx"" 模式
	if idx := strings.Index(message, "column \""); idx != -1 {
		if endIdx := strings.Index(message[idx+8:], "\""); endIdx != -1 {
			return message[idx+8 : idx+8+endIdx]
		}
	}

	// 尝试匹配 "Column xxx" 模式（不带引号）
	columnPattern := regexp.MustCompile(`[Cc]olumn\s+([^\s",]+)`)
	matches := columnPattern.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}

	// 尝试从整个错误信息中提取被引号包围的名称
	quotedPattern := regexp.MustCompile(`"([^"]+)"`)
	matches = quotedPattern.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}

	return "未知列"
}

// 从错误信息中提取表名
func extractTableFromMessage(message string) string {
	if message == "" {
		return "未知表"
	}

	// 尝试匹配 "table "xxx"" 模式
	if idx := strings.Index(message, "table \""); idx != -1 {
		if endIdx := strings.Index(message[idx+7:], "\""); endIdx != -1 {
			return message[idx+7 : idx+7+endIdx]
		}
	}

	// 尝试匹配 "relation "xxx"" 模式
	if idx := strings.Index(message, "relation \""); idx != -1 {
		if endIdx := strings.Index(message[idx+10:], "\""); endIdx != -1 {
			return message[idx+10 : idx+10+endIdx]
		}
	}

	// 尝试匹配 "Table xxx" 模式（不带引号）
	tablePattern := regexp.MustCompile(`[Tt]able\s+([^\s",]+)`)
	matches := tablePattern.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}

	// 尝试提取第一个被引号包围的名称
	quotedPattern := regexp.MustCompile(`"([^"]+)"`)
	matches = quotedPattern.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}

	return "未知表"
}

// 处理排除约束错误
func handleExclusionViolation(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.TableName)
	constraintName := pgErr.ConstraintName
	details := extractConstraintDetails(pgErr.Detail)

	dbErr.Message = fmt.Sprintf(
		"排除约束错误：在%s中无法创建或更新记录，违反了排除约束%s",
		tableName,
		constraintName,
	)

	if details != "" {
		dbErr.Hint = fmt.Sprintf("冲突条件：%s", details)
	} else {
		dbErr.Hint = "请检查是否有冲突的记录存在"
	}

	return dbErr
}

// 处理限制违规错误
func handleRestrictViolation(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.TableName)
	constraintName := pgErr.ConstraintName

	dbErr.Message = fmt.Sprintf(
		"数据限制错误：在%s表中的操作违反了%s限制条件",
		tableName,
		constraintName,
	)

	dbErr.Hint = "请检查操作是否符合表的限制条件"
	return dbErr
}

// 处理表重复错误
func handleDuplicateTable(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableFromMessage(pgErr.Message)

	dbErr.Message = fmt.Sprintf(
		"表已存在错误：数据表%s已存在",
		tableName,
	)

	dbErr.Hint = "请使用不同的表名，或者先删除已存在的表"
	return dbErr
}

// 处理列重复错误
func handleDuplicateColumn(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	columnName := extractColumnFromMessage(pgErr.Message)
	tableName := extractTableFromMessage(pgErr.Message)

	dbErr.Message = fmt.Sprintf(
		"列已存在错误：数据表%s中的列%s已存在",
		tableName,
		columnName,
	)

	dbErr.Hint = "请使用不同的列名，或者检查表结构"
	return dbErr
}

// 处理函数不存在错误
func handleUndefinedFunction(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	functionName := extractFunctionName(pgErr.Message)

	dbErr.Message = fmt.Sprintf(
		"函数不存在错误：函数%s不存在或参数类型不匹配",
		functionName,
	)

	dbErr.Hint = "请检查函数名称和参数类型是否正确"
	return dbErr
}

// 处理对象不存在错误
func handleUndefinedObject(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	objectName := extractObjectName(pgErr.Message)
	objectType := extractObjectType(pgErr.Message)

	dbErr.Message = fmt.Sprintf(
		"对象不存在错误：%s %s不存在",
		objectType,
		objectName,
	)

	dbErr.Hint = "请检查对象名称是否正确，或者确认对象是否已经创建"
	return dbErr
}

// 处理SQL语法错误
func handleSyntaxError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	details := extractSyntaxErrorDetails(pgErr.Message)
	position := pgErr.Position

	dbErr.Message = fmt.Sprintf(
		"SQL语法错误：%s",
		details,
	)

	if position > 0 {
		dbErr.Hint = fmt.Sprintf("错误位置在字符%d附近", position)
	} else {
		dbErr.Hint = "请检查SQL语法是否正确"
	}

	return dbErr
}

// 处理操作干预错误
func handleOperatorInterventionError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"操作被中断：%s",
		pgErr.Message,
	)

	switch pgErr.Code {
	case QueryCanceled:
		dbErr.Hint = "查询已被用户或系统取消"
	case AdminShutdown:
		dbErr.Hint = "数据库正在进行管理员关闭操作"
	case CrashShutdown:
		dbErr.Hint = "数据库因崩溃而关闭"
	case DatabaseDropped:
		dbErr.Hint = "数据库已被删除"
	default:
		dbErr.Hint = "数据库操作被干预，请稍后重试"
	}

	return dbErr
}

// 处理死锁错误
func handleDeadlockError(dbErr *DBError) *DBError {
	dbErr.Message = "数据库死锁错误：检测到事务间的死锁"
	dbErr.Hint = "请稍后重试操作，或者检查应用程序的事务逻辑"
	return dbErr
}

// 处理PL/pgSQL错误
func handlePlPgSQLError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"PL/pgSQL错误：%s",
		pgErr.Message,
	)

	switch pgErr.Code {
	case RaiseException:
		dbErr.Hint = "存储过程中抛出异常"
	case NoDataFound:
		dbErr.Hint = "存储过程中未找到数据"
	case TooManyRows:
		dbErr.Hint = "存储过程中返回了多行数据，但预期只有一行"
	default:
		dbErr.Hint = "执行存储过程时发生错误"
	}

	return dbErr
}

// 处理通用完整性约束错误
func handleGenericIntegrityConstraintError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	tableName := extractTableName(pgErr.TableName)
	constraintName := pgErr.ConstraintName

	dbErr.Message = fmt.Sprintf(
		"数据完整性错误：在%s表中违反了约束%s",
		tableName,
		constraintName,
	)

	dbErr.Hint = "请检查数据是否满足所有约束条件"
	return dbErr
}

// 处理通用权限错误
func handleGenericPermissionError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"权限或命名错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "请检查对象名称是否正确，或者确认您是否有足够的权限"
	return dbErr
}

// 处理通用连接错误
func handleGenericConnectionError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"数据库连接错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "请检查数据库连接状态和配置"
	return dbErr
}

// 处理通用数据错误
func handleGenericDataError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"数据错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "请检查数据格式和值是否符合要求"
	return dbErr
}

// 处理通用事务错误
func handleGenericTransactionError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"事务状态错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "请检查事务状态和操作顺序"
	return dbErr
}

// 处理通用系统错误
func handleGenericSystemError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"系统资源错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "系统资源不足或超出限制，请联系管理员"
	return dbErr
}

// 处理通用操作干预错误
func handleGenericOperatorError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"操作中断：%s",
		pgErr.Message,
	)

	dbErr.Hint = "操作被中断，请稍后重试"
	return dbErr
}

// 处理通用PL/pgSQL错误
func handleGenericPlPgSQLError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"存储过程错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "执行存储过程时发生错误"
	return dbErr
}

// 处理通用恢复错误
func handleGenericRecoveryError(dbErr *DBError, pgErr *pgconn.PgError) *DBError {
	dbErr.Message = fmt.Sprintf(
		"事务恢复错误：%s",
		pgErr.Message,
	)

	dbErr.Hint = "事务处理过程中发生冲突，请重试操作"
	return dbErr
}

// 从错误信息中提取约束细节
func extractConstraintDetails(detail string) string {
	if detail == "" {
		return ""
	}

	// 提取括号中的内容
	re := regexp.MustCompile(`\(([^)]+)\)`)
	matches := re.FindStringSubmatch(detail)
	if len(matches) > 1 {
		return matches[1]
	}

	return detail
}

// 从错误信息中提取函数名
func extractFunctionName(message string) string {
	if idx := strings.Index(message, "function "); idx != -1 {
		// 提取引号间的函数名
		start := idx + 9 // "function "的长度
		if quoteIdx := strings.Index(message[start:], "\""); quoteIdx != -1 {
			start = start + quoteIdx + 1
			if endQuoteIdx := strings.Index(message[start:], "\""); endQuoteIdx != -1 {
				return message[start : start+endQuoteIdx]
			}
		}

		// 如果没有引号，尝试提取括号前的部分
		if parenIdx := strings.Index(message[start:], "("); parenIdx != -1 {
			return strings.TrimSpace(message[start : start+parenIdx])
		}
	}
	return "未知函数"
}

// 从错误信息中提取对象名称
func extractObjectName(message string) string {
	// 尝试匹配 "xxx "name" does not exist"
	re := regexp.MustCompile(`"([^"]+)"\s+does\s+not\s+exist`)
	matches := re.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}

	// 如果上面的模式没有匹配，尝试从引号中提取
	if idx := strings.Index(message, "\""); idx != -1 {
		if endIdx := strings.Index(message[idx+1:], "\""); endIdx != -1 {
			return message[idx+1 : idx+1+endIdx]
		}
	}

	return "未知对象"
}

// 从错误信息中提取对象类型
func extractObjectType(message string) string {
	objectTypes := map[string]string{
		"relation":   "关系(表)",
		"table":      "表",
		"column":     "列",
		"schema":     "模式",
		"type":       "类型",
		"function":   "函数",
		"operator":   "操作符",
		"role":       "角色",
		"database":   "数据库",
		"sequence":   "序列",
		"view":       "视图",
		"index":      "索引",
		"constraint": "约束",
		"trigger":    "触发器",
	}

	for eng, chn := range objectTypes {
		if strings.Contains(strings.ToLower(message), eng) {
			return chn
		}
	}

	return "对象"
}

// 从错误信息中提取语法错误详情
func extractSyntaxErrorDetails(message string) string {
	// 移除"syntax error at or near"部分，保留真正的错误内容
	if idx := strings.Index(message, "syntax error at or near "); idx != -1 {
		return strings.TrimSpace(message[idx+24:])
	}

	return message
}
