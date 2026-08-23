package middleware_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gocooking/internal/middleware"
	"gocooking/pkg/apperr"

	"github.com/gin-gonic/gin"
)

func TestWriteErrorPreservesWrappedAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/validate", func(c *gin.Context) {
		err := fmt.Errorf("validate meal plan: %w", apperr.Required("week"))
		middleware.WriteError(c, err)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/validate", nil))

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusUnprocessableEntity, recorder.Body.String())
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != apperr.CodeValidation {
		t.Fatalf("error code = %q, want %q", response.Error.Code, apperr.CodeValidation)
	}
}
