package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"llm-gateway/internal/model"
	"llm-gateway/internal/service"
	"llm-gateway/pkg/apierror"
	"llm-gateway/pkg/response"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type AdminHandler struct {
	userSvc     *service.UserService
	apiKeySvc   *service.APIKeyService
	providerSvc *service.ProviderService
	db          *gorm.DB
}

func NewAdminHandler(
	userSvc *service.UserService,
	apiKeySvc *service.APIKeyService,
	providerSvc *service.ProviderService,
	db *gorm.DB,
) *AdminHandler {
	return &AdminHandler{
		userSvc:     userSvc,
		apiKeySvc:   apiKeySvc,
		providerSvc: providerSvc,
		db:          db,
	}
}

// ======================== Auth ========================

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *AdminHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}
	if req.Username == "" || req.Password == "" {
		return response.Error(c, apierror.BadRequest("username and password are required"))
	}

	token, err := h.userSvc.Login(req.Username, req.Password)
	if err != nil {
		return response.Error(c, apierror.Unauthorized("invalid credentials"))
	}

	return response.Success(c, loginResponse{Token: token})
}

// ======================== User Management ========================

type createUserRequest struct {
	Username   string     `json:"username"`
	Password   string     `json:"password"`
	Email      string     `json:"email"`
	Department string     `json:"department"`
	Role       model.Role `json:"role"`
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	var users []model.User
	if err := h.db.Find(&users).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, users)
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	var req createUserRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}
	if req.Username == "" || req.Password == "" {
		return response.Error(c, apierror.BadRequest("username and password are required"))
	}

	user, err := h.userSvc.CreateUser(req.Username, req.Password, req.Email, req.Department, req.Role)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, user)
}

func (h *AdminHandler) UpdateUser(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid user id"))
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(c.Request().Body).Decode(&updates); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, nil)
}

func (h *AdminHandler) DeleteUser(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid user id"))
	}

	if err := h.db.Delete(&model.User{}, id).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, nil)
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
		return response.Error(c, apierror.BadRequest("invalid user id"))
	}

	var req createAPIKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}
	if req.Name == "" {
		return response.Error(c, apierror.BadRequest("name is required"))
	}

	keyRecord, rawKey, err := h.apiKeySvc.CreateAPIKey(userID, req.Name, req.QuotaLimit, req.RateLimitQPM)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, createAPIKeyResponse{APIKey: keyRecord, RawKey: rawKey})
}

func (h *AdminHandler) ListAPIKeys(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid user id"))
	}

	var keys []model.APIKey
	if err := h.db.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, keys)
}

func (h *AdminHandler) DeleteAPIKey(c echo.Context) error {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid user id"))
	}

	kid, err := parseUintParam(c, "kid")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid api key id"))
	}

	if err := h.apiKeySvc.DeleteAPIKey(kid, userID); err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, nil)
}

// ListAllAPIKeys returns all API keys across all users.
func (h *AdminHandler) ListAllAPIKeys(c echo.Context) error {
	var keys []model.APIKey
	if err := h.db.Order("created_at desc").Find(&keys).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, keys)
}

// DeleteAPIKeyByID deletes an API key by ID without user ownership check.
func (h *AdminHandler) DeleteAPIKeyByID(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid api key id"))
	}
	if err := h.db.Delete(&model.APIKey{}, id).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, nil)
}

// ======================== Provider Management ========================

func (h *AdminHandler) ListProviders(c echo.Context) error {
	var providers []model.Provider
	if err := h.db.Find(&providers).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, providers)
}

func (h *AdminHandler) CreateProvider(c echo.Context) error {
	var provider model.Provider
	if err := json.NewDecoder(c.Request().Body).Decode(&provider); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}

	if err := h.db.Create(&provider).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	_ = h.providerSvc.ReloadProviders()

	return response.Success(c, provider)
}

func (h *AdminHandler) UpdateProvider(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid provider id"))
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(c.Request().Body).Decode(&updates); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}

	if err := h.db.Model(&model.Provider{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	_ = h.providerSvc.ReloadProviders()

	return response.Success(c, nil)
}

func (h *AdminHandler) ToggleProvider(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid provider id"))
	}

	var provider model.Provider
	if err := h.db.First(&provider, id).Error; err != nil {
		return response.Error(c, apierror.NotFound("provider not found"))
	}

	if err := h.db.Model(&provider).Update("is_active", !provider.IsActive).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	_ = h.providerSvc.ReloadProviders()

	return response.Success(c, map[string]interface{}{"is_active": !provider.IsActive})
}

func (h *AdminHandler) DeleteProvider(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid provider id"))
	}

	if err := h.db.Delete(&model.Provider{}, id).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	_ = h.providerSvc.ReloadProviders()
	return response.Success(c, nil)
}

// ======================== Model Routing Management ========================

func (h *AdminHandler) ListModels(c echo.Context) error {
	var models []model.Model
	if err := h.db.Preload("Provider").Find(&models).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	return response.Success(c, models)
}

func (h *AdminHandler) CreateModel(c echo.Context) error {
	var m model.Model
	if err := json.NewDecoder(c.Request().Body).Decode(&m); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}

	if err := h.db.Create(&m).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	_ = h.providerSvc.ReloadProviders()

	return response.Success(c, m)
}

func (h *AdminHandler) UpdateModel(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return response.Error(c, apierror.BadRequest("invalid model id"))
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(c.Request().Body).Decode(&updates); err != nil {
		return response.Error(c, apierror.BadRequest("invalid request: "+err.Error()))
	}

	if err := h.db.Model(&model.Model{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	_ = h.providerSvc.ReloadProviders()

	return response.Success(c, nil)
}

// ======================== Helpers ========================

func parseUintParam(c echo.Context, key string) (uint, error) {
	val, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}

// suppress unused import
var _ = http.StatusOK
