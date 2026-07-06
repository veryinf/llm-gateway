package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
)

type ProviderHandler struct {
	common.BaseHandler
}

func (h *ProviderHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/providers/search", h.SearchProviders)
	g.POST("/providers/fetch", h.FetchProvider)
	g.POST("/providers/add", h.AddProvider)
	g.POST("/providers/update", h.UpdateProvider)
	g.POST("/providers/remove", h.RemoveProvider)
	g.POST("/providers/fetch-models", h.FetchProviderModels)
	g.POST("/providers/test-model", h.TestProviderModel)
}

// SearchProviders 搜索 Provider 列表
func (h *ProviderHandler) SearchProviders(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}

	query := h.DB.Model(&model.Provider{}).Order("provider_id DESC")

	// 关键词搜索 — 转义 LIKE 通配符
	if pattern := input.EscapedKw(); pattern != "" {
		query = query.Where("title LIKE ? ESCAPE '\\' OR base_url LIKE ? ESCAPE '\\'", pattern, pattern)
	}

	// 过滤条件
	for _, filter := range input.Filters {
		switch filter.Field {
		case "is_active":
			if err := validation.Validate(filter.Value, validation.In(true, false)); err != nil {
				return h.Error(-11, "is_active 必须是布尔值")
			}
			query = query.Where("is_active = ?", filter.Value)
		case "support_openai":
			if err := validation.Validate(filter.Value, validation.In(true, false)); err != nil {
				return h.Error(-11, "support_openai 必须是布尔值")
			}
			query = query.Where("support_openai = ?", filter.Value)
		case "support_anthropic":
			if err := validation.Validate(filter.Value, validation.In(true, false)); err != nil {
				return h.Error(-11, "support_anthropic 必须是布尔值")
			}
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
	input, err := h.GetJSON(c)
	if err != nil {
		return err
	}

	// 构建 Provider 实体
	p := &model.Provider{
		Title:            input.Get("title").String(),
		BaseURL:          input.Get("baseUrl").String(),
		APIKey:           input.Get("apiKey").String(),
		SupportOpenai:    input.Get("supportOpenai").Bool(),
		OpenaiBaseURL:    input.Get("openaiBaseUrl").String(),
		SupportAnthropic: input.Get("supportAnthropic").Bool(),
		AnthropicBaseURL: input.Get("anthropicBaseUrl").String(),
		IsActive:         input.Get("isActive").Bool(),
		IsDefault:        input.Get("isDefault").Bool(),
	}

	if err := validation.ValidateStruct(p,
		validation.Field(&p.Title, validation.Required),
		validation.Field(&p.BaseURL, validation.Required),
		validation.Field(&p.APIKey, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	// 唯一性检查
	var exist int64
	h.DB.Model(&model.Provider{}).Where("title = ?", p.Title).Count(&exist)
	if exist > 0 {
		return h.Error(-12, "Provider 名称已存在")
	}

	// 必须支持至少一种协议
	if !p.SupportOpenai && !p.SupportAnthropic {
		return h.Error(-11, "请至少支持一种协议（OpenAI 或 Anthropic）")
	}

	p.IsActive = true

	// 提取 models 字段
	var modelNames []string
	if modelsArr := input.Get("models"); modelsArr.Exists() && modelsArr.IsArray() {
		for _, v := range modelsArr.Array() {
			name := v.String()
			if name != "" {
				modelNames = append(modelNames, name)
			}
		}
	}

	if err := h.DB.Create(p).Error; err != nil {
		return h.Error(-21, err.Error())
	}

	// 批量创建 ProviderModel
	if len(modelNames) > 0 {
		models := make([]model.ProviderModel, 0, len(modelNames))
		for _, name := range modelNames {
			models = append(models, model.ProviderModel{
				ProviderID: p.ProviderID,
				Name:       name,
				IsActive:   true,
			})
		}
		if err := h.DB.Create(&models).Error; err != nil {
			return h.Error(-21, "创建模型失败: "+err.Error())
		}
	}

	return common.NewData(p)
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

	if input.Get("apiKey").Exists() && input.Get("apiKey").String() != "" {
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

	if input.Get("isActive").Exists() {
		newState["is_active"] = input.Get("isActive").Bool()
	}

	if input.Get("isDefault").Exists() {
		newState["is_default"] = input.Get("isDefault").Bool()
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
				sOpenAI = existing.SupportOpenai
			}
			if !input.Get("supportAnthropic").Exists() {
				sAnthropic = existing.SupportAnthropic
			}
		}
		if !sOpenAI && !sAnthropic {
			return h.Error(-11, "请至少支持一种协议（OpenAI 或 Anthropic）")
		}
	}

	// 空更新检查（models 字段单独处理）
	if len(newState) == 0 && !input.Get("models").Exists() {
		return h.Success()
	}

	// 执行 Provider 字段更新
	if len(newState) > 0 {
		if err := h.DB.Model(&model.Provider{}).Where("provider_id = ?", providerID.Uint()).Updates(newState).Error; err != nil {
			return h.Error(-22, err.Error())
		}
	}

	// 同步 models（reconcile）
	if modelsArr := input.Get("models"); modelsArr.Exists() && modelsArr.IsArray() {
		var newNames []string
		for _, v := range modelsArr.Array() {
			name := v.String()
			if name != "" {
				newNames = append(newNames, name)
			}
		}

		// 查询现有模型
		var existing []model.ProviderModel
		h.DB.Where("provider_id = ?", providerID.Uint()).Find(&existing)

		existingMap := make(map[string]uint, len(existing))
		for _, m := range existing {
			existingMap[m.Name] = m.ModelID
		}

		newSet := make(map[string]bool, len(newNames))
		for _, name := range newNames {
			newSet[name] = true
		}

		// 删除：DB 有但新列表没有的
		for name, modelID := range existingMap {
			if !newSet[name] {
				h.DB.Where("upstream_model_id = ?", modelID).Delete(&model.UserModel{})
				h.DB.Where("provider_model_id = ?", modelID).Delete(&model.UserModelRouter{})
				h.DB.Delete(&model.ProviderModel{}, modelID)
			}
		}

		// 新增：新列表有但 DB 没有的
		for _, name := range newNames {
			if _, exists := existingMap[name]; !exists {
				h.DB.Create(&model.ProviderModel{
					ProviderID: uint(providerID.Uint()),
					Name:       name,
					IsActive:   true,
				})
			}
		}
	}

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
		h.DB.Where("upstream_model_id IN ?", modelIDs).Delete(&model.UserModel{})
		h.DB.Where("provider_model_id IN ?", modelIDs).Delete(&model.UserModelRouter{})
		h.DB.Where("provider_id = ?", input.ProviderID).Delete(&model.ProviderModel{})
	}

	if err := h.DB.Delete(&model.Provider{}, input.ProviderID).Error; err != nil {
		return h.Error(-23, err.Error())
	}

	return h.Success()
}

// FetchProviderModels 获取上游 Provider 的模型列表
func (h *ProviderHandler) FetchProviderModels(c echo.Context) error {
	input := &struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}{}
	if err := c.Bind(input); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.BaseURL, validation.Required),
		validation.Field(&input.APIKey, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	p := &model.Provider{
		Title:         "_fetch",
		BaseURL:       input.BaseURL,
		APIKey:        input.APIKey,
		SupportOpenai: true,
	}
	adapter, err := provider.NewAdapter(p)
	if err != nil {
		return h.Error(-22, err.Error())
	}
	models, err := adapter.ListModels(c.Request().Context())
	if err != nil {
		return h.Error(-22, err.Error())
	}
	return common.NewDataSet(models, int64(len(models)))
}

// TestModelResult 模型测试结果
type TestModelResult struct {
	Success   bool   `json:"success"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
}

// TestProviderModel 测试指定 Provider 上的某个模型是否可用
func (h *ProviderHandler) TestProviderModel(c echo.Context) error {
	input := &struct {
		ProviderID uint   `json:"providerId"`
		ModelName  string `json:"modelName"`
	}{}
	if err := c.Bind(input); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.ProviderID, validation.Required),
		validation.Field(&input.ModelName, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	var p model.Provider
	if err := h.DB.First(&p, input.ProviderID).Error; err != nil {
		return h.Error(-24, "Provider 不存在")
	}

	apiType := model.APITypeOpenAI
	if !p.SupportOpenai && p.SupportAnthropic {
		apiType = model.APITypeAnthropic
	}

	payload := map[string]any{
		"model": input.ModelName,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 1,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return common.NewData(TestModelResult{Success: false, Error: "构建请求失败: " + err.Error()})
	}

	ctx := c.Request().Context()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "", bytes.NewReader(bodyBytes))
	if err != nil {
		return common.NewData(TestModelResult{Success: false, Error: "构建 HTTP 请求失败: " + err.Error()})
	}
	httpReq.Header.Set("Content-Type", "application/json")

	llmReq := &provider.LLMRequest{
		APIType: apiType,
		Request: httpReq,
		Model:   input.ModelName,
		Stream:  false,
	}

	adapter, err := provider.NewAdapter(&p)
	if err != nil {
		return common.NewData(TestModelResult{Success: false, Error: "创建适配器失败: " + err.Error()})
	}

	start := time.Now()
	var testErr error
	if apiType == model.APITypeOpenAI {
		_, testErr = adapter.ChatCompletion(ctx, llmReq)
	} else {
		_, testErr = adapter.Message(ctx, llmReq)
	}
	latency := time.Since(start).Milliseconds()

	if testErr != nil {
		return common.NewData(TestModelResult{Success: false, LatencyMs: latency, Error: testErr.Error()})
	}
	return common.NewData(TestModelResult{Success: true, LatencyMs: latency})
}
