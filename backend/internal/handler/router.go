// Package handler 暴露 /api/v1：
//
//	GET    /health
//	POST   /auth/login
//	GET    /recipes  POST /recipes  GET|PUT|DELETE /recipes/:id  POST /recipes/:id/duplicate
//	GET    /ingredients
//	GET    /meal-plan
//	POST   /meal-plan/slots  PATCH|DELETE /meal-plan/slots/:id
//	POST   /meal-plan/clear  POST /meal-plan/copy-next
//	GET|POST /pantry  PUT|DELETE /pantry/:id
//	POST   /shopping-lists/generate
//	PATCH  /shopping-lists/checks
//	POST   /shopping-lists/restore
//	GET|PUT|POST /settings/staples
//
// 除 health/login 外全部走 JWT。错误体见 pkg/apperr。
package handler

import (
	"gocooking/internal/middleware"
	"gocooking/internal/service"
	"gocooking/pkg/timeutil"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	Catalog *service.Catalog
	Planner *service.Planner
	Secret  string
}

func NewRouter(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS(), middleware.AccessLog())

	v1 := r.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		middleware.WriteData(c, 200, gin.H{"status": "ok", "time": timeutil.FormatDateTime(timeutil.Now())})
	})

	auth := NewAuth(d)
	v1.POST("/auth/login", auth.Login)

	api := v1.Group("")
	api.Use(middleware.JWT(d.Secret))

	rec := NewRecipe(d)
	api.GET("/recipes", rec.List)
	api.GET("/recipes/:id", rec.Get)
	api.POST("/recipes", rec.Create)
	api.PUT("/recipes/:id", rec.Update)
	api.DELETE("/recipes/:id", rec.Delete)
	api.POST("/recipes/:id/duplicate", rec.Duplicate)

	ing := NewIngredient(d)
	api.GET("/ingredients", ing.List)

	mp := NewMealPlan(d)
	api.GET("/meal-plan", mp.Week)
	api.POST("/meal-plan/slots", mp.Add)
	api.PATCH("/meal-plan/slots/:id", mp.Patch)
	api.DELETE("/meal-plan/slots/:id", mp.Delete)
	api.POST("/meal-plan/clear", mp.Clear)
	api.POST("/meal-plan/copy-next", mp.CopyNext)

	pn := NewPantry(d)
	api.GET("/pantry", pn.List)
	api.POST("/pantry", pn.Create)
	api.PUT("/pantry/:id", pn.Update)
	api.DELETE("/pantry/:id", pn.Delete)

	sh := NewShopping(d)
	api.POST("/shopping-lists/generate", sh.Generate)
	api.PATCH("/shopping-lists/checks", sh.Check)
	api.POST("/shopping-lists/restore", sh.Restore)

	st := NewSettings(d)
	api.GET("/settings/staples", st.List)
	api.PUT("/settings/staples", st.Put)
	api.POST("/settings/staples", st.Add)
	return r
}
