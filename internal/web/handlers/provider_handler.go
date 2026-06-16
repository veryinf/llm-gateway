package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
)

type ProviderHandler struct {
	common.BaseHandler
	ProviderSvc *service.ProviderService
}

func (h *ProviderHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/providers/search", h.SearchProviders)
	g.POST("/providers/fetch", h.FetchProvider)
	g.POST("/providers/add", h.AddProvider)
	g.POST("/providers/update", h.UpdateProvider)
	g.POST("/providers/remove", h.RemoveProvider)
}

// SearchProviders 搜索 Provider 列表
func (h *ProviderHandler) SearchProviders(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}

	query := h.DB.Model(&model.Provider{}).Order("provider_id DESC")

	// 关键词搜索
	if input.Kw != "" {
		kw := "%" + input.Kw + "%"
		query = query.Where("title LIKE ? OR base_url LIKE ?", kw, kw)
	}

	// 过滤条件
	for _, filter := range input.Filters {
		switch filter.Field {
		case "is_active":
			query = query.Where("is_active = ?", filter.Value)
		case "support_openai":
			query = query.Where("support_openai = ?", filter.Value)
		case "support_anthropic":
			query = query.Where("support_anthropic = ?", filter.Value)
		}
	}

	// 分页查询
	var count int64
	var providers []model.Provider
	if err := h.Pagination(&input.Pagination, query, &providers, &count); err != nil {
		return err
	}

	// 关联计数：每个 Provider 的上游模型数量
	type providerIDCount struct {
		ProviderID uint
		Count      int
	}
	var counts []providerIDCount
	h.DB.Model(&model.ProviderModel{}).Select("provider_id, count(*) as count").Group("provider_id").Scan(&counts)

	countMap := make(map[uint]int, len(counts))
	for _, c := range counts {
		countMap[c.ProviderID] = c.Count
	}

	type providerWithCount struct {
		model.Provider
		ModelCount int `json:"modelCount"`
	}
	result := make([]providerWithCount, len(providers))
	for i, p := range providers {
		result[i] = providerWithCount{Provider: p, ModelCount: countMap[p.ProviderID]}
	}

	return common.NewDataSet(result, count)
}

// FetchProvider 获取单条 Provider
func (h *ProviderHandler) FetchProvider(c echo.Context) error {
	input := &struct {
		ProviderID int64 `json:"providerId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.ProviderID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	var provider model.Provider
	if err := h.DB.First(&provider, input.ProviderID).Error; err != nil {
		return h.Error(-24, "Provider 不存在")
	}
	return common.NewData(provider)
}

// AddProvider 新增 Provider
func (h *ProviderHandler) AddProvider(c echo.Context) error {
	input := &model.Provider{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Title, validation.Required),
		validation.Field(&input.BaseURL, validation.Required),
		validation.Field(&input.APIKey, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	// 唯一性检查
	var exist int64
	h.DB.Model(&model.Provider{}).Where("title = ?", input.Title).Count(&exist)
	if exist > 0 {
		return h.Error(-12, "Provider 名称已存在")
	}

	// 必须支持至少一种协议
	if !input.SupportOpenAI && !input.SupportAnthropic {
		return h.Error(-11, "请至少支持一种协议（OpenAI 或 Anthropic）")
	}

	// 设置默认值
	if input.PreferredAPI == "" {
		input.PreferredAPI = "openai"
	}
	input.IsActive = true

	if err := h.DB.Create(input).Error; err != nil {
		return h.Error(-21, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return common.NewData(input)
}

// UpdateProvider 更新 Provider
func (h *ProviderHandler) UpdateProvider(c echo.Context) error {
	input, err := h.GetJSON(c)
	if err != nil {
		return err
	}

	// 提取主键
	providerID := input.Get("providerId")
	if !providerID.Exists() || providerID.Uint() == 0 {
		return h.Error(-23, "providerId is required")
	}

	// 逐字段提取，构建更新 map
	newState := map[string]any{}

	if input.Get("title").Exists() {
		title := input.Get("title").String()
		if err := validation.Validate(title, validation.Required); err != nil {
			return h.Error(-11, "标题不能为空")
		}
		newState["title"] = title
	}

	if input.Get("baseUrl").Exists() {
		baseURL := input.Get("baseUrl").String()
		if err := validation.Validate(baseURL, validation.Required); err != nil {
			return h.Error(-11, "Base URL 不能为空")
		}
		newState["base_url"] = baseURL
	}

	if input.Get("apiKey").Exists() {
		newState["api_key"] = input.Get("apiKey").String()
	}

	if input.Get("supportOpenai").Exists() {
		newState["support_openai"] = input.Get("supportOpenai").Bool()
	}

	if input.Get("openaiBaseUrl").Exists() {
		newState["openai_base_url"] = input.Get("openaiBaseUrl").String()
	}

	if input.Get("supportAnthropic").Exists() {
		newState["support_anthropic"] = input.Get("supportAnthropic").Bool()
	}

	if input.Get("anthropicBaseUrl").Exists() {
		newState["anthropic_base_url"] = input.Get("anthropicBaseUrl").String()
	}

	if input.Get("preferredApi").Exists() {
		preferredAPI := input.Get("preferredApi").String()
		if err := validation.Validate(preferredAPI,
			validation.In("openai", "anthropic"),
		); err != nil {
			return h.Error(-11, err.Error())
		}
		newState["preferred_api"] = preferredAPI
	}

	if input.Get("isActive").Exists() {
		newState["is_active"] = input.Get("isActive").Bool()
	}

	// 校验：更新时也必须支持至少一种协议
	if input.Get("supportOpenai").Exists() || input.Get("supportAnthropic").Exists() {
		sOpenAI := input.Get("supportOpenai").Bool()
		sAnthropic := input.Get("supportAnthropic").Bool()
		// 如果只传了一个，需要用已有值补全
		if !input.Get("supportOpenai").Exists() || !input.Get("supportAnthropic").Exists() {
			var existing model.Provider
			if err := h.DB.First(&existing, providerID.Uint()).Error; err != nil {
				return h.Error(-24, "Provider 不存在")
			}
			if !input.Get("supportOpenai").Exists() {
				sOpenAI = existing.SupportOpenAI
			}
			if !input.Get("supportAnthropic").Exists() {
				sAnthropic = existing.SupportAnthropic
			}
		}
		if !sOpenAI && !sAnthropic {
			return h.Error(-11, "请至少支持一种协议（OpenAI 或 Anthropic）")
		}
	}

	// 空更新检查
	if len(newState) == 0 {
		return h.Success()
	}

	// 执行更新
	if err := h.DB.Model(&model.Provider{}).Where("provider_id = ?", providerID.Uint()).Updates(newState).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return h.Success()
}

// RemoveProvider 删除 Provider
func (h *ProviderHandler) RemoveProvider(c echo.Context) error {
	input := &struct {
		ProviderID int64 `json:"providerId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.ProviderID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	// 级联删除：先删关联的下游模型引用
	var modelIDs []uint
	h.DB.Model(&model.ProviderModel{}).Where("provider_id = ?", input.ProviderID).Pluck("model_id", &modelIDs)
	if len(modelIDs) > 0 {
		h.DB.Where("upstream_model_id IN ?", modelIDs).Delete(&model.DownstreamModel{})
		h.DB.Where("provider_id = ?", input.ProviderID).Delete(&model.ProviderModel{})
	}

	if err := h.DB.Delete(&model.Provider{}, input.ProviderID).Error; err != nil {
		return h.Error(-23, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return h.Success()
}

type fetchedModel struct {
	ID string `json:"id"`
}

// FetchProviderModels 获取上游 Provider 的模型列表
func (h *ProviderHandler) FetchProviderModels(c echo.Context) error {
	input := &struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
		APIType string `json:"apiType"`
	}{}
	if err := c.Bind(input); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if input.BaseURL == "" {
		return h.Error(-11, "Base URL 不能为空")
	}

	// Anthropic 没有标准的 /v1/models 端点
	if input.APIType == "anthropic" {
		return common.NewData([]fetchedModel{})
	}

	baseURL := input.BaseURL
	for len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	modelsURL := baseURL + "/v1/models"

	httpReq, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return h.Error(-21, "创建请求失败")
	}
	if input.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+input.APIKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return h.Error(-22, fmt.Sprintf("请求上游失败: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return h.Error(-23, fmt.Sprintf("上游返回 %d: %s", resp.StatusCode, truncateStr(string(body), 512)))
	}

	var result struct {
		Data []fetchedModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return h.Error(-24, "解析响应失败")
	}

	return common.NewData(result.Data)
}

// BatchImportModels 批量导入模型
func (h *ProviderHandler) BatchImportModels(c echo.Context) error {
	input := &struct {
		ProviderID uint     `json:"providerId"`
		ModelNames []string `json:"modelNames"`
	}{}
	if err := c.Bind(input); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if input.ProviderID == 0 || len(input.ModelNames) == 0 {
		return h.Error(-11, "参数不完整")
	}

	// 检查 Provider 是否存在
	var p model.Provider
	if err := h.DB.First(&p, input.ProviderID).Error; err != nil {
		return h.Error(-24, "Provider 不存在")
	}

	// 获取已存在的模型名称
	var existingNames []string
	h.DB.Model(&model.ProviderModel{}).Where("provider_id = ?", input.ProviderID).Pluck("name", &existingNames)
	existingSet := make(map[string]bool, len(existingNames))
	for _, name := range existingNames {
		existingSet[name] = true
	}

	// 批量创建
	created := 0
	skipped := 0
	for _, name := range input.ModelNames {
		if existingSet[name] {
			skipped++
			continue
		}
		m := model.ProviderModel{
			ProviderID: input.ProviderID,
			Name:       name,
			APIType:    model.APIType(p.PreferredAPI),
			IsActive:   true,
		}
		if err := h.DB.Create(&m).Error; err != nil {
			slog.Warn("create model", "name", name, "error", err)
			continue
		}
		created++
	}

	_ = h.ProviderSvc.ReloadProviders()
	return common.NewData(map[string]int{"created": created, "skipped": skipped})
}
