package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/middleware"
)

// envelope 统一响应格式：{code, message, data, trace_id}。
type envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// writeJSON 输出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("响应编码失败", "err", err)
	}
}

// respondOK 成功响应（200，code=0）。
func respondOK(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, http.StatusOK, envelope{
		Code:    0,
		Message: "ok",
		Data:    data,
		TraceID: middleware.TraceID(r.Context()),
	})
}

// respondCreated 创建成功（201）。
func respondCreated(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, http.StatusCreated, envelope{
		Code:    0,
		Message: "created",
		Data:    data,
		TraceID: middleware.TraceID(r.Context()),
	})
}

// respondMessage 仅返回消息的成功响应。
func respondMessage(w http.ResponseWriter, r *http.Request, message string, data any) {
	writeJSON(w, http.StatusOK, envelope{
		Code:    0,
		Message: message,
		Data:    data,
		TraceID: middleware.TraceID(r.Context()),
	})
}

// fail 输出指定状态码的统一错误。
func fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, status, envelope{
		Code:    status,
		Message: message,
		TraceID: middleware.TraceID(r.Context()),
	})
}

// statusOfDomainError 把领域错误映射为 HTTP 状态码。
func statusOfDomainError(err error) int {
	switch {
	case domain.IsKind(err, domain.KindNotFound):
		return http.StatusNotFound
	case domain.IsKind(err, domain.KindInvalidInput):
		return http.StatusBadRequest
	case domain.IsKind(err, domain.KindStateTransition), domain.IsKind(err, domain.KindConflict):
		return http.StatusConflict
	case domain.IsKind(err, domain.KindCalculationRejected):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// handle 包装 handler 统一错误处理：handler 返回 error 时映射状态码并输出。
type apiHandler func(w http.ResponseWriter, r *http.Request) error

func handle(fn apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			status := statusOfDomainError(err)
			var de *domain.Error
			if errors.As(err, &de) {
				fail(w, r, status, de.Message)
				return
			}
			slog.Error("未包装错误", "method", r.Method, "path", r.URL.Path, "err", err)
			fail(w, r, status, "服务器内部错误")
		}
	}
}

// decodeJSON 解析请求体并校验：仅允许单个 JSON 对象，禁止未知字段。
func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return domain.NewError(domain.KindInvalidInput, "请求体不能为空")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.NewError(domain.KindInvalidInput, "请求体解析失败：%v", err)
	}
	// 确保请求体之后没有第二个 JSON 值。
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return domain.NewError(domain.KindInvalidInput, "请求体只能包含一个 JSON 对象")
	}
	return nil
}
