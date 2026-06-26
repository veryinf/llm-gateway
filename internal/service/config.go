package service

import (
	"encoding/json"
	"llm-gateway/internal/core"
	"llm-gateway/internal/model"

	"log/slog"

	"github.com/tidwall/gjson"
)

func GetConfig(key string) gjson.Result {
	config := GetConfigRaw(key)
	if config == nil {
		return gjson.Result{}
	}
	if !gjson.Valid(config.Value) {
		return gjson.Result{}
	}
	return gjson.Parse(config.Value)
}

func GetConfigRaw(key string) *model.Config {
	var config model.Config
	result := core.DB.Where("`key` = ?", key).First(&config)
	if result.Error != nil {
		slog.Debug("config not found", "key", key)
		return nil
	}
	return &config
}

func GetConfigString(key string) string {
	result := GetConfig(key)
	if !result.Exists() {
		return ""
	}
	return result.String()
}

func GetConfigStringOrDefault(key, defaultValue string) string {
	result := GetConfig(key)
	if !result.Exists() {
		return defaultValue
	}
	val := result.String()
	if val == "" {
		return defaultValue
	}
	return val
}

func SetConfig(key string, value any) error {
	return SetConfigWithDesc(key, value, "")
}

// SetConfigRaw 直接存储原始字符串值，不做 json.Marshal
func SetConfigRaw(key string, value string, description string) error {
	var config model.Config
	result := core.DB.Where("`key` = ?", key).First(&config)
	if result.Error != nil {
		config = model.Config{
			Key:   key,
			Value: value,
		}
		result = core.DB.Create(&config)
	} else {
		updates := map[string]any{
			"value": value,
		}
		if description != "" {
			updates["description"] = description
		}
		result = core.DB.Model(&config).Updates(updates)
	}
	if result.Error != nil {
		slog.Error("failed to set config", "key", key, "error", result.Error)
		return result.Error
	}
	return nil
}

func SetConfigWithDesc(key string, value any, description string) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		slog.Error("failed to marshal config value", "key", key, "error", err)
		return err
	}

	return SetConfigRaw(key, string(jsonBytes), description)
}

func DeleteConfig(key string) error {
	result := core.DB.Where("`key` = ?", key).Delete(&model.Config{})
	if result.Error != nil {
		slog.Error("failed to delete config", "key", key, "error", result.Error)
		return result.Error
	}
	return nil
}
