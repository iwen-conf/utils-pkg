package errors

// 此文件展示如何使用通用错误包的示例代码

import "fmt"

// ExampleUsage 展示错误包的基本用法
func ExampleUsage() {
	// 1. 注册项目特定的错误码
	RegisterErrorCodes(map[string]string{
		"USER001": "用户不存在",
		"USER002": "用户已被禁用",
		"AUTH001": "认证失败",
		"AUTH002": "权限不足",
		"DB001":   "数据库连接失败",
		"API001":  "外部API调用失败",
	})

	// 2. 创建基础错误
	userErr := New("USER001", "用户不存在")
	fmt.Printf("基础错误: %s\n", userErr.Error())

	// 3. 创建带详细信息的错误
	detailErr := NewWithDetails("USER002", "用户已被禁用", "用户违反社区规定")
	fmt.Printf("详细错误: %s\n", detailErr.Error())

	// 4. 添加上下文信息
	contextErr := userErr.WithContext("user_id", "12345").WithContext("action", "login")
	fmt.Printf("上下文错误: %s, 上下文: %+v\n", contextErr.Error(), contextErr.Context)

	// 5. 包装系统错误
	systemErr := fmt.Errorf("connection timeout")
	wrappedErr := Wrap(systemErr, "DB001", "数据库连接失败")
	fmt.Printf("包装错误: %s\n", wrappedErr.Error())

	// 6. 使用错误构建器
	builderErr := NewBuilder().
		Code("API001").
		Message("外部API调用失败").
		Details("第三方服务暂不可用").
		Context("endpoint", "/api/v1/users").
		Context("status_code", 503).
		Build()
	fmt.Printf("构建器错误: %s\n", builderErr.Error())

	// 7. 从错误码创建错误
	codeErr := FromCode("AUTH001")
	fmt.Printf("从错误码创建: %s\n", codeErr.Error())

	// 8. 错误分类判断
	if IsClientError("4001") {
		fmt.Println("这是客户端错误")
	}
	if IsSystemError("5001") {
		fmt.Println("这是系统错误")
	}
	if IsRetryableErrorCode("5000") {
		fmt.Println("这个错误可以重试")
	}
}

// ExampleErrorHandler 展示错误处理器的用法
func ExampleErrorHandler() {
	// 创建错误处理器链
	chain := NewHandlerChain()
	
	// 添加日志处理器
	chain.Add(func(err *Error) error {
		fmt.Printf("记录错误日志: [%s] %s\n", err.Code, err.Message)
		return nil
	})
	
	// 添加监控处理器
	chain.Add(func(err *Error) error {
		if IsSystemError(err.Code) {
			fmt.Printf("发送系统错误告警: %s\n", err.Code)
		}
		return nil
	})
	
	// 处理错误
	err := New("5001", "服务不可用")
	chain.Handle(err)
}

// ExampleErrorAggregator 展示错误聚合器的用法
func ExampleErrorAggregator() {
	aggregator := NewAggregator()
	
	// 模拟批量操作中的多个错误
	aggregator.Add(New("USER001", "用户不存在"))
	aggregator.Add(New("USER002", "用户已被禁用"))
	aggregator.Add(fmt.Errorf("网络连接失败"))
	
	if aggregator.HasErrors() {
		fmt.Printf("批量操作失败: %s\n", aggregator.Error())
		for i, err := range aggregator.Errors() {
			fmt.Printf("错误 %d: %s\n", i+1, err.Error())
		}
	}
}