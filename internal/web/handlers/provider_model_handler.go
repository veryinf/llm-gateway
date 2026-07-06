package handlers

import (
	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
)

type ProviderModelHandler struct {
	common.BaseHandler
}

func (h *ProviderModelHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/provider-models/search", h.SearchProviderModels)
	g.POST("/provider-models/fetch", h.FetchProviderModel)
	g.POST("/provider-models/add", h.AddProviderModel)
	g.POST("/provider-models/update", h.UpdateProviderModel)
	g.POST("/provider-models/remove", h.RemoveProviderModel)
}

// SearchProviderModels 搜索 ProviderModel 列表
func (h *ProviderModelHandler) SearchProviderModels(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}

	query := h.DB.Model(&model.ProviderModel{}).Order("model_id DESC")

	// 关键词搜索
	if pattern := input.EscapedKw(); pattern != "" {
		query = query.Where("name LIKE ? ESCAPE '\\' OR display_name LIKE ? ESCAPE '\\'", pattern, pattern)
	}

	// 过滤条件
	for _, filter := range input.Filters {
		switch filter.Field {
		case "providerId":
			query = query.Where("provider_id = ?", filter.Value)
		case "isActive":
			query = query.Where("is_active = ?", filter.Value)
		}
	}

	// 分页查询
	var count int64
	var models []model.ProviderModel
	if err := h.Pagination(&input.Pagination, query, &models, &count); err != nil {
		return err
	}

	return common.NewDataSet(models, count)
}

// FetchProviderModel 获取单条 ProviderModel
func (h *ProviderModelHandler) FetchProviderModel(c echo.Context) error {
	input := &struct {
		ModelID int64 `json:"modelId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.ModelID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	var m model.ProviderModel
	if err := h.DB.First(&m, input.ModelID).Error; err != nil {
		return h.Error(-24, "模型不存在")
	}
	return common.NewData(m)
}

// AddProviderModel 新增 ProviderModel
func (h *ProviderModelHandler) AddProviderModel(c echo.Context) error {
	input := &model.ProviderModel{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Name, validation.Required),
		validation.Field(&input.ProviderID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	// 唯一性检查（同一 Provider 下模型名不能重复）
	var exist int64
	h.DB.Model(&model.ProviderModel{}).Where("provider_id = ? AND name = ?", input.ProviderID, input.Name).Count(&exist)
	if exist > 0 {
		return h.Error(-12, "该 Provider 下已存在同名模型")
	}

	if err := h.DB.Create(input).Error; err != nil {
		return h.Error(-21, err.Error())
	}

	return common.NewData(input)
}

// UpdateProviderModel 更新 ProviderModel
func (h *ProviderModelHandler) UpdateProviderModel(c echo.Context) error {
	input, err := h.GetJSON(c)
	if err != nil {
		return err
	}

	// 提取主键
	modelID := input.Get("modelId")
	if !modelID.Exists() || modelID.Uint() == 0 {
		return h.Error(-23, "modelId is required")
	}

	// 逐字段提取，构建更新 map
	newState := map[string]any{}

	if input.Get("providerId").Exists() {
		newState["provider_id"] = input.Get("providerId").Uint()
	}
	if input.Get("name").Exists() {
		newState["name"] = input.Get("name").String()
	}
	if input.Get("displayName").Exists() {
		newState["display_name"] = input.Get("displayName").String()
	}
	if input.Get("description").Exists() {
		newState["description"] = input.Get("description").String()
	}
	if input.Get("maxContextTokens").Exists() {
		newState["max_context_tokens"] = input.Get("maxContextTokens").Int()
	}
	if input.Get("maxOutputTokens").Exists() {
		newState["max_output_tokens"] = input.Get("maxOutputTokens").Int()
	}
	if input.Get("inputPrice").Exists() {
		newState["input_price"] = input.Get("inputPrice").Float()
	}
	if input.Get("outputPrice").Exists() {
		newState["output_price"] = input.Get("outputPrice").Float()
	}
	if input.Get("tpm").Exists() {
		newState["tpm"] = input.Get("tpm").Int()
	}
	if input.Get("qpm").Exists() {
		newState["qpm"] = input.Get("qpm").Int()
	}
	if input.Get("isActive").Exists() {
		newState["is_active"] = input.Get("isActive").Bool()
	}

	// 空更新检查
	if len(newState) == 0 {
		return h.Success()
	}

	// 执行更新
	if err := h.DB.Model(&model.ProviderModel{}).Where("model_id = ?", modelID.Uint()).Updates(newState).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	return h.Success()
}

// RemoveProviderModel 删除 ProviderModel
func (h *ProviderModelHandler) RemoveProviderModel(c echo.Context) error {
	input := &struct {
		ModelID int64 `json:"modelId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.ModelID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	// 级联删除：先删关联的下游模型引用与路由
	h.DB.Where("upstream_model_id = ?", input.ModelID).Delete(&model.UserModel{})
	h.DB.Where("provider_model_id = ?", input.ModelID).Delete(&model.UserModelRouter{})

	if err := h.DB.Delete(&model.ProviderModel{}, input.ModelID).Error; err != nil {
		return h.Error(-23, err.Error())
	}

	return h.Success()
}
