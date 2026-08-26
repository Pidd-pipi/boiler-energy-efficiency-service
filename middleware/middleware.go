// Package middleware 提供请求级横切关注点：
// requestID 注入、操作审计日志、统一错误/panic 处理。
package middleware

import "net/http"

// Middleware 中间件类型。
type Middleware func(http.Handler) http.Handler

// Chain 按顺序串联多个中间件，返回最终处理器。
// 先声明的中间件先执行（外层）。
func Chain(handler http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}

// statusWriter 包装 ResponseWriter 以记录状态码与字节数。
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Flush 支持 http.Flusher。
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
