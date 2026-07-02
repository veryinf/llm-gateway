package handlers

import (
	"crypto/rand"
	"encoding/hex"

	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
)

type UserKeyHandler struct {
	common.BaseHandler
}

// SearchUserKeys 搜索 API Key
func (h *UserKeyHandler) SearchUserKeys(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}

	query := h.DB.Model(&model.UserKey{}).Order("key_id DESC")

	// 关键词搜索：title、key
	if pattern := input.EscapedKw(); pattern != "" {
		query = query.Where("title LIKE ? ESCAPE '\\' OR key LIKE ? ESCAPE '\\'", pattern, pattern)
	}

	// 过滤条件
	for _, filter := range input.Filters {
		switch filter.Field {
		case "uid":
			query = query.Where("uid = ?", filter.Value)
		case "isActive":
			query = query.Where("is_active = ?", filter.Value)
		}
	}

	var count int64
	var keys []model.UserKey
	if err := h.Pagination(&input.Pagination, query, &keys, &count); err != nil {
		return err
	}

	return common.NewDataSet(keys, count)
}

// FetchUserKey 获取单个 API Key
func (h *UserKeyHandler) FetchUserKey(c echo.Context) error {
	input := &struct {
		KeyID int64 `json:"keyId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.KeyID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	var key model.UserKey
	if err := h.DB.First(&key, input.KeyID).Error; err != nil {
		return h.Error(-24, "API Key 不存在")
	}
	return common.NewData(key)
}

// AddUserKey 新增 API Key
func (h *UserKeyHandler) AddUserKey(c echo.Context) error {
	input := &model.UserKey{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.UID, validation.Required),
		validation.Field(&input.Title, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	// 验证用户存在
	var user model.User
	if err := h.DB.First(&user, input.UID).Error; err != nil {
		return h.Error(-24, "用户不存在")
	}

	// 生成 sk- 密钥
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return h.Error(-21, "生成密钥失败")
	}
	input.Key = "sk-" + hex.EncodeToString(raw)
	input.IsActive = true

	if err := h.DB.Create(input).Error; err != nil {
		return h.Error(-21, err.Error())
	}
	return common.NewData(input)
}

// UpdateUserKey 更新 API Key
func (h *UserKeyHandler) UpdateUserKey(c echo.Context) error {
	input, err := h.GetJSON(c)
	if err != nil {
		return err
	}

	keyID := input.Get("keyId")
	if !keyID.Exists() || keyID.Uint() == 0 {
		return h.Error(-23, "keyId is required")
	}

	newState := map[string]any{}

	if input.Get("title").Exists() {
		newState["title"] = input.Get("title").String()
	}
	if input.Get("isActive").Exists() {
		newState["is_active"] = input.Get("isActive").Bool()
	}

	if len(newState) == 0 {
		return h.Success()
	}

	if err := h.DB.Model(&model.UserKey{}).Where("key_id = ?", keyID.Uint()).Updates(newState).Error; err != nil {
		return h.Error(-22, err.Error())
	}
	return h.Success()
}

// RemoveUserKey 删除 API Key
func (h *UserKeyHandler) RemoveUserKey(c echo.Context) error {
	input := &struct {
		KeyID int64 `json:"keyId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.KeyID, validation.Required),
	); err != nil {
		return h.Error(-11, err.Error())
	}

	if err := h.DB.Delete(&model.UserKey{}, input.KeyID).Error; err != nil {
		return h.Error(-23, err.Error())
	}
	return h.Success()
}

func (h *UserKeyHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/apikey/search", h.SearchUserKeys)
	g.POST("/apikey/fetch", h.FetchUserKey)
	g.POST("/apikey/add", h.AddUserKey)
	g.POST("/apikey/update", h.UpdateUserKey)
	g.POST("/apikey/remove", h.RemoveUserKey)
}
