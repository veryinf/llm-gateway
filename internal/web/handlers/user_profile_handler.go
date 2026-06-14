package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"
	"strings"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type ProfileHandler struct {
	common.BaseHandler
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=6"`
}

// GetProfile 获取用户资料
func (h *ProfileHandler) GetProfile(c echo.Context) error {
	ctx := h.Context(c)
	user := ctx.AuthUser
	// 不返回密码字段
	user.UID = 0
	user.Password = ""
	return common.NewData(user)
}

// UpdateProfile 修改密码
func (h *ProfileHandler) UpdateProfile(c echo.Context) error {
	ctx := h.Context(c)
	input := &struct {
		model.User
		NewPassword string `json:"newPassword"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Password, validation.Required.When(input.Password != "")),
		validation.Field(&input.NewPassword, validation.Required.When(input.NewPassword != "")),
	); err != nil {
		return err
	}
	user := ctx.AuthUser
	if input.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
			return h.Error(-11, "原密码错误")
		}
		password, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
		user.Password = string(password)
		if err := h.DB.Save(user).Error; err != nil {
			return h.Error(-22, "更新密码失败")
		}
	}

	return h.Success()
}

// ResetAccessKey 重置访问密钥
func (h *ProfileHandler) ResetAccessKey(c echo.Context) error {
	ctx := h.Context(c)
	user := ctx.AuthUser
	for {
		user.AccessKey = strings.Replace(uuid.New().String(), "-", "", -1)[:16]
		user.SecretKey = strings.Replace(uuid.New().String(), "-", "", -1)
		var count int64
		_ = h.DB.Model(&model.User{}).Where("access_key = ?", user.AccessKey).Count(&count).Error
		if count == 0 {
			if err := h.DB.Save(user).Error; err != nil {
				return h.Error(-22, "重置密钥失败")
			}
			return h.Success()
		}
	}
}

func (h *ProfileHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/profile", h.GetProfile)
	g.POST("/profile/update", h.UpdateProfile)
	g.POST("/profile/resetKey", h.ResetAccessKey)
}
