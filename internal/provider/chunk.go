package provider

import (
	"encoding/json"
	"fmt"
	"time"

	"llm-gateway/internal/model"

	"github.com/tidwall/gjson"
)

type ChunkType string

const (
	ChunkTypeMessage ChunkType = "message" // 普通消息（数据块）
	ChunkTypeUsage   ChunkType = "usage"   // 结束时的用量消息
	ChunkTypeDone    ChunkType = "done"    // 结束事件
)

// LLMResponseChunk LLM 流式响应块
type LLMResponseChunk struct {
	APIType   model.LLMAPIType // API 类型：openai / anthropic
	Type      ChunkType
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
	chunk := &LLMResponseChunk{Raw: raw, RawData: raw, APIType: apiType, Type: ChunkTypeMessage}

	// 检查结束事件
	if apiType == model.APITypeOpenAI && string(raw) == "[DONE]" {
		chunk.Type = ChunkTypeDone
		return chunk
	}

	if gjson.ValidBytes(raw) {
		result := gjson.ParseBytes(raw)
		chunk.RawObject = &result

		// 检查 Anthropic 的 message_stop 事件
		if apiType == model.APITypeAnthropic && result.Get("type").String() == "message_stop" {
			chunk.Type = ChunkTypeDone
		} else if result.Get("usage").Exists() {
			chunk.Type = ChunkTypeUsage
		}
	}
	return chunk
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
	case ChunkTypeMessage:
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
	case ChunkTypeUsage:
		usage := c.RawObject.Get("usage").Value()
		newJSON = map[string]interface{}{
			"usage": usage,
		}
	case ChunkTypeDone:
		// OpenAI 的结束符是 "[DONE]"，直接返回特殊标记
		return &LLMResponseChunk{
			APIType: model.APITypeOpenAI,
			Type:    ChunkTypeDone,
			Raw:     []byte("[DONE]"),
			RawData: c.RawData,
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
	case ChunkTypeMessage:
		text := c.RawObject.Get("choices.0.delta.content").String()
		newJSON = map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": text,
			},
		}
	case ChunkTypeUsage:
		usage := c.RawObject.Get("usage").Value()
		newJSON = map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason": "end_turn",
			},
			"usage": usage,
		}
	case ChunkTypeDone:
		newJSON = map[string]interface{}{
			"type": "message_stop",
		}
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
func (sc *ChunkCollector) Add(data []byte) {
	if !sc.logDetail {
		return
	}
	sc.chunks = append(sc.chunks, &model.RequestChunk{
		TraceID:   sc.traceID,
		Index:     sc.index,
		Data:      string(data),
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
