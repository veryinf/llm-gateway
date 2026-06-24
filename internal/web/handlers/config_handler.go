package handlers

import (
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
)

type ConfigHandler struct {
	common.BaseHandler
}

// GetConfig 读取配置项（支持批量）
// POST /api/admin/config/get  body: {"keys": ["mqtt", "http.addr"]}
func (h *ConfigHandler) GetConfig(c echo.Context) error {
	input := &struct {
		Keys []string `json:"keys"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if len(input.Keys) == 0 {
		return h.Error(-1, "keys is required")
	}

	result := make(map[string]any, len(input.Keys))
	for _, key := range input.Keys {
		config := service.GetConfigRaw(key)
		if config != nil {
			result[key] = config.Value
		}
	}
	return common.NewData(result)
}

// SaveConfig 批量保存配置项
// PUT /api/admin/config/save  body: {"configs": {"mqtt.listen": ":1883", "http.addr": ":3001"}}
func (h *ConfigHandler) SaveConfig(c echo.Context) error {
	input := &struct {
		Configs map[string]string `json:"configs"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if len(input.Configs) == 0 {
		return h.Error(-1, "configs is required")
	}

	for key, value := range input.Configs {
		if err := service.SetConfigRaw(key, value); err != nil {
			return h.Error(-1, err.Error())
		}
	}
	return h.Success()
}

func (h *ConfigHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/config/get", h.GetConfig)
	g.POST("/config/save", h.SaveConfig)
}
