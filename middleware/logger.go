package middleware

import (
	"bytes"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 需要记录请求体的 Content-Type 前缀
var loggableContentTypes = []string{
	"application/json",
	"application/x-www-form-urlencoded",
	"text/plain",
}

// 敏感字段（请求头中不记录的字段，避免泄露密钥）
var sensitiveHeaders = map[string]bool{
	"X-Internal-Secret": true,
	"X-App-Sign":        true,
	"Authorization":     true,
}

// 请求体最大记录长度（超过则截断）
const maxBodyLogSize = 4096

// 响应体最大记录长度
const maxRespLogSize = 4096

// responseWriter 包装 gin.ResponseWriter，用于捕获响应体
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b) // 同时写入缓冲区
	return w.ResponseWriter.Write(b)
}

// Logger 请求/响应日志中间件
// 记录各接口的输入（方法、路径、查询参数、请求头、请求体）和输出（状态码、响应体）
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 健康检查接口不记录日志
		if path == "/health" || path == "/health/stats" {
			c.Next()
			return
		}

		start := time.Now()

		// ── 记录请求输入 ──
		method := c.Request.Method
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()

		// 收集非敏感请求头
		headerParts := make([]string, 0)
		for key, values := range c.Request.Header {
			if sensitiveHeaders[key] {
				headerParts = append(headerParts, key+"=***")
			} else {
				headerParts = append(headerParts, key+"="+strings.Join(values, ","))
			}
		}
		headerStr := strings.Join(headerParts, "; ")

		// 读取请求体（仅对可记录的 Content-Type）
		var reqBody string
		if shouldLogBody(c.ContentType()) {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				// 恢复请求体，供后续 handler 读取
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				reqBody = truncate(string(bodyBytes), maxBodyLogSize)
			}
		}

		// ── 包装 ResponseWriter 以捕获响应体 ──
		respWriter := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = respWriter

		// 执行后续处理
		c.Next()

		// ── 记录响应输出 ──
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		respBody := truncate(respWriter.body.String(), maxRespLogSize)

		// 构建日志输出
		logBuilder := strings.Builder{}
		logBuilder.WriteString("────────────────────────────────────────\n")
		logBuilder.WriteString("  ▶ 请求: " + method + " " + path)
		if query != "" {
			logBuilder.WriteString("?" + query)
		}
		logBuilder.WriteString("\n")
		logBuilder.WriteString("  ▶ 来源IP: " + clientIP + "\n")
		if headerStr != "" {
			logBuilder.WriteString("  ▶ 请求头: " + headerStr + "\n")
		}
		if reqBody != "" {
			logBuilder.WriteString("  ▶ 请求体: " + reqBody + "\n")
		}
		logBuilder.WriteString("  ◀ 状态码: " + statusCodeStr(statusCode) + "\n")
		if respBody != "" {
			logBuilder.WriteString("  ◀ 响应体: " + respBody + "\n")
		}
		logBuilder.WriteString("  ⏱ 耗时: " + duration.String() + "\n")
		logBuilder.WriteString("────────────────────────────────────────")

		log.Println(logBuilder.String())
	}
}

// shouldLogBody 判断是否需要记录请求体
func shouldLogBody(contentType string) bool {
	ct := strings.ToLower(contentType)
	for _, prefix := range loggableContentTypes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}

// truncate 截断字符串，超过最大长度时添加省略提示
func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(已截断)"
}

// statusCodeStr 将状态码转为带描述的字符串
func statusCodeStr(code int) string {
	s := strconv.Itoa(code)
	switch {
	case code >= 200 && code < 300:
		return s + " ✅"
	case code >= 400 && code < 500:
		return s + " ⚠️"
	case code >= 500:
		return s + " ❌"
	default:
		return s
	}
}
