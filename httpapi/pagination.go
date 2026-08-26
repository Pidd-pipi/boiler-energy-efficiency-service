package httpapi

import (
	"net/http"
	"strconv"

	"example.com/boiler-energy-efficiency-service/domain"
)

// parseLimitOffset 解析 list 接口的 limit/offset 查询参数。
// limit 缺省使用 def，超过 max 时截断到 max；offset 缺省为 0。
// 非法值（非整数或负数）返回 400 错误。
func parseLimitOffset(r *http.Request, def, max int) (int, int, error) {
	q := r.URL.Query()
	limit := def
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return 0, 0, domain.NewError(domain.KindInvalidInput, "limit 必须为非负整数：%q", v)
		}
		limit = n
	}
	if limit > max {
		limit = max
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return 0, 0, domain.NewError(domain.KindInvalidInput, "offset 必须为非负整数：%q", v)
		}
		offset = n
	}
	return limit, offset, nil
}

// paginateTail 从时间正序的完整列表中取“最新窗口”。
// offset=0 时返回最新 limit 条（保持与旧版前端一致的默认行为）；
// offset>0 表示从最新一端向前跳过 offset 条后再取 limit 条。
// 返回值：(窗口, 总条数)。
func paginateTail[T any](items []T, limit, offset int) ([]T, int) {
	total := len(items)
	if limit <= 0 {
		limit = total
	}
	if offset < 0 {
		offset = 0
	}
	start := total - offset - limit
	if start < 0 {
		start = 0
	}
	end := total - offset
	if end < 0 {
		end = 0
	}
	if end > total {
		end = total
	}
	if start > end {
		start = end
	}
	return items[start:end], total
}

// setPageHeaders 写分页元数据响应头，供前端/调用方读取 total。
func setPageHeaders(w http.ResponseWriter, total, limit, offset int) {
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Offset", strconv.Itoa(offset))
}
