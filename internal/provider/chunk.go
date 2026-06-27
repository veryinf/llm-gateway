package provider

import (
	"encoding/json"
	"fmt"
	"time"

	"llm-gateway/internal/model"

	"github.com/tidwall/gjson"
)

// LLMResponseChunk LLM 流式响应块
type LLMResponseChunk struct {
	APIType   model.LLMAPIType // API 类型：openai / anthropic
	Type      model.ChunkType
	Raw       []byte        //转换过的Data
	RawData   []byte        //原始的Data
	RawObject *gjson.Result //原始DataObject
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"ownedBy"`
}

// NewLLMResponseChunk 从原始字节构造 LLMResponseChunk
func NewLLMResponseChunk(raw []byte, apiType model.LLMAPIType) *LLMResponseChunk {
	chunk := &LLMResponseChunk{Raw: raw, RawData: raw, APIType: apiType, Type: model.ChunkTypeOther}

	// 检查 OpenAI 结束事件
	if apiType == model.APITypeOpenAI && string(raw) == "[DONE]" {
		chunk.Type = model.ChunkTypeDone
		return chunk
	}

	if !gjson.ValidBytes(raw) {
		return chunk
	}

	result := gjson.ParseBytes(raw)
	chunk.RawObject = &result

	// 根据 API 类型解析 chunk 类型
	if apiType == model.APITypeAnthropic {
		chunk.Type = parseAnthropicChunkType(result)
	} else {
		chunk.Type = parseOpenAIChunkType(result)
	}

	return chunk
}

// parseAnthropicChunkType 解析 Anthropic 格式的 chunk 类型
func parseAnthropicChunkType(result gjson.Result) model.ChunkType {
	eventType := result.Get("type").String()

	switch eventType {
	case "message_stop":
		return model.ChunkTypeDone
	case "message_delta":
		// message_delta 包含 usage 和 stop_reason
		if result.Get("usage").Exists() {
			return model.ChunkTypeUsage
		}
	case "content_block_delta":
		// content_block_delta 包含实际内容
		delta := result.Get("delta")
		if delta.Get("thinking").Exists() {
			return model.ChunkTypeReasoning
		}
		if delta.Get("text").Exists() {
			return model.ChunkTypeMessage
		}
	case "content_block_start":
		// 内容块开始，检查是否是 thinking 类型
		if result.Get("content_block.type").String() == "thinking" {
			return model.ChunkTypeReasoning
		}
	case "content_block_stop":
		// 内容块结束，无实际数据
		return model.ChunkTypeOther
	case "message_start":
		// 消息开始，包含 usage
		if result.Get("message.usage").Exists() {
			return model.ChunkTypeUsage
		}
	}

	return model.ChunkTypeOther
}

// parseOpenAIChunkType 解析 OpenAI 格式的 chunk 类型
func parseOpenAIChunkType(result gjson.Result) model.ChunkType {
	choices := result.Get("choices")
	if !choices.Exists() || !choices.IsArray() || len(choices.Array()) == 0 {
		// 没有 choices，可能是 usage chunk
		if result.Get("usage").Exists() {
			return model.ChunkTypeUsage
		}
		return model.ChunkTypeOther
	}

	delta := choices.Array()[0].Get("delta")
	if !delta.Exists() {
		return model.ChunkTypeOther
	}

	// 检查推理内容
	if reasoning := delta.Get("reasoning_content"); reasoning.Exists() && reasoning.Type != gjson.Null {
		return model.ChunkTypeReasoning
	}

	// 检查普通内容
	if content := delta.Get("content"); content.Exists() && content.Type != gjson.Null {
		return model.ChunkTypeMessage
	}

	// 检查 role（通常是第一个 chunk）
	if delta.Get("role").Exists() {
		return model.ChunkTypeOther
	}

	return model.ChunkTypeOther
}

// ToOpenAI 将 chunk 转换为 OpenAI 格式
func (c *LLMResponseChunk) ToOpenAI() (*LLMResponseChunk, error) {
	if c.APIType == model.APITypeOpenAI {
		return c, nil
	}
	if c.RawObject == nil {
		return nil, fmt.Errorf("chunk data is nil")
	}

	var newJSON map[string]interface{}
	switch c.Type {
	case model.ChunkTypeMessage:
		text := c.RawObject.Get("delta.text").String()
		newJSON = map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"delta": map[string]interface{}{
						"content": text,
					},
				},
			},
		}
	case model.ChunkTypeReasoning:
		thinking := c.RawObject.Get("thinking").String()
		if thinking == "" {
			thinking = c.RawObject.Get("delta.thinking").String()
		}
		newJSON = map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"delta": map[string]interface{}{
						"reasoning_content": thinking,
					},
				},
			},
		}
	case model.ChunkTypeUsage:
		usage := c.RawObject.Get("usage").Value()
		newJSON = map[string]interface{}{
			"usage": usage,
		}
	case model.ChunkTypeDone:
		return &LLMResponseChunk{
			APIType: model.APITypeOpenAI,
			Type:    model.ChunkTypeDone,
			Raw:     []byte("[DONE]"),
			RawData: c.RawData,
		}, nil
	case model.ChunkTypeOther:
		// 其他类型，尝试保持原样或跳过
		return &LLMResponseChunk{
			APIType:   model.APITypeOpenAI,
			Type:      model.ChunkTypeOther,
			Raw:       c.Raw,
			RawData:   c.RawData,
			RawObject: c.RawObject,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported chunk type: %s", c.Type)
	}

	newRaw, err := json.Marshal(newJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	newData := gjson.ParseBytes(newRaw)
	return &LLMResponseChunk{
		APIType:   model.APITypeOpenAI,
		Type:      c.Type,
		Raw:       newRaw,
		RawData:   c.RawData,
		RawObject: &newData,
	}, nil
}

// ToAnthropic 将 chunk 转换为 Anthropic 格式
func (c *LLMResponseChunk) ToAnthropic() (*LLMResponseChunk, error) {
	if c.APIType == model.APITypeAnthropic {
		return c, nil
	}
	if c.RawObject == nil {
		return nil, fmt.Errorf("chunk data is nil")
	}

	var newJSON map[string]interface{}
	switch c.Type {
	case model.ChunkTypeMessage:
		text := c.RawObject.Get("choices.0.delta.content").String()
		newJSON = map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": text,
			},
		}
	case model.ChunkTypeReasoning:
		thinking := c.RawObject.Get("choices.0.delta.reasoning_content").String()
		newJSON = map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type":     "thinking",
				"thinking": thinking,
			},
		}
	case model.ChunkTypeUsage:
		usage := c.RawObject.Get("usage").Value()
		newJSON = map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason": "end_turn",
			},
			"usage": usage,
		}
	case model.ChunkTypeDone:
		newJSON = map[string]interface{}{
			"type": "message_stop",
		}
	case model.ChunkTypeOther:
		// 其他类型，尝试保持原样或跳过
		return &LLMResponseChunk{
			APIType:   model.APITypeAnthropic,
			Type:      model.ChunkTypeOther,
			Raw:       c.Raw,
			RawData:   c.RawData,
			RawObject: c.RawObject,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported chunk type: %s", c.Type)
	}

	newRaw, err := json.Marshal(newJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	newData := gjson.ParseBytes(newRaw)
	return &LLMResponseChunk{
		APIType:   model.APITypeAnthropic,
		Type:      c.Type,
		Raw:       newRaw,
		RawData:   c.RawData,
		RawObject: &newData,
	}, nil
}

// ChunkCollector 收集流式响应的 chunks
type ChunkCollector struct {
	traceID   string
	logDetail bool
	chunks    []*model.RequestChunk
	index     int
}

// NewChunkCollector 创建 chunk 收集器
func NewChunkCollector(traceID string, logDetail bool) *ChunkCollector {
	return &ChunkCollector{
		traceID:   traceID,
		logDetail: logDetail,
	}
}

// Add 添加一个 chunk
func (sc *ChunkCollector) Add(chunk *LLMResponseChunk) {
	if !sc.logDetail {
		return
	}
	sc.chunks = append(sc.chunks, &model.RequestChunk{
		TraceID:   sc.traceID,
		Index:     sc.index,
		Type:      chunk.Type,
		Data:      string(chunk.RawData),
		CreatedAt: time.Now(),
	})
	sc.index++
}

// Chunks 返回收集的 chunks
func (sc *ChunkCollector) Chunks() []*model.RequestChunk {
	if !sc.logDetail {
		return nil
	}
	return sc.chunks
}
