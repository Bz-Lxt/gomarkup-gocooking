package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gocooking/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestAccessLogPreservesRequestCancellation(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.AccessLog())
	router.GET("/work", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.Status(http.StatusRequestTimeout)
		default:
			c.Status(http.StatusOK)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/work", nil).WithContext(ctx)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusRequestTimeout {
		t.Fatalf("canceled request reached handler with status %d; want %d", resp.Code, http.StatusRequestTimeout)
	}
}
