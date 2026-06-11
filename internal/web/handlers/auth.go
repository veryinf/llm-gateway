package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	common.BaseHandler
	TokenManager *common.TokenManager
}

func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/login", h.Login)
	g.POST("/logout", h.Logout)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if req.Username == "" || req.Password == "" {
		return h.Error(-11, "用户名和密码不能为空")
	}

	var user model.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return h.Error(-11, "用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return h.Error(-11, "用户名或密码错误")
	}

	token := h.TokenManager.CreateToken(user.ID)
	return c.JSON(200, common.NewData(map[string]string{"token": token}))
}

func (h *AuthHandler) Logout(c echo.Context) error {
	cc := h.Context(c)
	if cc.AuthToken != nil {
		h.TokenManager.DeleteToken(cc.AuthToken.Token)
	}
	return c.JSON(200, common.NewResponse(0, "ok"))
}
