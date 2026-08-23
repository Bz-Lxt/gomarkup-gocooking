package apperr

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestCancelled(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain context.Canceled", context.Canceled, true},
		{"plain context.DeadlineExceeded", context.DeadlineExceeded, true},
		{"wrapped Canceled", fmt.Errorf("query: %w", context.Canceled), true},
		{"wrapped DeadlineExceeded", fmt.Errorf("db: %w", context.DeadlineExceeded), true},
		{"unrelated error", errors.New("connection refused"), false},
		{"apperr internal", Internal(errors.New("boom")), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Cancelled(tt.err); got != tt.want {
				t.Errorf("Cancelled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCancelledWithNestedWrap(t *testing.T) {
	// 模拟 GORM/pqx 返回的多层包装错误
	inner := fmt.Errorf("pgx query: %w", context.Canceled)
	outer := fmt.Errorf("gorm: %w", inner)
	if !Cancelled(outer) {
		t.Error("expected nested wrapped context.Canceled to be detected")
	}
}
