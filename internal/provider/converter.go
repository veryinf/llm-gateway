package provider

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/tidwall/gjson"
)

// ==================== 请求转换 ====================

// RequestOpenAIToAnthropic 将 OpenAI 请求转为 Anthropic 请求
func RequestOpenAIToAnthropic(req *http.Request) (*http.Request, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	result := gjson.ParseBytes(bodyBytes)

	// 构建 Anthropic 格式
	anthropicBody := map[string]interface{}{
		"model":       result.Get("model").String(),
		"max_tokens":  result.Get("max_tokens").Int(),
		"stream":      result.Get("stream").Bool(),
		"temperature": result.Get("temperature").Float(),
	}

	// 转换 messages
	var messages []map[string]interface{}
	var system string
	result.Get("messages").ForEach(func(_, msg gjson.Result) bool {
		role := msg.Get("role").String()
		content := msg.Get("content").String()
		if role == "system" {
			system = content
			return true
		}
		messages = append(messages, map[string]interface{}{
			"role":    role,
			"content": content,
		})
		return true
	})

	anthropicBody["messages"] = messages
	if system != "" {
		anthropicBody["system"] = system
	}

	newBody, err := json.Marshal(anthropicBody)
	if err != nil {
		return nil, err
	}

	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), bytes.NewReader(newBody))
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header.Clone()
	return newReq, nil
}

// RequestAnthropicToOpenAI 将 Anthropic 请求转为 OpenAI 请求
func RequestAnthropicToOpenAI(req *http.Request) (*http.Request, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	result := gjson.ParseBytes(bodyBytes)

	// 构建 OpenAI 格式
	openaiBody := map[string]interface{}{
		"model":       result.Get("model").String(),
		"stream":      result.Get("stream").Bool(),
		"max_tokens":  result.Get("max_tokens").Int(),
		"temperature": result.Get("temperature").Float(),
	}

	// 转换 messages
	var messages []map[string]interface{}
	system := result.Get("system").String()
	if system != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": system,
		})
	}
	result.Get("messages").ForEach(func(_, msg gjson.Result) bool {
		messages = append(messages, map[string]interface{}{
			"role":    msg.Get("role").String(),
			"content": msg.Get("content").String(),
		})
		return true
	})
	openaiBody["messages"] = messages

	newBody, err := json.Marshal(openaiBody)
	if err != nil {
		return nil, err
	}

	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), bytes.NewReader(newBody))
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header.Clone()
	return newReq, nil
}

// ==================== 响应转换 ====================

// ResponseOpenAIToAnthropic 将 OpenAI 响应转为 Anthropic 响应
func ResponseOpenAIToAnthropic(body io.ReadCloser) (io.ReadCloser, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	_ = body.Close()

	result := gjson.ParseBytes(bodyBytes)

	anthropicResp := map[string]interface{}{
		"id":    result.Get("id").String(),
		"type":  "message",
		"role":  "assistant",
		"model": result.Get("model").String(),
		"content": []map[string]interface{}{
			{"type": "text", "text": result.Get("choices.0.message.content").String()},
		},
		"stop_reason": "end_turn",
		"usage": map[string]interface{}{
			"input_tokens":  result.Get("usage.prompt_tokens").Int(),
			"output_tokens": result.Get("usage.completion_tokens").Int(),
		},
	}

	newBody, err := json.Marshal(anthropicResp)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(newBody)), nil
}

// ResponseAnthropicToOpenAI 将 Anthropic 响应转为 OpenAI 响应
func ResponseAnthropicToOpenAI(body io.ReadCloser) (io.ReadCloser, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	_ = body.Close()

	result := gjson.ParseBytes(bodyBytes)

	// 提取文本内容
	var content string
	result.Get("content").ForEach(func(_, block gjson.Result) bool {
		if block.Get("type").String() == "text" {
			content = block.Get("text").String()
			return false
		}
		return true
	})

	openaiResp := map[string]interface{}{
		"id":      result.Get("id").String(),
		"object":  "chat.completion",
		"created": 0,
		"model":   result.Get("model").String(),
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     result.Get("usage.input_tokens").Int(),
			"completion_tokens": result.Get("usage.output_tokens").Int(),
			"total_tokens":      result.Get("usage.input_tokens").Int() + result.Get("usage.output_tokens").Int(),
		},
	}

	newBody, err := json.Marshal(openaiResp)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(newBody)), nil
}
