package handler

// Recipe：系统菜谱 UserID 为空，只允许复制后改；删除仅限用户自己的副本。

import (
	"net/http"
	"strconv"

	"gocooking/internal/dto"
	"gocooking/internal/middleware"
	"gocooking/internal/service"
	"gocooking/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type Recipe struct{ Deps }

func NewRecipe(d Deps) *Recipe { return &Recipe{d} }

func (h *Recipe) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	per, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	rows, total, err := h.Catalog.ListRecipes(c.Request.Context(), middleware.UID(c), c.Query("q"), c.Query("tag"), page, per)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	out := make([]dto.RecipeOut, 0, len(rows))
	for _, r := range rows {
		out = append(out, service.RecipeOut(r))
	}
	if page < 1 {
		page = 1
	}
	if per < 1 {
		per = 50
	}
	middleware.WriteList(c, out, total, page, per)
}

func (h *Recipe) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	r, err := h.Catalog.GetRecipe(c.Request.Context(), middleware.UID(c), id)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, service.RecipeOut(*r))
}

func (h *Recipe) Create(c *gin.Context) {
	var in dto.RecipeIn
	if !middleware.BindJSON(c, &in) {
		return
	}
	r, err := h.Catalog.CreateRecipe(c.Request.Context(), middleware.UID(c), in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.Header("Location", "/api/v1/recipes/"+strconv.Itoa(int(r.ID)))
	middleware.WriteData(c, http.StatusCreated, service.RecipeOut(*r))
}

func (h *Recipe) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	var in dto.RecipeIn
	if !middleware.BindJSON(c, &in) {
		return
	}
	r, err := h.Catalog.UpdateRecipe(c.Request.Context(), middleware.UID(c), id, in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, service.RecipeOut(*r))
}

func (h *Recipe) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	if err := h.Catalog.DeleteRecipe(c.Request.Context(), middleware.UID(c), id); err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Recipe) Duplicate(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	r, err := h.Catalog.DuplicateRecipe(c.Request.Context(), middleware.UID(c), id)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusCreated, service.RecipeOut(*r))
}

func parseID(c *gin.Context) (uint, error) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || n == 0 {
		return 0, apperr.Validation("无效的 id", apperr.FieldError{Field: "id", Message: "须为正整数", Code: "invalid"})
	}
	return uint(n), nil
}
