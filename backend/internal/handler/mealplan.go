package handler

// MealPlan：周视图、拖入槽位、改份数、拖出删除、清空、复制到下周。
// week 缺省时用北京时间今天所在周。slot ∈ breakfast|lunch|dinner。
// servings_multiplier 闭区间 [0.5, 4]。

import (
	"net/http"

	"gocooking/internal/dto"
	"gocooking/internal/middleware"
	"gocooking/pkg/apperr"
	"gocooking/pkg/timeutil"

	"github.com/gin-gonic/gin"
)

type MealPlan struct{ Deps }

func NewMealPlan(d Deps) *MealPlan { return &MealPlan{d} }

func (h *MealPlan) Week(c *gin.Context) {
	week := c.Query("week")
	if week == "" {
		week = timeutil.FormatDate(timeutil.Today())
	}
	out, err := h.Planner.WeekPlan(middleware.UID(c), week)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}

func (h *MealPlan) Add(c *gin.Context) {
	var in dto.SlotIn
	if !middleware.BindJSON(c, &in) {
		return
	}
	out, err := h.Planner.AddSlot(middleware.UID(c), in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusCreated, out)
}

func (h *MealPlan) Patch(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	var in dto.SlotPatch
	if !middleware.BindJSON(c, &in) {
		return
	}
	out, err := h.Planner.PatchSlot(middleware.UID(c), id, in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}

func (h *MealPlan) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	if err := h.Planner.DeleteSlot(middleware.UID(c), id); err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *MealPlan) Clear(c *gin.Context) {
	var in dto.WeekOp
	if !middleware.BindJSON(c, &in) {
		return
	}
	if in.Week == "" {
		middleware.WriteError(c, apperr.Required("week"))
		return
	}
	if err := h.Planner.ClearWeek(middleware.UID(c), in.Week); err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *MealPlan) CopyNext(c *gin.Context) {
	var in dto.WeekOp
	if !middleware.BindJSON(c, &in) {
		return
	}
	if in.Week == "" {
		middleware.WriteError(c, apperr.Required("week"))
		return
	}
	if err := h.Planner.CopyNext(middleware.UID(c), in.Week); err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, gin.H{"ok": true})
}
