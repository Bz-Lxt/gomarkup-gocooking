package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gocooking/pkg/apperr"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestAccessLogPropagatesCancel 验证 AccessLog 中间件不会剥离 context 的取消信号。
// 之前的实现用 context.WithoutCancel 切断取消传播，导致客户端断开后查询仍继续。
func TestAccessLogPropagatesCancel(t *testing.T) {
	var ctxAfterMiddleware context.Context

	r := gin.New()
	r.Use(AccessLog())
	r.GET("/test", func(c *gin.Context) {
		ctxAfterMiddleware = c.Request.Context()
		c.Status(200)
	})

	// 创建一个会被外部取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// AccessLog 不应改变请求 context 的取消行为
	if ctxAfterMiddleware == nil {
		t.Fatal("context not captured")
	}
	// 关键断言：中间件后的 context 必须在父 context 取消时也被取消
	cancel()
	select {
	case <-ctxAfterMiddleware.Done():
		// 正确：取消信号传播下来了
	default:
		t.Fatal("AccessLog middleware broke cancel propagation: ctx not done after parent cancel")
	}
}

// TestWriteErrorHandlesCancellation 验证 WriteError 对 context 取消错误做正确处理。
func TestWriteErrorHandlesCancellation(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// 模拟 GORM 查询因 context 取消返回的错误
	WriteError(c, context.Canceled)

	// 应记录 503 并 abort，不应尝试写 JSON 错误体
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body on cancelled request, got %q", w.Body.String())
	}
}

// TestWriteErrorNormalPath 验证普通业务错误仍然正常写 JSON。
func TestWriteErrorNormalPath(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	WriteError(c, apperr.NotFound("菜谱"))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty JSON body for normal error")
	}
}
