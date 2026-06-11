package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
)

type ProfileHandler struct {
	common.BaseHandler
}

func (h *ProfileHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/profile", h.GetProfile)
	g.POST("/profile/update", h.UpdateProfile)
}

func (h *ProfileHandler) GetProfile(c echo.Context) error {
	cc := h.Context(c)
	if cc.AuthUser == nil {
		return h.Error(401, "用户未登录")
	}
	return c.JSON(200, common.NewData(cc.AuthUser))
}

type updateProfileRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
}

func (h *ProfileHandler) UpdateProfile(c echo.Context) error {
	cc := h.Context(c)
	if cc.AuthUser == nil {
		return h.Error(401, "用户未登录")
	}

	var req updateProfileRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Department != "" {
		updates["department"] = req.Department
	}

	if len(updates) == 0 {
		return c.JSON(200, common.NewData(cc.AuthUser))
	}

	var user model.User
	if err := h.DB.Model(&model.User{}).Where("id = ?", cc.AuthUser.ID).Updates(updates).First(&user).Error; err != nil {
		return h.Error(-22, err.Error())
	}
	return c.JSON(200, common.NewData(user))
}
