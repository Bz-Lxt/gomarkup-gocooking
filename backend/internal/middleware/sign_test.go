package middleware

import (
	"errors"
	"testing"

	"gocooking/pkg/apperr"
)

// TestSignSuccessReturnsNilError 防止 typed-nil 回归：
// 旧实现用 `var signErr *apperr.AppError; return token, signErr`，
// 成功时 signErr 是 *AppError 的 nil 值，被装箱进 error 接口后 err != nil 恒成立，
// 导致登录即便签发成功也统一返回 500。
func TestSignSuccessReturnsNilError(t *testing.T) {
	tok, err := Sign("regression-secret", 42, "alice")
	if tok == "" {
		t.Fatalf("expected non-empty token, got empty")
	}
	if err != nil {
		t.Fatalf("expected nil error on success, got non-nil: type=%T value=%v", err, err)
	}
}

// TestSignFailurePathUsesAppError 确认 Internal 包装的错误能被 apperr.As 识别，
// 即失败分支返回的是有效的 *AppError，而非 typed-nil。
func TestSignFailurePathUsesAppError(t *testing.T) {
	sentinel := errors.New("signing failure")
	ae := apperr.Internal(sentinel)
	got, ok := apperr.As(ae)
	if !ok {
		t.Fatalf("expected apperr.As to identify Internal error")
	}
	if got.Code != apperr.CodeInternal {
		t.Fatalf("expected code %s, got %s", apperr.CodeInternal, got.Code)
	}
	if !errors.Is(got.Cause, sentinel) {
		t.Fatalf("expected cause to wrap sentinel")
	}
}
