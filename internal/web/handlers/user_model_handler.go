package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
)

type UserModelHandler struct {
	common.BaseHandler
}

func (h *UserModelHandler) SearchUserModels(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}
	query := h.DB.Model(&model.UserModel{}).Order("user_model_id DESC")

	if input.Kw != "" {
		kw := "%" + input.Kw + "%"
		query = query.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?", kw, kw, kw)
	}

	for _, filter := range input.Filters {
		switch filter.Field {
		case "isActive":
			if err := validation.Validate(filter.Value, validation.Required); err != nil {
				return h.Error(-11, err.Error())
			}
			query = query.Where("is_active = ?", filter.Value)
		}
	}

	var count int64
	var models []model.UserModel
	if err := h.Pagination(&input.Pagination, query, &models, &count); err != nil {
		return err
	}

	return common.NewDataSet(models, count)
}

func (h *UserModelHandler) FetchUserModel(c echo.Context) error {
	input := &struct {
		UserModelID uint `json:"userModelId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.UserModelID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	var m model.UserModel
	if err := h.DB.First(&m, input.UserModelID).Error; err != nil {
		return h.Error(-24, "模型不存在")
	}
	return common.NewData(m)
}

func (h *UserModelHandler) AddUserModel(c echo.Context) error {
	input := &model.UserModel{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Name, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	var exist int64
	h.DB.Model(&model.UserModel{}).Where("name = ?", input.Name).Count(&exist)
	if exist > 0 {
		return h.Error(-12, "模型名称已存在")
	}

	input.IsActive = true
	if err := h.DB.Create(input).Error; err != nil {
		return h.Error(-21, err.Error())
	}
	return common.NewData(input)
}

func (h *UserModelHandler) UpdateUserModel(c echo.Context) error {
	input, err := h.GetJSON(c)
	if err != nil {
		return err
	}
	userModelID := input.Get("userModelId")
	if !userModelID.Exists() || userModelID.Uint() == 0 {
		return h.Error(-23, "userModelId is required")
	}

	newState := map[string]any{}
	if input.Get("name").Exists() {
		name := input.Get("name").String()
		if err := validation.Validate(name, validation.Required); err != nil {
			return h.Error(-11, err.Error())
		}
		newState["name"] = name
	}
	if input.Get("displayName").Exists() {
		newState["display_name"] = input.Get("displayName").String()
	}
	if input.Get("description").Exists() {
		newState["description"] = input.Get("description").String()
	}
	if input.Get("isActive").Exists() {
		newState["is_active"] = input.Get("isActive").Bool()
	}

	if len(newState) == 0 {
		return h.Success()
	}
	if err := h.DB.Model(&model.UserModel{}).Where("user_model_id = ?", userModelID.Uint()).Updates(newState).Error; err != nil {
		return h.Error(-22, err.Error())
	}
	return h.Success()
}

func (h *UserModelHandler) RemoveUserModel(c echo.Context) error {
	input := &struct {
		UserModelID uint `json:"userModelId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.UserModelID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	if err := h.DB.Delete(&model.UserModel{}, input.UserModelID).Error; err != nil {
		return h.Error(-23, err.Error())
	}
	return h.Success()
}

func (h *UserModelHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/user-models/search", h.SearchUserModels)
	g.POST("/user-models/fetch", h.FetchUserModel)
	g.POST("/user-models/add", h.AddUserModel)
	g.POST("/user-models/update", h.UpdateUserModel)
	g.POST("/user-models/remove", h.RemoveUserModel)
}
