package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
)

type UserModelRouterHandler struct {
	common.BaseHandler
}

func (h *UserModelRouterHandler) SearchUserModelRouters(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}

	query := h.DB.Model(&model.UserModelRouter{}).Order("priority ASC, router_id ASC")

	for _, filter := range input.Filters {
		switch filter.Field {
		case "userModelId":
			query = query.Where("user_model_id = ?", filter.Value)
		}
	}

	var count int64
	var routers []model.UserModelRouter
	if err := h.Pagination(&input.Pagination, query, &routers, &count); err != nil {
		return err
	}

	return common.NewDataSet(routers, count)
}

func (h *UserModelRouterHandler) FetchUserModelRouter(c echo.Context) error {
	input := &struct {
		RouterID uint `json:"routerId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.RouterID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	var r model.UserModelRouter
	if err := h.DB.First(&r, input.RouterID).Error; err != nil {
		return h.Error(-24, "路由规则不存在")
	}
	return common.NewData(r)
}

func (h *UserModelRouterHandler) AddUserModelRouter(c echo.Context) error {
	input := &model.UserModelRouter{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.UserModelID, validation.Required),
		validation.Field(&input.ProviderModelID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	input.IsActive = true

	if err := h.DB.Create(input).Error; err != nil {
		return h.Error(-21, err.Error())
	}
	return common.NewData(input)
}

func (h *UserModelRouterHandler) UpdateUserModelRouter(c echo.Context) error {
	input, err := h.GetJSON(c)
	if err != nil {
		return err
	}
	routerID := input.Get("routerId")
	if !routerID.Exists() || routerID.Uint() == 0 {
		return h.Error(-23, "routerId is required")
	}

	newState := map[string]any{}
	if input.Get("providerModelId").Exists() {
		newState["provider_model_id"] = input.Get("providerModelId").Uint()
	}
	if input.Get("priority").Exists() {
		newState["priority"] = input.Get("priority").Uint()
	}
	if input.Get("isActive").Exists() {
		newState["is_active"] = input.Get("isActive").Bool()
	}

	if len(newState) == 0 {
		return h.Success()
	}
	if err := h.DB.Model(&model.UserModelRouter{}).Where("router_id = ?", routerID.Uint()).Updates(newState).Error; err != nil {
		return h.Error(-22, err.Error())
	}
	return h.Success()
}

func (h *UserModelRouterHandler) RemoveUserModelRouter(c echo.Context) error {
	input := &struct {
		RouterID uint `json:"routerId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.RouterID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	if err := h.DB.Delete(&model.UserModelRouter{}, input.RouterID).Error; err != nil {
		return h.Error(-23, err.Error())
	}
	return h.Success()
}

func (h *UserModelRouterHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/user-model-routers/search", h.SearchUserModelRouters)
	g.POST("/user-model-routers/fetch", h.FetchUserModelRouter)
	g.POST("/user-model-routers/add", h.AddUserModelRouter)
	g.POST("/user-model-routers/update", h.UpdateUserModelRouter)
	g.POST("/user-model-routers/remove", h.RemoveUserModelRouter)
}
