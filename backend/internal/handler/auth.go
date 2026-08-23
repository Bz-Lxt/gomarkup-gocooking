package handler

// Auth 只负责登录。JWT 有效期 7 天，密钥来自 JWT_SECRET。
// 失败统一 401 + unauthorized，不区分「用户不存在」以免枚举账号。

import (
	"net/http"

	"gocooking/internal/dto"
	"gocooking/internal/middleware"
	"gocooking/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type Auth struct{ Deps }

func NewAuth(d Deps) *Auth { return &Auth{d} }

func (h *Auth) Login(c *gin.Context) {
	var req dto.LoginReq
	if !middleware.BindJSON(c, &req) {
		return
	}
	u, err := h.Catalog.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	tok, err := middleware.Sign(h.Secret, u.ID, u.Username)
	if err != nil {
		middleware.WriteError(c, apperr.Internal(err))
		return
	}
	middleware.WriteData(c, http.StatusOK, dto.LoginResp{Token: tok, Username: u.Username})
}
