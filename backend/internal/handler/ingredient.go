package handler

// Ingredient 只读目录。q 同时搜 name 与 aliases，保证「番茄」能找到西红柿。

import (
	"strconv"

	"gocooking/internal/middleware"
	"gocooking/internal/service"

	"github.com/gin-gonic/gin"
)

type Ingredient struct{ Deps }

func NewIngredient(d Deps) *Ingredient { return &Ingredient{d} }

func (h *Ingredient) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	per, _ := strconv.Atoi(c.DefaultQuery("per_page", "200"))
	rows, total, err := h.Catalog.ListIngredients(c.Query("q"), c.Query("stall"), c.Query("category"), page, per)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, service.IngredientOut(r))
	}
	if page < 1 {
		page = 1
	}
	if per < 1 {
		per = 200
	}
	middleware.WriteList(c, out, total, page, per)
}
