package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	common.BaseHandler
}

// Login 用户登录
func (h *AuthHandler) Login(c echo.Context) error {
	input := &struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Username, validation.Required),
		validation.Field(&input.Password, validation.Required),
	); err != nil {
		return err
	}

	var user model.User
	if err := h.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		return h.Error(-11, "用户名或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return h.Error(-11, "用户名或密码错误")
	}

	token := h.TokenManager.CreateToken(user.UID)
	ret := map[string]any{
		"token": token,
	}
	return common.NewData(ret)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c echo.Context) error {
	lc := h.Context(c)
	if lc.AuthToken != nil {
		h.TokenManager.DeleteToken(lc.AuthToken.Token)
	}
	return h.Success()
}

func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/auth/login", h.Login)
	g.POST("/auth/logout", h.Logout)
}
