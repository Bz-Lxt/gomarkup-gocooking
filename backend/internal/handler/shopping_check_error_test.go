package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gocooking/internal/handler"
	"gocooking/internal/middleware"
	"gocooking/internal/service"
	"gocooking/pkg/apperr"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func TestShoppingCheckReportsPersistenceFailure(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=test dbname=test sslmode=disable"), &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		Logger:                 glogger.Default.LogMode(glogger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	writeErr := errors.New("shopping check write interrupted")
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_shopping_check_write", func(tx *gorm.DB) {
		tx.AddError(writeErr)
	}); err != nil {
		t.Fatalf("register write failure: %v", err)
	}

	const secret = "shopping-check-error-test-secret"
	router := handler.NewRouter(handler.Deps{
		Catalog: service.NewCatalog(db),
		Planner: service.NewPlanner(db),
		Secret:  secret,
	})
	token, err := middleware.Sign(secret, 7, "tester")
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/shopping-lists/checks", strings.NewReader(`{
		"from": "2026-08-24",
		"to": "2026-08-24",
		"ingredient_id": 1,
		"unit": "g",
		"dimension": "weight",
		"checked": true
	}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusInternalServerError, res.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != apperr.CodeInternal {
		t.Fatalf("error code = %q, want %q", body.Error.Code, apperr.CodeInternal)
	}
}
