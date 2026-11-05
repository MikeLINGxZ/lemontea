package onboarding_dto

// IsInitializedInput 是初始化状态查询的空请求
type IsInitializedInput struct{}

// IsInitializedOutput 返回应用是否已完成首次启动引导
type IsInitializedOutput struct {
	// 已初始化
	Initialized bool `json:"initialized"`
}
