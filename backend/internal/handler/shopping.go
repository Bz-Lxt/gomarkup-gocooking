package handler

// Shopping 三个动作：
//   POST generate — 默认本周一到周日
//   PATCH checks  — 勾选持久化，刷新不丢
//   POST restore  — 把常备项临时加回清单（盐用完了）
// 前端必须传 check_unit（基准单位），不要传展示单位「斤」。

import (
	"net/http"

	"gocooking/internal/dto"
	"gocooking/internal/middleware"
	"gocooking/pkg/timeutil"

	"github.com/gin-gonic/gin"
)

type Shopping struct{ Deps }

func NewShopping(d Deps) *Shopping { return &Shopping{d} }

func (h *Shopping) Generate(c *gin.Context) {
	var in dto.GenerateReq
	if !middleware.BindJSON(c, &in) {
		return
	}
	if in.From == "" {
		in.From = timeutil.FormatDate(timeutil.StartOfWeek(timeutil.Today()))
	}
	if in.To == "" {
		in.To = timeutil.FormatDate(timeutil.EndOfWeek(timeutil.Today()))
	}
	out, err := h.Planner.Generate(middleware.UID(c), in.From, in.To)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}

func (h *Shopping) Check(c *gin.Context) {
	var in dto.CheckReq
	if !middleware.BindJSON(c, &in) {
		return
	}
	if err := h.Planner.SetCheck(middleware.UID(c), in); err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Shopping) Restore(c *gin.Context) {
	var in dto.RestoreReq
	if !middleware.BindJSON(c, &in) {
		return
	}
	if err := h.Planner.Restore(middleware.UID(c), in); err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, gin.H{"ok": true})
}
