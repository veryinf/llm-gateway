package handlers

import (
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
)

type ConfigHandler struct {
	common.BaseHandler
}

// GetConfig 读取配置项
// POST /api/v1/config  body: {"key": "mqtt"}
func (h *ConfigHandler) GetConfig(c echo.Context) error {
	input := &struct {
		Key string `json:"key"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Key, validation.Required),
	); err != nil {
		return err
	}

	config := service.GetConfigRaw(input.Key)
	if config == nil {
		return common.NewData[any](nil)
	}
	return common.NewData(config)
}

// SaveConfig 保存配置项
// POST /api/v1/config/save  body: {"key": "mqtt.listen", "value": ":1883"}
func (h *ConfigHandler) SaveConfig(c echo.Context) error {
	input := &struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if err := validation.ValidateStruct(input,
		validation.Field(&input.Key, validation.Required),
	); err != nil {
		return err
	}

	// 直接存储原始字符串值，不经过 json.Marshal（SetConfigWithDesc 会对 string 双重编码）
	if err := service.SetConfigRaw(input.Key, input.Value); err != nil {
		return h.Error(-1, err.Error())
	}
	return h.Success()
}

func (h *ConfigHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/config", h.GetConfig)
	g.POST("/config/save", h.SaveConfig)
}
