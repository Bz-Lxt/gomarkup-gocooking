package handler

// Settings 常备开关。PUT 批量覆盖，POST 追加自定义常备项。

import (
	"net/http"

	"gocooking/internal/dto"
	"gocooking/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Settings struct{ Deps }

func NewSettings(d Deps) *Settings { return &Settings{d} }

func (h *Settings) List(c *gin.Context) {
	out, err := h.Planner.Staples(middleware.UID(c))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}

func (h *Settings) Put(c *gin.Context) {
	var in dto.StaplesPut
	if !middleware.BindJSON(c, &in) {
		return
	}
	out, err := h.Planner.PutStaples(middleware.UID(c), in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}

func (h *Settings) Add(c *gin.Context) {
	var in dto.StapleAdd
	if !middleware.BindJSON(c, &in) {
		return
	}
	out, err := h.Planner.AddStaple(middleware.UID(c), in)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	middleware.WriteData(c, http.StatusOK, out)
}
