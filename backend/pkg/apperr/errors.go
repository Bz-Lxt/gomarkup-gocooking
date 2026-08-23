// Package apperr 把业务错误映射到 HTTP 状态与稳定错误码。
// handler 只调用 WriteError，不拼接内部堆栈给客户端。
package apperr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	CodeInvalidJSON  = "invalid_json"
	CodeValidation   = "validation_error"
	CodeUnauthorized = "unauthorized"
	CodeNotFound     = "not_found"
	CodeConflict     = "conflict"
	CodeInternal     = "internal_error"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type AppError struct {
	HTTP    int
	Code    string
	Message string
	Details []FieldError
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Cause)
	}
	return e.Code + ": " + e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

func InvalidJSON(err error) *AppError {
	return &AppError{HTTP: http.StatusBadRequest, Code: CodeInvalidJSON, Message: "请求体不是合法 JSON", Cause: err}
}

func Validation(msg string, details ...FieldError) *AppError {
	if msg == "" {
		msg = "请求校验失败"
	}
	return &AppError{HTTP: http.StatusUnprocessableEntity, Code: CodeValidation, Message: msg, Details: details}
}

func Required(field string) *AppError {
	return Validation("请求校验失败", FieldError{Field: field, Message: "必填", Code: "required"})
}

func Unauthorized(msg string) *AppError {
	if msg == "" {
		msg = "未登录或登录已失效"
	}
	return &AppError{HTTP: http.StatusUnauthorized, Code: CodeUnauthorized, Message: msg}
}

func NotFound(resource string) *AppError {
	return &AppError{HTTP: http.StatusNotFound, Code: CodeNotFound, Message: resource + "不存在"}
}

func Conflict(msg string) *AppError {
	return &AppError{HTTP: http.StatusConflict, Code: CodeConflict, Message: msg}
}

func Internal(err error) *AppError {
	return &AppError{HTTP: http.StatusInternalServerError, Code: CodeInternal, Message: "服务内部错误", Cause: err}
}

func Is(err error, code string) bool {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code == code
	}
	return false
}

// Cancelled 判断 err 是否由 context 取消/超时引起（客户端断开或请求超时）。
// GORM/pqx 在查询被取消时返回的 error 经 errors.Is 可追溯到 context.Canceled。
func Cancelled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

func JoinDetails(errs []FieldError) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, e.Field+": "+e.Message)
	}
	return strings.Join(parts, "; ")
}
