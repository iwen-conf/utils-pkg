package errors

// 通用错误码
const (
	// 系统错误
	CodeInternal      = "INTERNAL_ERROR"
	CodeTimeout       = "TIMEOUT_ERROR"
	CodeUnavailable   = "SERVICE_UNAVAILABLE"
	CodeNotFound      = "NOT_FOUND"
	CodeAlreadyExists = "ALREADY_EXISTS"
	
	// 认证和授权错误
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeInvalidToken = "INVALID_TOKEN"
	CodeExpiredToken = "EXPIRED_TOKEN"
	
	// 验证错误
	CodeInvalidInput  = "INVALID_INPUT"
	CodeMissingField  = "MISSING_FIELD"
	CodeInvalidFormat = "INVALID_FORMAT"
	CodeOutOfRange    = "OUT_OF_RANGE"
	CodeInvalidLength = "INVALID_LENGTH"
	
	// 网络和外部服务错误
	CodeNetworkError    = "NETWORK_ERROR"
	CodeConnectionError = "CONNECTION_ERROR"
	CodeExternalService = "EXTERNAL_SERVICE_ERROR"
	
	// 数据库错误
	CodeDatabaseError    = "DATABASE_ERROR"
	CodeQueryError       = "QUERY_ERROR"
	CodeTransactionError = "TRANSACTION_ERROR"
	
	// 业务逻辑错误
	CodeBusinessRule      = "BUSINESS_RULE_VIOLATION"
	CodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	CodeQuotaExceeded     = "QUOTA_EXCEEDED"
)

// 错误严重级别
type Severity string

const (
	SeverityLow      Severity = "低"
	SeverityMedium   Severity = "中"
	SeverityHigh     Severity = "高"
	SeverityCritical Severity = "严重"
)

// 错误类别，用于对相关错误进行分组
type Category string

const (
	CategorySystem     Category = "系统"
	CategoryAuth       Category = "认证"
	CategoryValidation Category = "验证"
	CategoryNetwork    Category = "网络"
	CategoryDatabase   Category = "数据库"
	CategoryBusiness   Category = "业务"
	CategoryExternal   Category = "外部"
)

// ErrorType 表示带有默认值的预定义错误类型
type ErrorType struct {
	Code     string
	Message  string
	Severity Severity
	Category Category
}

// 预定义的错误类型
var (
	// 系统错误类型
	InternalError = ErrorType{
		Code:     CodeInternal,
		Message:  "发生内部服务器错误",
		Severity: SeverityCritical,
		Category: CategorySystem,
	}
	
	TimeoutError = ErrorType{
		Code:     CodeTimeout,
		Message:  "操作超时",
		Severity: SeverityHigh,
		Category: CategorySystem,
	}
	
	NotFoundError = ErrorType{
		Code:     CodeNotFound,
		Message:  "资源未找到",
		Severity: SeverityMedium,
		Category: CategorySystem,
	}
	
	// 认证错误类型
	UnauthorizedError = ErrorType{
		Code:     CodeUnauthorized,
		Message:  "需要认证",
		Severity: SeverityHigh,
		Category: CategoryAuth,
	}
	
	ForbiddenError = ErrorType{
		Code:     CodeForbidden,
		Message:  "访问被拒绝",
		Severity: SeverityHigh,
		Category: CategoryAuth,
	}
	
	// 验证错误类型
	InvalidInputError = ErrorType{
		Code:     CodeInvalidInput,
		Message:  "提供了无效的输入",
		Severity: SeverityMedium,
		Category: CategoryValidation,
	}
	
	MissingFieldError = ErrorType{
		Code:     CodeMissingField,
		Message:  "缺少必填字段",
		Severity: SeverityMedium,
		Category: CategoryValidation,
	}
	
	// 网络错误类型
	NetworkError = ErrorType{
		Code:     CodeNetworkError,
		Message:  "网络通信失败",
		Severity: SeverityHigh,
		Category: CategoryNetwork,
	}
	
	// 数据库错误类型
	DatabaseError = ErrorType{
		Code:     CodeDatabaseError,
		Message:  "数据库操作失败",
		Severity: SeverityHigh,
		Category: CategoryDatabase,
	}
)