package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

// TestAsUnwrapsWrappedError 验证 fmt.Errorf %w 包装过的 AppError
// 仍能被 As 识别为原始业务错误（保持 422 + validation_error），
// 而不是退化成 internal_error/500。
func TestAsUnwrapsWrappedError(t *testing.T) {
	orig := Validation("日期格式错误", FieldError{Field: "date", Message: "格式错误", Code: "invalid_format"})
	wrapped := fmt.Errorf("生成购物清单失败: %w", orig)

	ae, ok := As(wrapped)
	if !ok {
		t.Fatalf("As 应能从 wrapped 错误中解出 AppError")
	}
	if ae.HTTP != http.StatusUnprocessableEntity {
		t.Fatalf("状态码应保持 422, got %d", ae.HTTP)
	}
	if ae.Code != CodeValidation {
		t.Fatalf("错误码应保持 validation_error, got %s", ae.Code)
	}
	if len(ae.Details) != 1 || ae.Details[0].Field != "date" {
		t.Fatalf("字段错误应保留: %+v", ae.Details)
	}
}

func TestAsOnUnwrappedAppError(t *testing.T) {
	orig := NotFound("菜谱")
	ae, ok := As(orig)
	if !ok || ae.Code != CodeNotFound {
		t.Fatalf("未包装的 AppError 应直接识别: %v %s", ok, ae.Code)
	}
}

func TestAsOnNonAppError(t *testing.T) {
	_, ok := As(errors.New("plain error"))
	if ok {
		t.Fatal("普通 error 不应被识别为 AppError")
	}
}

func TestIsTraversesWrapChain(t *testing.T) {
	wrapped := fmt.Errorf("ctx: %w", Validation("x"))
	if !Is(wrapped, CodeValidation) {
		t.Fatal("Is 应穿透 wrap 链识别 code")
	}
}

func TestWriteErrorPreservesStatusForWrappedValidation(t *testing.T) {
	// 模拟服务层补上下文 + handler WriteError 路径。
	orig := Required("ingredient_id")
	wrapped := fmt.Errorf("add slot: %w", orig)

	ae, ok := As(wrapped)
	if !ok {
		t.Fatal("应解出 AppError")
	}
	// 与 middleware.WriteError 同样的判定逻辑。
	if !ok || ae.HTTP >= 500 {
		t.Fatalf("包装后的校验错误不应退化为 5xx: HTTP=%d", ae.HTTP)
	}
	if ae.Code == CodeInternal {
		t.Fatal("包装后的校验错误不应变成 internal_error")
	}
}
