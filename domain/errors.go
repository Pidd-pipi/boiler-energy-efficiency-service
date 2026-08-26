package domain

import "fmt"

// ErrKind 描述领域错误的种类，用于映射 HTTP 状态码。
type ErrKind string

const (
	// KindNotFound 资源不存在（404）。
	KindNotFound ErrKind = "not_found"
	// KindInvalidInput 输入参数非法（400）。
	KindInvalidInput ErrKind = "invalid_input"
	// KindStateTransition 状态迁移被状态机拒绝（409）。
	KindStateTransition ErrKind = "state_transition"
	// KindCalculationRejected 能效计算输入缺失被拒绝（422）。
	KindCalculationRejected ErrKind = "calculation_rejected"
	// KindConflict 业务冲突（409）。
	KindConflict ErrKind = "conflict"
	// KindInternal 内部错误（500）。
	KindInternal ErrKind = "internal"
)

// Error 是贯穿 domain -> service -> httpapi 的统一领域错误。
type Error struct {
	Kind    ErrKind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 支持 errors.Is/As 链路。
func (e *Error) Unwrap() error { return e.Cause }

// NewError 构造领域错误。
func NewError(kind ErrKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

// WrapError 用领域错误包装底层错误。
func WrapError(kind ErrKind, cause error, format string, args ...any) *Error {
	return &Error{
		Kind:    kind,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// IsKind 判断错误是否属于指定领域错误种类。
func IsKind(err error, kind ErrKind) bool {
	for err != nil {
		if de, ok := err.(*Error); ok && de.Kind == kind {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
