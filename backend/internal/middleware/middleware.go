// Package middleware 提供 CORS、访问日志、JWT 签发/校验，以及统一 JSON 写出。
// production 环境的 debug 日志在 pkg/logger 被抬升为 info，避免散落 fmt.Println。
package middleware

import (
	"net/http"
	"strings"
	"time"

	"gocooking/pkg/apperr"
	"gocooking/pkg/logger"
	"gocooking/pkg/timeutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint   `json:"uid"`
	Username string `json:"uname"`
	jwt.RegisteredClaims
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ms", time.Since(start).Milliseconds(),
		)
	}
}

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteError(c, apperr.Unauthorized(""))
			c.Abort()
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		claims := &Claims{}
		tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !tok.Valid || claims.UserID == 0 {
			WriteError(c, apperr.Unauthorized("登录已失效"))
			c.Abort()
			return
		}
		c.Set("uid", claims.UserID)
		c.Set("uname", claims.Username)
		c.Next()
	}
}

func Sign(secret string, userID uint, username string) (string, error) {
	now := timeutil.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		// 失败时返回裸的 *AppError；成功时返回未经接口包装的 nil，
		// 否则 *AppError 的 nil 值被装箱进 error 接口后会变成 typed-nil，
		// 调用方的 err != nil 判定永远成立，导致成功签发也被当成 500。
		return "", apperr.Internal(err)
	}
	return token, nil
}

func UID(c *gin.Context) uint {
	v, _ := c.Get("uid")
	id, _ := v.(uint)
	return id
}

func WriteError(c *gin.Context, err error) {
	ae, ok := apperr.As(err)
	if !ok {
		logger.Error("unhandled", "err", err)
		ae = apperr.Internal(err)
	} else if ae.HTTP >= 500 {
		logger.Error("internal", "code", ae.Code, "err", ae.Cause)
	}
	details := make([]gin.H, 0, len(ae.Details))
	for _, d := range ae.Details {
		details = append(details, gin.H{"field": d.Field, "message": d.Message, "code": d.Code})
	}
	body := gin.H{"error": gin.H{"code": ae.Code, "message": ae.Message}}
	if len(details) > 0 {
		body["error"].(gin.H)["details"] = details
	}
	c.JSON(ae.HTTP, body)
}

func WriteData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func WriteList(c *gin.Context, data any, total int64, page, per int) {
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": gin.H{"total": total, "page": page, "per_page": per}})
}

func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		WriteError(c, apperr.InvalidJSON(err))
		return false
	}
	return true
}
