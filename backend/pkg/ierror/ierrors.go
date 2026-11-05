package ierror

import (
	"encoding/json"
	"errors"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
)

// IError 是通过 Wails 绑定的服务方法统一返回的错误类型
// Detail 保存原始错误文本，Msg 保存面向用户的本地化消息
type IError struct {
	// 详情
	Detail string `json:"detail"`
	// 消息
	Msg string `json:"msg"`
}

// Error 返回 IError 的 JSON 表示
func (e *IError) Error() string {
	errBytes, err := json.Marshal(e) // 错误字节、将数据序列化时的错误
	if err != nil {
		return i18n.TCurrent("ierror.unknown_error", nil)
	}
	return string(errBytes)
}

// Is 允许 errors.Is 按 Msg 比较 IError
// 接收目标，返回是否满足条件
func (e *IError) Is(target error) bool {
	if /* 当前条目、是否成功取得对应值 */ t, ok := target.(*IError); ok {
		return e.Msg == t.Msg
	}
	return false
}

// Error 将 err 包装为 IError，并根据传入的错误码生成本地化的 Msg
// err 为 nil 时返回 nil，已有的 *IError 原样返回
func Error(code errorCode, err error) error {
	if err == nil {
		return nil
	}

	// 循环索引错误
	var iErr *IError
	if errors.As(err, &iErr) {
		return iErr
	}

	return &IError{
		Detail: err.Error(),
		Msg:    code.Msg(),
	}
}
