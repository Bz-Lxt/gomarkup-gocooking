package handler

// Pantry CRUD。status 由服务层按北京时间计算：expired / soon(≤3天) / ok。

import (
	"net/http"

	"gocooking/internal/dto"
	"gocooking/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Pantry struct{ Deps }

func NewPantry(d Deps) *Pantry { return &Pantry{d} }

func (h *Pantry) List(c *gin.Context) {
	out, err := h.Planner.ListPantry(c.Request.Context(), middleware.UID(c))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}

func (h *Pantry) Create(c *gin.Context) {
	var in dto.PantryIn
	if !middleware.BindJSON(c, &in) {
		return
	}
	out, err := h.Planner.CreatePantry(c.Request.Context(), middleware.UID(c), in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusCreated, out)
}

func (h *Pantry) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	var in dto.PantryIn
	if !middleware.BindJSON(c, &in) {
		return
	}
	out, err := h.Planner.UpdatePantry(c.Request.Context(), middleware.UID(c), id, in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}

func (h *Pantry) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	if err := h.Planner.DeletePantry(c.Request.Context(), middleware.UID(c), id); err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
