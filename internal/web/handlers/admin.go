package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	common.BaseHandler
	ProviderSvc *service.ProviderService
}

func (h *AdminHandler) RegisterRoutes(g *echo.Group) {
	// User CRUD
	g.GET("/users", h.ListUsers)
	g.POST("/users", h.CreateUser)
	g.PUT("/users/:id", h.UpdateUser)
	g.DELETE("/users/:id", h.DeleteUser)

	// API Key CRUD
	g.GET("/users/:id/api-keys", h.ListAPIKeys)
	g.POST("/users/:id/api-keys", h.CreateAPIKey)
	g.DELETE("/users/:id/api-keys/:kid", h.DeleteAPIKey)

	g.GET("/api-keys", h.ListAllAPIKeys)
	g.DELETE("/api-keys/:id", h.DeleteAPIKeyByID)
	g.PUT("/api-keys/:id/toggle", h.ToggleAPIKey)

	// AKSK
	g.POST("/users/:id/aksk", h.GenerateAKSK)
	g.GET("/users/:id/aksk", h.GetAKSK)

	// Provider CRUD
	g.GET("/providers", h.ListProviders)
	g.POST("/providers", h.CreateProvider)
	g.PUT("/providers/:id", h.UpdateProvider)
	g.DELETE("/providers/:id", h.DeleteProvider)
	g.PUT("/providers/:id/toggle", h.ToggleProvider)
	g.POST("/providers/fetch-models", h.FetchProviderModels)
	g.POST("/providers/batch-import-models", h.BatchImportModels)

	// Model CRUD
	g.GET("/models", h.ListModels)
	g.POST("/models", h.CreateModel)
	g.PUT("/models/:id", h.UpdateModel)
	g.DELETE("/models/:id", h.DeleteModel)

	// Downstream Model CRUD
	g.GET("/downstream-models", h.ListDownstreamModels)
	g.POST("/downstream-models", h.CreateDownstreamModel)
	g.PUT("/downstream-models/:id", h.UpdateDownstreamModel)
	g.DELETE("/downstream-models/:id", h.DeleteDownstreamModel)

	// Config
	g.GET("/configs", h.ListConfigs)
	g.PUT("/configs", h.UpdateConfig)
}

// ======================== User Management ========================

type createUserRequest struct {
	Username   string     `json:"username"`
	Password   string     `json:"password"`
	Name       string     `json:"name"`
	Phone      string     `json:"phone"`
	Department string     `json:"department"`
	Role       model.Role `json:"role"`
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	var users []model.User
	if err := h.DB.Find(&users).Error; err != nil {
		return h.Error(-20, err.Error())
	}

	type userWithCount struct {
		model.User
		APIKeyCount int `json:"api_key_count"`
	}

	var counts []struct {
		UserID uint
		Count  int
	}
	h.DB.Model(&model.APIKey{}).Select("user_id, count(*) as count").Group("user_id").Scan(&counts)

	countMap := make(map[uint]int, len(counts))
	for _, c := range counts {
		countMap[c.UserID] = c.Count
	}

	result := make([]userWithCount, len(users))
	for i, u := range users {
		result[i] = userWithCount{User: u, APIKeyCount: countMap[u.UID]}
	}

	return c.JSON(200, common.NewDataSet(result, int64(len(result))))
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	var req createUserRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if req.Username == "" || req.Password == "" {
		return h.Error(-11, "用户名和密码不能为空")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return h.Error(-21, "密码加密失败")
	}

	user := &model.User{
		Username:   req.Username,
		Password:   string(hash),
		Name:       req.Name,
		Phone:      req.Phone,
		Department: req.Department,
		Role:       req.Role,
		Status:     "active",
	}

	if err := h.DB.Create(user).Error; err != nil {
		return h.Error(-21, err.Error())
	}
	return c.JSON(200, common.NewData(user))
}

func (h *AdminHandler) UpdateUser(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的用户 ID")
	}

	var updates map[string]interface{}
	if err := c.Bind(&updates); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	delete(updates, "id")
	delete(updates, "password_hash")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	if password, ok := updates["password"].(string); ok && password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return h.Error(-22, "密码加密失败")
		}
		updates["password_hash"] = string(hash)
	}
	delete(updates, "password")

	if len(updates) == 0 {
		return c.JSON(200, common.NewResponse(0, "ok"))
	}

	if err := h.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return h.Error(-22, err.Error())
	}
	return c.JSON(200, common.NewResponse(0, "ok"))
}

func (h *AdminHandler) DeleteUser(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的用户 ID")
	}

	if err := h.DB.Delete(&model.User{}, id).Error; err != nil {
		return h.Error(-23, err.Error())
	}
	return c.JSON(200, common.NewResponse(0, "ok"))
}

// ======================== API Key Management ========================

type createAPIKeyRequest struct {
	Name         string `json:"name"`
	QuotaLimit   int64  `json:"quota_limit"`
	RateLimitQPM int    `json:"rate_limit_qpm"`
}

type createAPIKeyResponse struct {
	APIKey *model.APIKey `json:"api_key"`
	RawKey string        `json:"raw_key"`
}

func (h *AdminHandler) CreateAPIKey(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的用户 ID")
	}

	var req createAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if req.Name == "" {
		return h.Error(-11, "名称不能为空")
	}

	keyRecord, rawKey, err := service.GenerateAPIKeyRecord(h.DB, userID, req.Name, req.QuotaLimit, req.RateLimitQPM)
	if err != nil {
		return h.Error(-21, err.Error())
	}
	return c.JSON(200, common.NewData(createAPIKeyResponse{APIKey: keyRecord, RawKey: rawKey}))
}

func (h *AdminHandler) ListAPIKeys(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的用户 ID")
	}

	var keys []model.APIKey
	if err := h.DB.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	return c.JSON(200, common.NewDataSet(keys, int64(len(keys))))
}

func (h *AdminHandler) DeleteAPIKey(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的用户 ID")
	}

	kid, err := parseUintParam(c, "kid")
	if err != nil {
		return h.Error(-23, "无效的 API Key ID")
	}

	var key model.APIKey
	if err := h.DB.Where("id = ? AND user_id = ?", kid, userID).First(&key).Error; err != nil {
		return h.Error(-24, "API Key 不存在")
	}

	if err := h.DB.Delete(&model.APIKey{}, kid).Error; err != nil {
		return h.Error(-23, err.Error())
	}
	return c.JSON(200, common.NewResponse(0, "ok"))
}

func (h *AdminHandler) ListAllAPIKeys(c echo.Context) error {
	var keys []model.APIKey
	if err := h.DB.Order("created_at desc").Find(&keys).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	return c.JSON(200, common.NewDataSet(keys, int64(len(keys))))
}

func (h *AdminHandler) DeleteAPIKeyByID(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 API Key ID")
	}
	if err := h.DB.Delete(&model.APIKey{}, id).Error; err != nil {
		return h.Error(-23, err.Error())
	}
	return c.JSON(200, common.NewResponse(0, "ok"))
}

func (h *AdminHandler) ToggleAPIKey(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 API Key ID")
	}

	var key model.APIKey
	if err := h.DB.First(&key, id).Error; err != nil {
		return h.Error(-24, "API Key 不存在")
	}

	if err := h.DB.Model(&key).Update("is_active", !key.IsActive).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	return c.JSON(200, common.NewData(map[string]interface{}{"is_active": !key.IsActive}))
}

// ======================== AKSK Management ========================

type akskResponse struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

func (h *AdminHandler) GenerateAKSK(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的用户 ID")
	}

	accessKey, secretKey, err := service.GenerateAKSK(h.DB, id)
	if err != nil {
		return h.Error(-21, err.Error())
	}

	return c.JSON(200, common.NewData(&akskResponse{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}))
}

func (h *AdminHandler) GetAKSK(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的用户 ID")
	}

	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		return h.Error(-24, "用户不存在")
	}

	return c.JSON(200, common.NewData(map[string]string{
		"access_key": user.AccessKey,
	}))
}

// ======================== Provider Management ========================

type createProviderWithModelsRequest struct {
	model.Provider
	Models []model.Model `json:"models"`
}

func (h *AdminHandler) ListProviders(c echo.Context) error {
	var providers []model.Provider
	if err := h.DB.Find(&providers).Error; err != nil {
		return h.Error(-20, err.Error())
	}

	type providerWithCount struct {
		model.Provider
		ModelCount int `json:"model_count"`
	}

	var counts []struct {
		ProviderID uint
		Count      int
	}
	h.DB.Model(&model.Model{}).Select("provider_id, count(*) as count").Group("provider_id").Scan(&counts)

	countMap := make(map[uint]int, len(counts))
	for _, cnt := range counts {
		countMap[cnt.ProviderID] = cnt.Count
	}

	result := make([]providerWithCount, len(providers))
	for i, p := range providers {
		result[i] = providerWithCount{Provider: p, ModelCount: countMap[p.ID]}
	}

	return c.JSON(200, common.NewDataSet(result, int64(len(result))))
}

func (h *AdminHandler) CreateProvider(c echo.Context) error {
	var req createProviderWithModelsRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	p := req.Provider
	if err := h.DB.Create(&p).Error; err != nil {
		return h.Error(-21, err.Error())
	}

	// Batch create associated models
	if len(req.Models) > 0 {
		for i := range req.Models {
			req.Models[i].ProviderID = p.ID
			req.Models[i].ID = 0
		}
		if err := h.DB.Create(&req.Models).Error; err != nil {
			slog.Warn("batch create models", "error", err)
		}
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewData(p))
}

type updateProviderRequest struct {
	Provider model.Provider `json:"provider"`
	Models   *[]model.Model `json:"models,omitempty"`
}

func (h *AdminHandler) UpdateProvider(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Provider ID")
	}

	var req updateProviderRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	// Load existing provider to preserve api_key if empty
	var existing model.Provider
	if err := h.DB.First(&existing, id).Error; err != nil {
		return h.Error(-24, "Provider 不存在")
	}

	p := req.Provider
	p.ID = id
	p.CreatedAt = existing.CreatedAt

	// api_key: empty string means "don't modify"
	if p.APIKey == "" {
		p.APIKey = existing.APIKey
	}

	// Use Save for full update (handles GORM column naming correctly)
	if err := h.DB.Save(&p).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	// Reconcile models if provided
	if req.Models != nil {
		h.reconcileModels(id, *req.Models)
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
}

// reconcileModels syncs upstream models for a provider: create new, update existing, delete removed.
func (h *AdminHandler) reconcileModels(providerID uint, incoming []model.Model) {
	var existing []model.Model
	h.DB.Where("provider_id = ?", providerID).Find(&existing)

	existingMap := make(map[uint]model.Model, len(existing))
	for _, m := range existing {
		existingMap[m.ID] = m
	}

	incomingIDs := make(map[uint]bool)
	for _, m := range incoming {
		if m.ID > 0 {
			incomingIDs[m.ID] = true
			// Update existing
			h.DB.Model(&model.Model{}).Where("id = ? AND provider_id = ?", m.ID, providerID).Updates(map[string]interface{}{
				"name":               m.Name,
				"api_type":           m.APIType,
				"display_name":       m.DisplayName,
				"description":        m.Description,
				"max_context_tokens": m.MaxContextTokens,
				"max_output_tokens":  m.MaxOutputTokens,
				"input_price":        m.InputPrice,
				"output_price":       m.OutputPrice,
				"tpm":                m.TPM,
				"qpm":                m.QPM,
				"is_chat":            m.IsChat,
				"is_completion":      m.IsCompletion,
				"is_vision":          m.IsVision,
				"is_embedding":       m.IsEmbedding,
				"is_active":          m.IsActive,
			})
		} else {
			// Create new
			m.ProviderID = providerID
			m.ID = 0
			h.DB.Create(&m)
		}
	}

	// Delete models not in incoming list
	for _, existing := range existing {
		if !incomingIDs[existing.ID] {
			h.DB.Where("upstream_model_id = ?", existing.ID).Delete(&model.DownstreamModel{})
			h.DB.Delete(&model.Model{}, existing.ID)
		}
	}
}

func (h *AdminHandler) ToggleProvider(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Provider ID")
	}

	var p model.Provider
	if err := h.DB.First(&p, id).Error; err != nil {
		return h.Error(-24, "Provider 不存在")
	}

	if err := h.DB.Model(&p).Update("is_active", !p.IsActive).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewData(map[string]interface{}{"is_active": !p.IsActive}))
}

func (h *AdminHandler) DeleteProvider(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Provider ID")
	}

	// Cascade: delete downstream models referencing this provider's upstream models
	var modelIDs []uint
	h.DB.Model(&model.Model{}).Where("provider_id = ?", id).Pluck("id", &modelIDs)
	if len(modelIDs) > 0 {
		h.DB.Where("upstream_model_id IN ?", modelIDs).Delete(&model.DownstreamModel{})
		h.DB.Where("provider_id = ?", id).Delete(&model.Model{})
	}

	if err := h.DB.Delete(&model.Provider{}, id).Error; err != nil {
		return h.Error(-23, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
}

// ======================== Fetch Provider Models ========================

type fetchModelsRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	APIType string `json:"api_type"` // "openai" or "anthropic"
}

type fetchedModel struct {
	ID string `json:"id"`
}

func (h *AdminHandler) FetchProviderModels(c echo.Context) error {
	var req fetchModelsRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if req.BaseURL == "" {
		return h.Error(-11, "Base URL 不能为空")
	}

	// Anthropic doesn't have a standard /v1/models endpoint
	if req.APIType == "anthropic" {
		return c.JSON(200, common.NewData([]fetchedModel{}))
	}

	baseURL := req.BaseURL
	for len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	modelsURL := baseURL + "/v1/models"

	httpReq, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return h.Error(-21, "创建请求失败")
	}
	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
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

	return c.JSON(200, common.NewData(result.Data))
}

type batchImportModelsRequest struct {
	ProviderID uint     `json:"provider_id"`
	ModelNames []string `json:"model_names"`
}

func (h *AdminHandler) BatchImportModels(c echo.Context) error {
	var req batchImportModelsRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if req.ProviderID == 0 || len(req.ModelNames) == 0 {
		return h.Error(-11, "参数不完整")
	}

	// Check provider exists
	var p model.Provider
	if err := h.DB.First(&p, req.ProviderID).Error; err != nil {
		return h.Error(-24, "Provider 不存在")
	}

	// Get existing model names for this provider
	var existing []string
	h.DB.Model(&model.Model{}).Where("provider_id = ?", req.ProviderID).Pluck("name", &existing)
	existingSet := make(map[string]bool, len(existing))
	for _, name := range existing {
		existingSet[name] = true
	}

	var created int
	for _, name := range req.ModelNames {
		if existingSet[name] {
			continue
		}
		m := model.Model{
			ProviderID: req.ProviderID,
			Name:       name,
			IsActive:   true,
		}
		if err := h.DB.Create(&m).Error; err != nil {
			continue
		}
		created++
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewData(map[string]interface{}{"created": created, "skipped": len(req.ModelNames) - created}))
}

// ======================== Model Routing Management ========================

func (h *AdminHandler) ListModels(c echo.Context) error {
	query := h.DB.Preload("Provider")
	if providerID := c.QueryParam("provider_id"); providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	var models []model.Model
	if err := query.Find(&models).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	return c.JSON(200, common.NewDataSet(models, int64(len(models))))
}

func (h *AdminHandler) CreateModel(c echo.Context) error {
	var m model.Model
	if err := c.Bind(&m); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	if err := h.DB.Create(&m).Error; err != nil {
		return h.Error(-21, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewData(m))
}

func (h *AdminHandler) UpdateModel(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Model ID")
	}

	var m model.Model
	if err := c.Bind(&m); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	// Load existing to preserve provider_id and timestamps
	var existing model.Model
	if err := h.DB.First(&existing, id).Error; err != nil {
		return h.Error(-24, "Model 不存在")
	}

	m.ID = id
	m.ProviderID = existing.ProviderID
	m.CreatedAt = existing.CreatedAt

	if err := h.DB.Save(&m).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
}

func (h *AdminHandler) DeleteModel(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Model ID")
	}

	if err := h.DB.Delete(&model.Model{}, id).Error; err != nil {
		return h.Error(-23, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
}

// ======================== Downstream Model Management ========================

func (h *AdminHandler) ListDownstreamModels(c echo.Context) error {
	query := h.DB.Preload("UpstreamModel")
	if upstreamModelID := c.QueryParam("upstream_model_id"); upstreamModelID != "" {
		query = query.Where("upstream_model_id = ?", upstreamModelID)
	}
	var models []model.DownstreamModel
	if err := query.Find(&models).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	return c.JSON(200, common.NewDataSet(models, int64(len(models))))
}

func (h *AdminHandler) CreateDownstreamModel(c echo.Context) error {
	var m model.DownstreamModel
	if err := c.Bind(&m); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	if err := h.DB.Create(&m).Error; err != nil {
		return h.Error(-21, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewData(m))
}

func (h *AdminHandler) UpdateDownstreamModel(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Downstream Model ID")
	}

	var updates map[string]interface{}
	if err := c.Bind(&updates); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	if err := h.DB.Model(&model.DownstreamModel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
}

func (h *AdminHandler) DeleteDownstreamModel(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Downstream Model ID")
	}

	if err := h.DB.Delete(&model.DownstreamModel{}, id).Error; err != nil {
		return h.Error(-23, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
}

// ======================== Config Management ========================

type configUpdateRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (h *AdminHandler) ListConfigs(c echo.Context) error {
	var configs []model.Config
	if err := h.DB.Find(&configs).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	return c.JSON(200, common.NewDataSet(configs, int64(len(configs))))
}

func (h *AdminHandler) UpdateConfig(c echo.Context) error {
	var req configUpdateRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-11, "请求参数错误")
	}
	if req.Key == "" {
		return h.Error(-11, "key 不能为空")
	}

	var config model.Config
	if err := h.DB.Where("`key` = ?", req.Key).First(&config).Error; err != nil {
		config = model.Config{Key: req.Key, Value: req.Value}
		if err := h.DB.Create(&config).Error; err != nil {
			return h.Error(-21, err.Error())
		}
		return c.JSON(200, common.NewData(config))
	}

	config.Value = req.Value
	if err := h.DB.Save(&config).Error; err != nil {
		return h.Error(-22, err.Error())
	}
	return c.JSON(200, common.NewData(config))
}

// ======================== Helpers ========================

func parseUintParam(c echo.Context, key string) (uint, error) {
	val, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}
