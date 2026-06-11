package handlers

import (
	"strconv"

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

	// Model CRUD
	g.GET("/models", h.ListModels)
	g.POST("/models", h.CreateModel)
	g.PUT("/models/:id", h.UpdateModel)

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
		result[i] = userWithCount{User: u, APIKeyCount: countMap[u.ID]}
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
		Username:     req.Username,
		PasswordHash: string(hash),
		Name:         req.Name,
		Phone:        req.Phone,
		Department:   req.Department,
		Role:         req.Role,
		IsActive:     true,
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

func (h *AdminHandler) ListProviders(c echo.Context) error {
	var providers []model.Provider
	if err := h.DB.Find(&providers).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	return c.JSON(200, common.NewDataSet(providers, int64(len(providers))))
}

func (h *AdminHandler) CreateProvider(c echo.Context) error {
	var p model.Provider
	if err := c.Bind(&p); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	if err := h.DB.Create(&p).Error; err != nil {
		return h.Error(-21, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewData(p))
}

func (h *AdminHandler) UpdateProvider(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.Error(-23, "无效的 Provider ID")
	}

	var updates map[string]interface{}
	if err := c.Bind(&updates); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	// 不允许通过更新接口修改的字段
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	// api_key 为空时保留原值（前端留空表示不修改）
	if v, ok := updates["api_key"]; ok {
		if s, ok := v.(string); ok && s == "" {
			delete(updates, "api_key")
		}
	}

	if len(updates) == 0 {
		return c.JSON(200, common.NewResponse(0, "ok"))
	}

	if err := h.DB.Model(&model.Provider{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return h.Error(-22, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
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

	if err := h.DB.Delete(&model.Provider{}, id).Error; err != nil {
		return h.Error(-23, err.Error())
	}

	_ = h.ProviderSvc.ReloadProviders()
	return c.JSON(200, common.NewResponse(0, "ok"))
}

// ======================== Model Routing Management ========================

func (h *AdminHandler) ListModels(c echo.Context) error {
	var models []model.Model
	if err := h.DB.Preload("Provider").Find(&models).Error; err != nil {
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

	var updates map[string]interface{}
	if err := c.Bind(&updates); err != nil {
		return h.Error(-11, "请求参数错误")
	}

	if err := h.DB.Model(&model.Model{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return h.Error(-22, err.Error())
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
