package claude

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel/openrouter"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relay/reasonmap"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	WebSearchMaxUsesLow    = 1
	WebSearchMaxUsesMedium = 5
	WebSearchMaxUsesHigh   = 10
)

func stopReasonClaude2OpenAI(reason string) string {
	return reasonmap.ClaudeStopReasonToOpenAIFinishReason(reason)
}

func maybeMarkClaudeRefusal(c *gin.Context, stopReason string) {
	if c == nil {
		return
	}
	if strings.EqualFold(stopReason, "refusal") {
		common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "claude_stop_reason=refusal")
	}
}

func stripDataURLPrefix(data string) string {
	if strings.HasPrefix(data, "data:") {
		if idx := strings.Index(data, ","); idx != -1 {
			return data[idx+1:]
		}
	}
	return data
}

func openAIFileContentToClaudeMessages(c *gin.Context, mediaMessage dto.MediaContent) ([]dto.ClaudeMediaMessage, error) {
	file := mediaMessage.GetFile()
	if file == nil || file.FileData == "" {
		return nil, nil
	}

	ext := strings.ToLower(filepath.Ext(file.FileName))
	switch ext {
	case ".pdf":
		source := types.NewFileSourceFromData(file.FileData, "application/pdf")
		base64Data, mimeType, err := service.GetBase64Data(c, source, "formatting file for Claude")
		if err != nil {
			return nil, fmt.Errorf("get file data failed: %s", err.Error())
		}
		return []dto.ClaudeMediaMessage{
			{
				Type: "document",
				Source: &dto.ClaudeMessageSource{
					Type:      "base64",
					MediaType: mimeType,
					Data:      base64Data,
				},
			},
		}, nil
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		mimeType := "image/" + strings.TrimPrefix(ext, ".")
		if ext == ".jpg" {
			mimeType = "image/jpeg"
		}
		source := types.NewFileSourceFromData(file.FileData, mimeType)
		base64Data, resolvedMimeType, err := service.GetBase64Data(c, source, "formatting file for Claude")
		if err != nil {
			return nil, fmt.Errorf("get file data failed: %s", err.Error())
		}
		return []dto.ClaudeMediaMessage{
			{
				Type: "image",
				Source: &dto.ClaudeMessageSource{
					Type:      "base64",
					MediaType: resolvedMimeType,
					Data:      base64Data,
				},
			},
		}, nil
	case ".txt", ".text", ".md", ".markdown", ".json", ".jsonl", ".csv", ".tsv", ".xml", ".html", ".htm", ".yaml", ".yml", ".log":
		decoded, err := base64.StdEncoding.DecodeString(stripDataURLPrefix(file.FileData))
		if err != nil {
			return nil, fmt.Errorf("decode text file failed: %w", err)
		}
		return []dto.ClaudeMediaMessage{
			{
				Type: "text",
				Text: common.GetPointer[string](string(decoded)),
			},
		}, nil
	default:
		return nil, nil
	}
}

func RequestOpenAI2ClaudeMessage(c *gin.Context, textRequest dto.GeneralOpenAIRequest) (*dto.ClaudeRequest, error) {
	claudeTools := make([]any, 0, len(textRequest.Tools))

	for _, tool := range textRequest.Tools {
		if params, ok := tool.Function.Parameters.(map[string]any); ok {
			claudeTool := dto.Tool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
			}
			claudeTool.InputSchema = make(map[string]interface{})
			if params["type"] != nil {
				claudeTool.InputSchema["type"] = params["type"].(string)
			}
			claudeTool.InputSchema["properties"] = params["properties"]
			claudeTool.InputSchema["required"] = params["required"]
			for s, a := range params {
				if s == "type" || s == "properties" || s == "required" {
					continue
				}
				claudeTool.InputSchema[s] = a
			}
			claudeTools = append(claudeTools, &claudeTool)
		}
	}

	// Web search tool
	// https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/web-search-tool
	if textRequest.WebSearchOptions != nil {
		webSearchTool := dto.ClaudeWebSearchTool{
			Type: "web_search_20250305",
			Name: "web_search",
		}

		// 处理 user_location
		if textRequest.WebSearchOptions.UserLocation != nil {
			anthropicUserLocation := &dto.ClaudeWebSearchUserLocation{
				Type: "approximate", // 固定为 "approximate"
			}

			// 解析 UserLocation JSON
			var userLocationMap map[string]interface{}
			if err := common.Unmarshal(textRequest.WebSearchOptions.UserLocation, &userLocationMap); err == nil {
				// 检查是否有 approximate 字段
				if approximateData, ok := userLocationMap["approximate"].(map[string]interface{}); ok {
					if timezone, ok := approximateData["timezone"].(string); ok && timezone != "" {
						anthropicUserLocation.Timezone = timezone
					}
					if country, ok := approximateData["country"].(string); ok && country != "" {
						anthropicUserLocation.Country = country
					}
					if region, ok := approximateData["region"].(string); ok && region != "" {
						anthropicUserLocation.Region = region
					}
					if city, ok := approximateData["city"].(string); ok && city != "" {
						anthropicUserLocation.City = city
					}
				}
			}

			webSearchTool.UserLocation = anthropicUserLocation
		}

		// 处理 search_context_size 转换为 max_uses
		if textRequest.WebSearchOptions.SearchContextSize != "" {
			switch textRequest.WebSearchOptions.SearchContextSize {
			case "low":
				webSearchTool.MaxUses = WebSearchMaxUsesLow
			case "medium":
				webSearchTool.MaxUses = WebSearchMaxUsesMedium
			case "high":
				webSearchTool.MaxUses = WebSearchMaxUsesHigh
			}
		}

		claudeTools = append(claudeTools, &webSearchTool)
	}

	claudeRequest := dto.ClaudeRequest{
		Model:         textRequest.Model,
		StopSequences: nil,
		Temperature:   textRequest.Temperature,
		Tools:         claudeTools,
	}
	if maxTokens := textRequest.GetMaxTokens(); maxTokens > 0 {
		claudeRequest.MaxTokens = common.GetPointer(maxTokens)
	}
	if textRequest.TopP != nil {
		claudeRequest.TopP = common.GetPointer(*textRequest.TopP)
	}
	if textRequest.TopK != nil {
		claudeRequest.TopK = common.GetPointer(*textRequest.TopK)
	}
	if textRequest.IsStream(nil) {
		claudeRequest.Stream = common.GetPointer(true)
	}

	// 处理 tool_choice 和 parallel_tool_calls
	if textRequest.ToolChoice != nil || textRequest.ParallelTooCalls != nil {
		claudeToolChoice := mapToolChoice(textRequest.ToolChoice, textRequest.ParallelTooCalls)
		if claudeToolChoice != nil {
			claudeRequest.ToolChoice = claudeToolChoice
		}
	}

	if claudeRequest.MaxTokens == nil || *claudeRequest.MaxTokens == 0 {
		defaultMaxTokens := uint(model_setting.GetClaudeSettings().GetDefaultMaxTokens(textRequest.Model))
		claudeRequest.MaxTokens = &defaultMaxTokens
	}

	if baseModel, effortLevel, ok := reasoning.TrimEffortSuffix(textRequest.Model); ok && effortLevel != "" &&
		(strings.HasPrefix(textRequest.Model, "claude-opus-4-6") ||
			strings.HasPrefix(textRequest.Model, "claude-opus-4-7") ||
			strings.HasPrefix(textRequest.Model, "claude-opus-4-8")) {
		claudeRequest.Model = baseModel
		claudeRequest.Thinking = &dto.Thinking{
			Type: "adaptive",
		}
		claudeRequest.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, effortLevel))
		if strings.HasPrefix(baseModel, "claude-opus-4-7") ||
			strings.HasPrefix(baseModel, "claude-opus-4-8") {
			// Opus 4.7/4.8 reject non-default temperature/top_p/top_k with 400
			// and defaults display to "omitted"; restore the 4.6 visible summary.
			claudeRequest.Thinking.Display = "summarized"
			claudeRequest.Temperature = nil
			claudeRequest.TopP = nil
			claudeRequest.TopK = nil
		} else {
			claudeRequest.TopP = nil
			claudeRequest.Temperature = common.GetPointer[float64](1.0)
		}
	} else if model_setting.GetClaudeSettings().ThinkingAdapterEnabled &&
		strings.HasSuffix(textRequest.Model, "-thinking") {

		trimmedModel := strings.TrimSuffix(textRequest.Model, "-thinking")
		if strings.HasPrefix(trimmedModel, "claude-opus-4-7") ||
			strings.HasPrefix(trimmedModel, "claude-opus-4-8") {
			// Opus 4.7/4.8 reject thinking.type="enabled"; use adaptive at high effort.
			claudeRequest.Thinking = &dto.Thinking{Type: "adaptive", Display: "summarized"}
			claudeRequest.OutputConfig = json.RawMessage(`{"effort":"high"}`)
			claudeRequest.Temperature = nil
			claudeRequest.TopP = nil
			claudeRequest.TopK = nil
		} else {
			// 因为BudgetTokens 必须大于1024
			if claudeRequest.MaxTokens == nil || *claudeRequest.MaxTokens < 1280 {
				claudeRequest.MaxTokens = common.GetPointer[uint](1280)
			}

			// BudgetTokens 为 max_tokens 的 80%
			claudeRequest.Thinking = &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: common.GetPointer[int](int(float64(*claudeRequest.MaxTokens) * model_setting.GetClaudeSettings().ThinkingAdapterBudgetTokensPercentage)),
			}
			// TODO: 临时处理
			// https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking#important-considerations-when-using-extended-thinking
			claudeRequest.TopP = nil
			claudeRequest.Temperature = common.GetPointer[float64](1.0)
		}
		if !model_setting.ShouldPreserveThinkingSuffix(textRequest.Model) {
			claudeRequest.Model = trimmedModel
		}
	}

	if textRequest.ReasoningEffort != "" {
		switch textRequest.ReasoningEffort {
		case "low":
			claudeRequest.Thinking = &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: common.GetPointer[int](1280),
			}
		case "medium":
			claudeRequest.Thinking = &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: common.GetPointer[int](2048),
			}
		case "high":
			claudeRequest.Thinking = &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: common.GetPointer[int](4096),
			}
		}
	}

	// 指定了 reasoning 参数,覆盖 budgetTokens
	if textRequest.Reasoning != nil {
		var reasoning openrouter.RequestReasoning
		if err := common.Unmarshal(textRequest.Reasoning, &reasoning); err != nil {
			return nil, err
		}

		budgetTokens := reasoning.MaxTokens
		if budgetTokens > 0 {
			claudeRequest.Thinking = &dto.Thinking{
				Type:         "enabled",
				BudgetTokens: &budgetTokens,
			}
		}
	}

	if textRequest.Stop != nil {
		// stop maybe string/array string, convert to array string
		switch textRequest.Stop.(type) {
		case string:
			claudeRequest.StopSequences = []string{textRequest.Stop.(string)}
		case []interface{}:
			stopSequences := make([]string, 0)
			for _, stop := range textRequest.Stop.([]interface{}) {
				stopSequences = append(stopSequences, stop.(string))
			}
			claudeRequest.StopSequences = stopSequences
		}
	}
	formatMessages := make([]dto.Message, 0)
	lastMessage := dto.Message{
		Role: "tool",
	}
	for i, message := range textRequest.Messages {
		if message.Role == "" {
			textRequest.Messages[i].Role = "user"
		}
		fmtMessage := dto.Message{
			Role:    message.Role,
			Content: message.Content,
		}
		if message.Role == "tool" {
			fmtMessage.ToolCallId = message.ToolCallId
		}
		if message.Role == "assistant" && message.ToolCalls != nil {
			fmtMessage.ToolCalls = message.ToolCalls
		}
		if lastMessage.Role == message.Role && lastMessage.Role != "tool" {
			if lastMessage.IsStringContent() && message.IsStringContent() {
				fmtMessage.SetStringContent(strings.Trim(fmt.Sprintf("%s %s", lastMessage.StringContent(), message.StringContent()), "\""))
				// delete last message
				formatMessages = formatMessages[:len(formatMessages)-1]
			}
		}
		if fmtMessage.Content == nil || (fmtMessage.IsStringContent() && fmtMessage.StringContent() == "") {
			fmtMessage.SetStringContent("...")
		}
		formatMessages = append(formatMessages, fmtMessage)
		lastMessage = fmtMessage
	}

	claudeMessages := make([]dto.ClaudeMessage, 0)
	isFirstMessage := true
	// 初始化system消息数组，用于累积多个system消息
	var systemMessages []dto.ClaudeMediaMessage

	for _, message := range formatMessages {
		if message.Role == "system" {
			// 根据Claude API规范，system字段使用数组格式更有通用性
			if message.IsStringContent() {
				if text := message.StringContent(); text != "" {
					systemMessages = append(systemMessages, dto.ClaudeMediaMessage{
						Type: "text",
						Text: common.GetPointer[string](text),
					})
				}
			} else {
				// 支持复合内容的system消息（虽然不常见，但需要考虑完整性）
				for _, ctx := range message.ParseContent() {
					if ctx.Type == "text" && ctx.Text != "" {
						systemMessages = append(systemMessages, dto.ClaudeMediaMessage{
							Type: "text",
							Text: common.GetPointer[string](ctx.Text),
						})
					}
					// 未来可以在这里扩展对图片等其他类型的支持
				}
			}
		} else {
			if isFirstMessage {
				isFirstMessage = false
				if message.Role != "user" {
					// fix: first message is assistant, add user message
					claudeMessage := dto.ClaudeMessage{
						Role: "user",
						Content: []dto.ClaudeMediaMessage{
							{
								Type: "text",
								Text: common.GetPointer[string]("..."),
							},
						},
					}
					claudeMessages = append(claudeMessages, claudeMessage)
				}
			}
			claudeMessage := dto.ClaudeMessage{
				Role: message.Role,
			}
			if message.Role == "tool" {
				if len(claudeMessages) > 0 && claudeMessages[len(claudeMessages)-1].Role == "user" {
					lastMessage := claudeMessages[len(claudeMessages)-1]
					if content, ok := lastMessage.Content.(string); ok {
						lastMessage.Content = []dto.ClaudeMediaMessage{
							{
								Type: "text",
								Text: common.GetPointer[string](content),
							},
						}
					}
					lastMessage.Content = append(lastMessage.Content.([]dto.ClaudeMediaMessage), dto.ClaudeMediaMessage{
						Type:      "tool_result",
						ToolUseId: message.ToolCallId,
						Content:   message.Content,
					})
					claudeMessages[len(claudeMessages)-1] = lastMessage
					continue
				} else {
					claudeMessage.Role = "user"
					claudeMessage.Content = []dto.ClaudeMediaMessage{
						{
							Type:      "tool_result",
							ToolUseId: message.ToolCallId,
							Content:   message.Content,
						},
					}
				}
			} else if message.IsStringContent() && message.ToolCalls == nil {
				text := message.StringContent()
				if text == "" {
					text = "..."
				}
				claudeMessage.Content = text
			} else {
				claudeMediaMessages := make([]dto.ClaudeMediaMessage, 0)
				for _, mediaMessage := range message.ParseContent() {
					switch mediaMessage.Type {
					case "text":
						if mediaMessage.Text != "" {
							claudeMediaMessages = append(claudeMediaMessages, dto.ClaudeMediaMessage{
								Type: "text",
								Text: common.GetPointer[string](mediaMessage.Text),
							})
						}
					case dto.ContentTypeFile:
						fileMessages, err := openAIFileContentToClaudeMessages(c, mediaMessage)
						if err != nil {
							return nil, err
						}
						claudeMediaMessages = append(claudeMediaMessages, fileMessages...)
					default:
						source := mediaMessage.ToFileSource()
						if source == nil {
							continue
						}
						base64Data, mimeType, err := service.GetBase64Data(c, source, "formatting image for Claude")
						if err != nil {
							return nil, fmt.Errorf("get file data failed: %s", err.Error())
						}
						claudeMediaMessage := dto.ClaudeMediaMessage{
							Source: &dto.ClaudeMessageSource{
								Type: "base64",
							},
						}
						if strings.HasPrefix(mimeType, "application/pdf") {
							claudeMediaMessage.Type = "document"
						} else {
							claudeMediaMessage.Type = "image"
						}

						claudeMediaMessage.Source.MediaType = mimeType
						claudeMediaMessage.Source.Data = base64Data
						claudeMediaMessages = append(claudeMediaMessages, claudeMediaMessage)
						continue
					}
				}

				if message.ToolCalls != nil {
					for _, toolCall := range message.ParseToolCalls() {
						inputObj := make(map[string]any)
						if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &inputObj); err != nil {
							common.SysLog("tool call function arguments is not a map[string]any: " + fmt.Sprintf("%v", toolCall.Function.Arguments))
							continue
						}
						claudeMediaMessages = append(claudeMediaMessages, dto.ClaudeMediaMessage{
							Type:  "tool_use",
							Id:    toolCall.ID,
							Name:  toolCall.Function.Name,
							Input: inputObj,
						})
					}
				}
				claudeMessage.Content = claudeMediaMessages
			}
			claudeMessages = append(claudeMessages, claudeMessage)
		}
	}

	// 设置累积的system消息
	if len(systemMessages) > 0 {
		claudeRequest.System = systemMessages
	}

	claudeRequest.Prompt = ""
	claudeRequest.Messages = claudeMessages
	return &claudeRequest, nil
}

type responsesInputItem struct {
	Type      string          `json:"type,omitempty"`
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Output    any             `json:"output,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments any             `json:"arguments,omitempty"`
}

func RequestOpenAIResponses2ClaudeMessage(c *gin.Context, request dto.OpenAIResponsesRequest) (*dto.ClaudeRequest, error) {
	openAIRequest, err := openAIResponsesRequestToChatRequest(request)
	if err != nil {
		return nil, err
	}
	return RequestOpenAI2ClaudeMessage(c, *openAIRequest)
}

func openAIResponsesRequestToChatRequest(request dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	messages, err := openAIResponsesInputToMessages(request.Input)
	if err != nil {
		return nil, err
	}
	if len(request.Instructions) > 0 && string(request.Instructions) != "null" {
		instructionText, err := rawMessageToText(request.Instructions)
		if err != nil {
			return nil, fmt.Errorf("invalid responses instructions: %w", err)
		}
		if strings.TrimSpace(instructionText) != "" {
			messages = append([]dto.Message{{
				Role:    "system",
				Content: instructionText,
			}}, messages...)
		}
	}
	if len(messages) == 0 {
		messages = []dto.Message{{
			Role:    "user",
			Content: "...",
		}}
	}

	chatRequest := &dto.GeneralOpenAIRequest{
		Model:            request.Model,
		Messages:         messages,
		Stream:           request.Stream,
		Temperature:      request.Temperature,
		TopP:             request.TopP,
		MaxTokens:        request.MaxOutputTokens,
		StreamOptions:    request.StreamOptions,
		Metadata:         request.Metadata,
		Store:            request.Store,
		User:             request.User,
		PromptCacheKey:   rawMessageString(request.PromptCacheKey),
		ServiceTier:      stringRawMessage(request.ServiceTier),
		Reasoning:        reasoningRawMessage(request.Reasoning),
		ResponseFormat:   responsesTextToChatResponseFormat(request.Text),
		ParallelTooCalls: rawMessageBoolPointer(request.ParallelToolCalls),
	}

	if len(request.Tools) > 0 {
		tools, err := openAIResponsesToolsToChatTools(request.Tools)
		if err != nil {
			return nil, err
		}
		chatRequest.Tools = tools
	}
	if len(request.ToolChoice) > 0 {
		toolChoice, err := openAIResponsesToolChoiceToChatToolChoice(request.ToolChoice)
		if err != nil {
			return nil, err
		}
		chatRequest.ToolChoice = toolChoice
	}

	return chatRequest, nil
}

func openAIResponsesInputToMessages(input json.RawMessage) ([]dto.Message, error) {
	if len(input) == 0 || string(input) == "null" {
		return nil, nil
	}

	switch common.GetJsonType(input) {
	case "string":
		text, err := rawMessageToText(input)
		if err != nil {
			return nil, err
		}
		return []dto.Message{{Role: "user", Content: text}}, nil
	case "array":
		var items []responsesInputItem
		if err := common.Unmarshal(input, &items); err != nil {
			return nil, err
		}
		messages := make([]dto.Message, 0, len(items))
		for _, item := range items {
			message, ok, err := responsesInputItemToMessage(item)
			if err != nil {
				return nil, err
			}
			if ok {
				messages = append(messages, message)
			}
		}
		return messages, nil
	default:
		return nil, fmt.Errorf("unsupported responses input type: %s", common.GetJsonType(input))
	}
}

func responsesInputItemToMessage(item responsesInputItem) (dto.Message, bool, error) {
	switch item.Type {
	case "function_call_output":
		content := common.Interface2String(item.Output)
		if content == "" && item.Output != nil {
			data, err := common.Marshal(item.Output)
			if err != nil {
				return dto.Message{}, false, err
			}
			content = string(data)
		}
		return dto.Message{
			Role:       "tool",
			Content:    content,
			ToolCallId: item.CallID,
		}, true, nil
	case "function_call":
		arguments, err := argumentsString(item.Arguments)
		if err != nil {
			return dto.Message{}, false, err
		}
		message := dto.Message{Role: "assistant"}
		message.SetNullContent()
		message.SetToolCalls([]dto.ToolCallRequest{{
			ID:   item.CallID,
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      item.Name,
				Arguments: arguments,
			},
		}})
		return message, true, nil
	case "", "message", "input_message", "output_message":
		role := strings.TrimSpace(item.Role)
		if role == "" {
			role = "user"
		}
		content, err := responsesContentToChatContent(item.Content, role)
		if err != nil {
			return dto.Message{}, false, err
		}
		return dto.Message{Role: role, Content: content}, true, nil
	default:
		return dto.Message{}, false, nil
	}
}

func responsesContentToChatContent(content json.RawMessage, role string) (any, error) {
	if len(content) == 0 || string(content) == "null" {
		return "", nil
	}
	switch common.GetJsonType(content) {
	case "string":
		return rawMessageToText(content)
	case "array":
		var parts []map[string]any
		if err := common.Unmarshal(content, &parts); err != nil {
			return nil, err
		}
		chatParts := make([]any, 0, len(parts))
		for _, part := range parts {
			contentType, _ := part["type"].(string)
			switch contentType {
			case "input_text", "output_text", "text":
				chatParts = append(chatParts, map[string]any{
					"type": "text",
					"text": common.Interface2String(part["text"]),
				})
			case "input_image", dto.ContentTypeImageURL:
				chatParts = append(chatParts, map[string]any{
					"type":      dto.ContentTypeImageURL,
					"image_url": responsesURLValue(part["image_url"]),
				})
			case "input_file", dto.ContentTypeFile:
				chatParts = append(chatParts, map[string]any{
					"type": "file",
					"file": responsesFileValue(part),
				})
			}
		}
		if len(chatParts) == 0 {
			return "", nil
		}
		if len(chatParts) == 1 {
			if textPart, ok := chatParts[0].(map[string]any); ok && textPart["type"] == "text" {
				return common.Interface2String(textPart["text"]), nil
			}
		}
		return chatParts, nil
	default:
		text, err := rawMessageToText(content)
		if err != nil {
			return nil, err
		}
		if role == "assistant" || role == "user" || role == "system" || role == "developer" {
			return text, nil
		}
		return text, nil
	}
}

func rawMessageToText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	if common.GetJsonType(raw) == "string" {
		var text string
		if err := common.Unmarshal(raw, &text); err != nil {
			return "", err
		}
		return text, nil
	}
	return string(raw), nil
}

func rawMessageString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	text, err := rawMessageToText(raw)
	if err != nil {
		return ""
	}
	return text
}

func stringRawMessage(value string) json.RawMessage {
	if value == "" {
		return nil
	}
	data, err := common.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}

func rawMessageBoolPointer(raw json.RawMessage) *bool {
	if len(raw) == 0 || string(raw) == "null" || common.GetJsonType(raw) != "boolean" {
		return nil
	}
	var value bool
	if err := common.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return &value
}

func reasoningRawMessage(reasoning *dto.Reasoning) json.RawMessage {
	if reasoning == nil {
		return nil
	}
	data, err := common.Marshal(reasoning)
	if err != nil {
		return nil
	}
	return data
}

func argumentsString(arguments any) (string, error) {
	switch value := arguments.(type) {
	case nil:
		return "{}", nil
	case string:
		return value, nil
	case json.RawMessage:
		if len(value) == 0 {
			return "{}", nil
		}
		if common.GetJsonType(value) == "string" {
			var text string
			if err := common.Unmarshal(value, &text); err != nil {
				return "", err
			}
			return text, nil
		}
		return string(value), nil
	default:
		data, err := common.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func responsesURLValue(value any) any {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		if url := common.Interface2String(typed["url"]); url != "" {
			return map[string]any{
				"url":    url,
				"detail": common.Interface2String(typed["detail"]),
			}
		}
	}
	return value
}

func responsesFileValue(part map[string]any) any {
	if file := part["file"]; file != nil {
		return file
	}
	fileValue := map[string]any{}
	if fileID := common.Interface2String(part["file_id"]); fileID != "" {
		fileValue["file_id"] = fileID
	}
	if fileData := common.Interface2String(part["file_data"]); fileData != "" {
		fileValue["file_data"] = fileData
	}
	if filename := common.Interface2String(part["filename"]); filename != "" {
		fileValue["filename"] = filename
	}
	if fileURL := common.Interface2String(part["file_url"]); fileURL != "" {
		fileValue["file_data"] = fileURL
	}
	if len(fileValue) == 0 {
		return part
	}
	return fileValue
}

func openAIResponsesToolsToChatTools(raw json.RawMessage) ([]dto.ToolCallRequest, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var tools []map[string]any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil, err
	}
	chatTools := make([]dto.ToolCallRequest, 0, len(tools))
	for _, tool := range tools {
		if toolType, _ := tool["type"].(string); toolType != "function" {
			continue
		}
		name := common.Interface2String(tool["name"])
		if name == "" {
			continue
		}
		parameters := tool["parameters"]
		if parameters == nil {
			parameters = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []any{},
			}
		}
		chatTools = append(chatTools, dto.ToolCallRequest{
			Type: "function",
			Function: dto.FunctionRequest{
				Name:        name,
				Description: common.Interface2String(tool["description"]),
				Parameters:  parameters,
			},
		})
	}
	return chatTools, nil
}

func openAIResponsesToolChoiceToChatToolChoice(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if common.GetJsonType(raw) == "string" {
		var toolChoice string
		if err := common.Unmarshal(raw, &toolChoice); err != nil {
			return nil, err
		}
		return toolChoice, nil
	}
	var toolChoice map[string]any
	if err := common.Unmarshal(raw, &toolChoice); err != nil {
		return nil, err
	}
	if toolChoice["type"] == "function" {
		if name := common.Interface2String(toolChoice["name"]); name != "" {
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": name,
				},
			}, nil
		}
	}
	return toolChoice, nil
}

func responsesTextToChatResponseFormat(raw json.RawMessage) *dto.ResponseFormat {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var payload map[string]json.RawMessage
	if err := common.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	formatRaw, ok := payload["format"]
	if !ok || len(formatRaw) == 0 || string(formatRaw) == "null" {
		return nil
	}
	var format map[string]json.RawMessage
	if err := common.Unmarshal(formatRaw, &format); err != nil {
		return nil
	}
	typeRaw, ok := format["type"]
	if !ok {
		return nil
	}
	formatType, err := rawMessageToText(typeRaw)
	if err != nil || formatType == "" {
		return nil
	}
	responseFormat := &dto.ResponseFormat{Type: formatType}
	if formatType == "json_schema" {
		responseFormat.JsonSchema = formatRaw
	}
	return responseFormat
}

func StreamResponseClaude2OpenAI(claudeResponse *dto.ClaudeResponse) *dto.ChatCompletionsStreamResponse {
	var response dto.ChatCompletionsStreamResponse
	response.Object = "chat.completion.chunk"
	response.Model = claudeResponse.Model
	response.Choices = make([]dto.ChatCompletionsStreamResponseChoice, 0)
	tools := make([]dto.ToolCallResponse, 0)
	fcIdx := 0
	if claudeResponse.Index != nil {
		fcIdx = *claudeResponse.Index
	}
	var choice dto.ChatCompletionsStreamResponseChoice
	if claudeResponse.Type == "message_start" {
		if claudeResponse.Message != nil {
			response.Id = claudeResponse.Message.Id
			response.Model = claudeResponse.Message.Model
		}
		//claudeUsage = &claudeResponse.Message.Usage
		choice.Delta.SetContentString("")
		choice.Delta.Role = "assistant"
	} else if claudeResponse.Type == "content_block_start" {
		if claudeResponse.ContentBlock != nil {
			// 如果是文本块，尽可能发送首段文本（若存在）
			if claudeResponse.ContentBlock.Type == "text" && claudeResponse.ContentBlock.Text != nil {
				choice.Delta.SetContentString(*claudeResponse.ContentBlock.Text)
			}
			if claudeResponse.ContentBlock.Type == "tool_use" {
				tools = append(tools, dto.ToolCallResponse{
					Index: common.GetPointer(fcIdx),
					ID:    claudeResponse.ContentBlock.Id,
					Type:  "function",
					Function: dto.FunctionResponse{
						Name:      claudeResponse.ContentBlock.Name,
						Arguments: "",
					},
				})
			}
		} else {
			return nil
		}
	} else if claudeResponse.Type == "content_block_delta" {
		if claudeResponse.Delta != nil {
			choice.Delta.Content = claudeResponse.Delta.Text
			switch claudeResponse.Delta.Type {
			case "input_json_delta":
				tools = append(tools, dto.ToolCallResponse{
					Type:  "function",
					Index: common.GetPointer(fcIdx),
					Function: dto.FunctionResponse{
						Arguments: *claudeResponse.Delta.PartialJson,
					},
				})
			case "signature_delta":
				// 加密的不处理
				signatureContent := "\n"
				choice.Delta.ReasoningContent = &signatureContent
			case "thinking_delta":
				choice.Delta.ReasoningContent = claudeResponse.Delta.Thinking
			}
		}
	} else if claudeResponse.Type == "message_delta" {
		if claudeResponse.Delta != nil && claudeResponse.Delta.StopReason != nil {
			finishReason := stopReasonClaude2OpenAI(*claudeResponse.Delta.StopReason)
			if finishReason != "null" {
				choice.FinishReason = &finishReason
			}
		}
		//claudeUsage = &claudeResponse.Usage
	} else if claudeResponse.Type == "message_stop" {
		return nil
	} else {
		return nil
	}
	if len(tools) > 0 {
		choice.Delta.Content = nil // compatible with other OpenAI derivative applications, like LobeOpenAICompatibleFactory ...
		choice.Delta.ToolCalls = tools
	}
	response.Choices = append(response.Choices, choice)

	return &response
}

func ResponseClaude2OpenAI(claudeResponse *dto.ClaudeResponse) *dto.OpenAITextResponse {
	choices := make([]dto.OpenAITextResponseChoice, 0)
	fullTextResponse := dto.OpenAITextResponse{
		Id:      fmt.Sprintf("chatcmpl-%s", common.GetUUID()),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
	}
	var responseText string
	var responseThinking string
	if len(claudeResponse.Content) > 0 {
		responseText = claudeResponse.Content[0].GetText()
		if claudeResponse.Content[0].Thinking != nil {
			responseThinking = *claudeResponse.Content[0].Thinking
		}
	}
	tools := make([]dto.ToolCallResponse, 0)
	thinkingContent := ""

	fullTextResponse.Id = claudeResponse.Id
	for _, message := range claudeResponse.Content {
		switch message.Type {
		case "tool_use":
			args, _ := json.Marshal(message.Input)
			tools = append(tools, dto.ToolCallResponse{
				ID:   message.Id,
				Type: "function", // compatible with other OpenAI derivative applications
				Function: dto.FunctionResponse{
					Name:      message.Name,
					Arguments: string(args),
				},
			})
		case "thinking":
			// 加密的不管， 只输出明文的推理过程
			if message.Thinking != nil {
				thinkingContent = *message.Thinking
			}
		case "text":
			responseText = message.GetText()
		}
	}
	choice := dto.OpenAITextResponseChoice{
		Index: 0,
		Message: dto.Message{
			Role: "assistant",
		},
		FinishReason: stopReasonClaude2OpenAI(claudeResponse.StopReason),
	}
	choice.SetStringContent(responseText)
	if len(responseThinking) > 0 {
		choice.ReasoningContent = &responseThinking
	}
	if len(tools) > 0 {
		choice.Message.SetToolCalls(tools)
	}
	if thinkingContent != "" {
		choice.Message.ReasoningContent = &thinkingContent
	}
	fullTextResponse.Model = claudeResponse.Model
	choices = append(choices, choice)
	fullTextResponse.Choices = choices
	return &fullTextResponse
}

func ResponseClaude2OpenAIResponses(claudeResponse *dto.ClaudeResponse, usage *dto.Usage, created int64) *dto.OpenAIResponsesResponse {
	response := &dto.OpenAIResponsesResponse{
		ID:        claudeResponse.Id,
		Object:    "response",
		CreatedAt: int(created),
		Status:    json.RawMessage(`"completed"`),
		Model:     claudeResponse.Model,
		Output:    make([]dto.ResponsesOutput, 0, len(claudeResponse.Content)),
		Usage:     buildResponsesUsageFromClaudeUsage(usage),
	}
	if response.ID == "" {
		response.ID = fmt.Sprintf("resp_%s", common.GetUUID())
	}
	for _, content := range claudeResponse.Content {
		switch content.Type {
		case "text":
			output := dto.ResponsesOutput{
				Type:   "message",
				ID:     fmt.Sprintf("msg_%s", common.GetUUID()),
				Status: "completed",
				Role:   "assistant",
				Content: []dto.ResponsesOutputContent{{
					Type:        "output_text",
					Text:        content.GetText(),
					Annotations: []interface{}{},
				}},
			}
			response.Output = append(response.Output, output)
		case "tool_use":
			args, _ := common.Marshal(content.Input)
			response.Output = append(response.Output, dto.ResponsesOutput{
				Type:      "function_call",
				ID:        content.Id,
				Status:    "completed",
				CallId:    content.Id,
				Name:      content.Name,
				Arguments: args,
			})
		}
	}
	return response
}

func buildResponsesUsageFromClaudeUsage(usage *dto.Usage) *dto.Usage {
	if usage == nil {
		return nil
	}
	openAIUsage := buildOpenAIStyleUsageFromClaudeUsage(usage)
	openAIUsage.InputTokens = openAIUsage.PromptTokens
	openAIUsage.OutputTokens = openAIUsage.CompletionTokens
	openAIUsage.TotalTokens = openAIUsage.InputTokens + openAIUsage.OutputTokens
	openAIUsage.InputTokensDetails = &dto.InputTokenDetails{
		CachedTokens: openAIUsage.PromptTokensDetails.CachedTokens,
	}
	return &openAIUsage
}

func handleClaudeStreamAsResponses(c *gin.Context, info *relaycommon.RelayInfo, claudeInfo *ClaudeResponseInfo, claudeResponse *dto.ClaudeResponse) *types.NewAPIError {
	if !FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo) {
		return nil
	}

	responseID := claudeInfo.ResponseId
	if responseID == "" {
		responseID = fmt.Sprintf("resp_%s", common.GetUUID())
	}
	response := &dto.OpenAIResponsesResponse{
		ID:        responseID,
		Object:    "response",
		CreatedAt: int(claudeInfo.Created),
		Status:    json.RawMessage(`"in_progress"`),
		Model:     claudeInfo.Model,
	}
	if response.Model == "" {
		response.Model = info.UpstreamModelName
	}

	send := func(event dto.ResponsesStreamResponse) *types.NewAPIError {
		data, err := common.Marshal(event)
		if err != nil {
			return types.NewError(err, types.ErrorCodeBadResponseBody)
		}
		helper.ResponseChunkData(c, event, string(data))
		return nil
	}

	switch claudeResponse.Type {
	case "message_start":
		return send(dto.ResponsesStreamResponse{
			Type:     "response.created",
			Response: response,
		})
	case "content_block_start":
		if claudeResponse.ContentBlock == nil {
			return nil
		}
		index := claudeResponse.GetIndex()
		switch claudeResponse.ContentBlock.Type {
		case "text":
			item := &dto.ResponsesOutput{
				Type:   "message",
				ID:     fmt.Sprintf("msg_%s", responseID),
				Status: "in_progress",
				Role:   "assistant",
			}
			if err := send(dto.ResponsesStreamResponse{
				Type:        dto.ResponsesOutputTypeItemAdded,
				OutputIndex: common.GetPointer(index),
				Item:        item,
			}); err != nil {
				return err
			}
			return send(dto.ResponsesStreamResponse{
				Type:         "response.content_part.added",
				OutputIndex:  common.GetPointer(index),
				ContentIndex: common.GetPointer(0),
				ItemID:       item.ID,
			})
		case "tool_use":
			item := &dto.ResponsesOutput{
				Type:      "function_call",
				ID:        claudeResponse.ContentBlock.Id,
				Status:    "in_progress",
				CallId:    claudeResponse.ContentBlock.Id,
				Name:      claudeResponse.ContentBlock.Name,
				Arguments: json.RawMessage(`{}`),
			}
			return send(dto.ResponsesStreamResponse{
				Type:        dto.ResponsesOutputTypeItemAdded,
				OutputIndex: common.GetPointer(index),
				Item:        item,
			})
		}
	case "content_block_delta":
		if claudeResponse.Delta == nil {
			return nil
		}
		index := claudeResponse.GetIndex()
		if claudeResponse.Delta.Text != nil {
			return send(dto.ResponsesStreamResponse{
				Type:         "response.output_text.delta",
				OutputIndex:  common.GetPointer(index),
				ContentIndex: common.GetPointer(0),
				Delta:        *claudeResponse.Delta.Text,
			})
		}
		if claudeResponse.Delta.PartialJson != nil {
			return send(dto.ResponsesStreamResponse{
				Type:        "response.function_call_arguments.delta",
				OutputIndex: common.GetPointer(index),
				Delta:       *claudeResponse.Delta.PartialJson,
			})
		}
	case "content_block_stop":
		index := claudeResponse.GetIndex()
		return send(dto.ResponsesStreamResponse{
			Type:        dto.ResponsesOutputTypeItemDone,
			OutputIndex: common.GetPointer(index),
		})
	case "message_delta":
		return nil
	case "message_stop":
		completed := &dto.OpenAIResponsesResponse{
			ID:        responseID,
			Object:    "response",
			CreatedAt: int(claudeInfo.Created),
			Status:    json.RawMessage(`"completed"`),
			Model:     response.Model,
			Usage:     buildResponsesUsageFromClaudeUsage(claudeInfo.Usage),
		}
		return send(dto.ResponsesStreamResponse{
			Type:     "response.completed",
			Response: completed,
		})
	}

	return nil
}

type ClaudeResponseInfo struct {
	ResponseId   string
	Created      int64
	Model        string
	ResponseText strings.Builder
	Usage        *dto.Usage
	Done         bool
}

func cacheCreationTokensForOpenAIUsage(usage *dto.Usage) int {
	if usage == nil {
		return 0
	}
	splitCacheCreationTokens := usage.ClaudeCacheCreation5mTokens + usage.ClaudeCacheCreation1hTokens
	if splitCacheCreationTokens == 0 {
		return usage.PromptTokensDetails.CachedCreationTokens
	}
	if usage.PromptTokensDetails.CachedCreationTokens > splitCacheCreationTokens {
		return usage.PromptTokensDetails.CachedCreationTokens
	}
	return splitCacheCreationTokens
}

func buildOpenAIStyleUsageFromClaudeUsage(usage *dto.Usage) dto.Usage {
	if usage == nil {
		return dto.Usage{}
	}
	clone := *usage
	clone.ClaudeCacheCreation5mTokens, clone.ClaudeCacheCreation1hTokens = service.NormalizeCacheCreationSplit(
		usage.PromptTokensDetails.CachedCreationTokens,
		usage.ClaudeCacheCreation5mTokens,
		usage.ClaudeCacheCreation1hTokens,
	)
	cacheCreationTokens := cacheCreationTokensForOpenAIUsage(usage)
	totalInputTokens := usage.PromptTokens + usage.PromptTokensDetails.CachedTokens + cacheCreationTokens
	clone.PromptTokens = totalInputTokens
	clone.InputTokens = totalInputTokens
	clone.TotalTokens = totalInputTokens + usage.CompletionTokens
	clone.UsageSemantic = "openai"
	clone.UsageSource = "anthropic"
	return clone
}

func buildMessageDeltaPatchUsage(claudeResponse *dto.ClaudeResponse, claudeInfo *ClaudeResponseInfo) *dto.ClaudeUsage {
	usage := &dto.ClaudeUsage{}
	if claudeResponse != nil && claudeResponse.Usage != nil {
		*usage = *claudeResponse.Usage
	}

	if claudeInfo == nil || claudeInfo.Usage == nil {
		return usage
	}

	if usage.InputTokens == 0 && claudeInfo.Usage.PromptTokens > 0 {
		usage.InputTokens = claudeInfo.Usage.PromptTokens
	}
	if usage.CacheReadInputTokens == 0 && claudeInfo.Usage.PromptTokensDetails.CachedTokens > 0 {
		usage.CacheReadInputTokens = claudeInfo.Usage.PromptTokensDetails.CachedTokens
	}
	if usage.CacheCreationInputTokens == 0 && claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens > 0 {
		usage.CacheCreationInputTokens = claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens
	}
	cacheCreation5m := 0
	cacheCreation1h := 0
	if usage.CacheCreation != nil {
		cacheCreation5m = usage.CacheCreation.Ephemeral5mInputTokens
		cacheCreation1h = usage.CacheCreation.Ephemeral1hInputTokens
	} else {
		cacheCreation5m = claudeInfo.Usage.ClaudeCacheCreation5mTokens
		cacheCreation1h = claudeInfo.Usage.ClaudeCacheCreation1hTokens
	}
	cacheCreation5m, cacheCreation1h = service.NormalizeCacheCreationSplit(
		usage.CacheCreationInputTokens,
		cacheCreation5m,
		cacheCreation1h,
	)
	if usage.CacheCreation == nil && (cacheCreation5m > 0 || cacheCreation1h > 0) {
		usage.CacheCreation = &dto.ClaudeCacheCreationUsage{}
	}
	if usage.CacheCreation != nil {
		usage.CacheCreation.Ephemeral5mInputTokens = cacheCreation5m
		usage.CacheCreation.Ephemeral1hInputTokens = cacheCreation1h
	}
	return usage
}

func shouldSkipClaudeMessageDeltaUsagePatch(info *relaycommon.RelayInfo) bool {
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled {
		return true
	}
	if info == nil {
		return false
	}
	return info.ChannelSetting.PassThroughBodyEnabled
}

func patchClaudeMessageDeltaUsageData(data string, usage *dto.ClaudeUsage) string {
	if data == "" || usage == nil {
		return data
	}

	data = setMessageDeltaUsageInt(data, "usage.input_tokens", usage.InputTokens)
	data = setMessageDeltaUsageInt(data, "usage.cache_read_input_tokens", usage.CacheReadInputTokens)
	data = setMessageDeltaUsageInt(data, "usage.cache_creation_input_tokens", usage.CacheCreationInputTokens)

	if usage.CacheCreation != nil {
		data = setMessageDeltaUsageInt(data, "usage.cache_creation.ephemeral_5m_input_tokens", usage.CacheCreation.Ephemeral5mInputTokens)
		data = setMessageDeltaUsageInt(data, "usage.cache_creation.ephemeral_1h_input_tokens", usage.CacheCreation.Ephemeral1hInputTokens)
	}

	return data
}

func setMessageDeltaUsageInt(data string, path string, localValue int) string {
	if localValue <= 0 {
		return data
	}

	upstreamValue := gjson.Get(data, path)
	if upstreamValue.Exists() && upstreamValue.Int() > 0 {
		return data
	}

	patchedData, err := sjson.Set(data, path, localValue)
	if err != nil {
		return data
	}
	return patchedData
}

func FormatClaudeResponseInfo(claudeResponse *dto.ClaudeResponse, oaiResponse *dto.ChatCompletionsStreamResponse, claudeInfo *ClaudeResponseInfo) bool {
	if claudeInfo == nil {
		return false
	}
	if claudeInfo.Usage == nil {
		claudeInfo.Usage = &dto.Usage{}
	}
	if claudeResponse.Type == "message_start" {
		if claudeResponse.Message != nil {
			claudeInfo.ResponseId = claudeResponse.Message.Id
			claudeInfo.Model = claudeResponse.Message.Model
		}

		// message_start, 获取usage
		if claudeResponse.Message != nil && claudeResponse.Message.Usage != nil {
			claudeInfo.Usage.PromptTokens = claudeResponse.Message.Usage.InputTokens
			claudeInfo.Usage.UsageSemantic = "anthropic"
			claudeInfo.Usage.PromptTokensDetails.CachedTokens = claudeResponse.Message.Usage.CacheReadInputTokens
			claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens = claudeResponse.Message.Usage.CacheCreationInputTokens
			claudeInfo.Usage.ClaudeCacheCreation5mTokens = claudeResponse.Message.Usage.GetCacheCreation5mTokens()
			claudeInfo.Usage.ClaudeCacheCreation1hTokens = claudeResponse.Message.Usage.GetCacheCreation1hTokens()
			claudeInfo.Usage.CompletionTokens = claudeResponse.Message.Usage.OutputTokens
		}
	} else if claudeResponse.Type == "content_block_delta" {
		if claudeResponse.Delta != nil {
			if claudeResponse.Delta.Text != nil {
				claudeInfo.ResponseText.WriteString(*claudeResponse.Delta.Text)
			}
			if claudeResponse.Delta.Thinking != nil {
				claudeInfo.ResponseText.WriteString(*claudeResponse.Delta.Thinking)
			}
		}
	} else if claudeResponse.Type == "message_delta" {
		// 最终的usage获取
		if claudeResponse.Usage != nil {
			claudeInfo.Usage.UsageSemantic = "anthropic"
			if claudeResponse.Usage.InputTokens > 0 {
				// 不叠加，只取最新的
				claudeInfo.Usage.PromptTokens = claudeResponse.Usage.InputTokens
			}
			if claudeResponse.Usage.CacheReadInputTokens > 0 {
				claudeInfo.Usage.PromptTokensDetails.CachedTokens = claudeResponse.Usage.CacheReadInputTokens
			}
			if claudeResponse.Usage.CacheCreationInputTokens > 0 {
				claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens = claudeResponse.Usage.CacheCreationInputTokens
			}
			if cacheCreation5m := claudeResponse.Usage.GetCacheCreation5mTokens(); cacheCreation5m > 0 {
				claudeInfo.Usage.ClaudeCacheCreation5mTokens = cacheCreation5m
			}
			if cacheCreation1h := claudeResponse.Usage.GetCacheCreation1hTokens(); cacheCreation1h > 0 {
				claudeInfo.Usage.ClaudeCacheCreation1hTokens = cacheCreation1h
			}
			if claudeResponse.Usage.OutputTokens > 0 {
				claudeInfo.Usage.CompletionTokens = claudeResponse.Usage.OutputTokens
			}
			claudeInfo.Usage.TotalTokens = claudeInfo.Usage.PromptTokens + claudeInfo.Usage.CompletionTokens
		}

		// 判断是否完整
		claudeInfo.Done = true
	} else if claudeResponse.Type == "content_block_start" {
	} else {
		return false
	}
	if oaiResponse != nil {
		oaiResponse.Id = claudeInfo.ResponseId
		oaiResponse.Created = claudeInfo.Created
		oaiResponse.Model = claudeInfo.Model
	}
	return true
}

func HandleStreamResponseData(c *gin.Context, info *relaycommon.RelayInfo, claudeInfo *ClaudeResponseInfo, data string) *types.NewAPIError {
	var claudeResponse dto.ClaudeResponse
	err := common.UnmarshalJsonStr(data, &claudeResponse)
	if err != nil {
		common.SysLog("error unmarshalling stream response: " + err.Error())
		return types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	if claudeError := claudeResponse.GetClaudeError(); claudeError != nil && claudeError.Type != "" {
		return types.WithClaudeError(*claudeError, http.StatusInternalServerError)
	}
	if claudeResponse.StopReason != "" {
		maybeMarkClaudeRefusal(c, claudeResponse.StopReason)
	}
	if claudeResponse.Delta != nil && claudeResponse.Delta.StopReason != nil {
		maybeMarkClaudeRefusal(c, *claudeResponse.Delta.StopReason)
	}
	if info.RelayFormat == types.RelayFormatClaude {
		FormatClaudeResponseInfo(&claudeResponse, nil, claudeInfo)

		if claudeResponse.Type == "message_start" {
			// message_start, 获取usage
			if claudeResponse.Message != nil {
				info.UpstreamModelName = claudeResponse.Message.Model
			}
		} else if claudeResponse.Type == "message_delta" {
			// 确保 message_delta 的 usage 包含完整的 input_tokens 和 cache 相关字段
			// 解决 AWS Bedrock 等上游返回的 message_delta 缺少这些字段的问题
			if !shouldSkipClaudeMessageDeltaUsagePatch(info) {
				data = patchClaudeMessageDeltaUsageData(data, buildMessageDeltaPatchUsage(&claudeResponse, claudeInfo))
			}
		}
		helper.ClaudeChunkData(c, claudeResponse, data)
	} else if info.RelayFormat == types.RelayFormatOpenAI {
		response := StreamResponseClaude2OpenAI(&claudeResponse)

		if !FormatClaudeResponseInfo(&claudeResponse, response, claudeInfo) {
			return nil
		}

		err = helper.ObjectData(c, response)
		if err != nil {
			logger.LogError(c, "send_stream_response_failed: "+err.Error())
		}
	} else if info.RelayFormat == types.RelayFormatOpenAIResponses {
		return handleClaudeStreamAsResponses(c, info, claudeInfo, &claudeResponse)
	}
	return nil
}

func HandleStreamFinalResponse(c *gin.Context, info *relaycommon.RelayInfo, claudeInfo *ClaudeResponseInfo) {
	if claudeInfo.Usage.PromptTokens == 0 {
		//上游出错
	}
	if claudeInfo.Usage.CompletionTokens == 0 || !claudeInfo.Done {
		if common.DebugEnabled {
			common.SysLog("claude response usage is not complete, maybe upstream error")
		}
		// 只补缺失字段，不整份覆盖——保留 message_start 已拿到的 cache 字段
		fallback := service.ResponseText2Usage(c, claudeInfo.ResponseText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
		if claudeInfo.Usage.CompletionTokens == 0 ||
			(!claudeInfo.Done && fallback.CompletionTokens > claudeInfo.Usage.CompletionTokens) {
			claudeInfo.Usage.CompletionTokens = fallback.CompletionTokens
		}
		if claudeInfo.Usage.PromptTokens == 0 {
			claudeInfo.Usage.PromptTokens = fallback.PromptTokens
		}
		claudeInfo.Usage.TotalTokens = claudeInfo.Usage.PromptTokens + claudeInfo.Usage.CompletionTokens
	}
	if claudeInfo.Usage != nil {
		claudeInfo.Usage.UsageSemantic = "anthropic"
	}

	if info.RelayFormat == types.RelayFormatClaude {
		//
	} else if info.RelayFormat == types.RelayFormatOpenAI {
		if info.ShouldIncludeUsage {
			openAIUsage := buildOpenAIStyleUsageFromClaudeUsage(claudeInfo.Usage)
			response := helper.GenerateFinalUsageResponse(claudeInfo.ResponseId, claudeInfo.Created, info.UpstreamModelName, openAIUsage)
			err := helper.ObjectData(c, response)
			if err != nil {
				common.SysLog("send final response failed: " + err.Error())
			}
		}
		helper.Done(c)
	}
}

func ClaudeStreamHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	claudeInfo := &ClaudeResponseInfo{
		ResponseId:   helper.GetResponseID(c),
		Created:      common.GetTimestamp(),
		Model:        info.UpstreamModelName,
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}
	var err *types.NewAPIError
	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		err = HandleStreamResponseData(c, info, claudeInfo, data)
		if err != nil {
			sr.Stop(err)
		}
	})
	if err != nil {
		return nil, err
	}

	HandleStreamFinalResponse(c, info, claudeInfo)
	return claudeInfo.Usage, nil
}

func HandleClaudeResponseData(c *gin.Context, info *relaycommon.RelayInfo, claudeInfo *ClaudeResponseInfo, httpResp *http.Response, data []byte) *types.NewAPIError {
	var claudeResponse dto.ClaudeResponse
	err := common.Unmarshal(data, &claudeResponse)
	if err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	if claudeError := claudeResponse.GetClaudeError(); claudeError != nil && claudeError.Type != "" {
		return types.WithClaudeError(*claudeError, http.StatusInternalServerError)
	}
	maybeMarkClaudeRefusal(c, claudeResponse.StopReason)
	if claudeInfo.Usage == nil {
		claudeInfo.Usage = &dto.Usage{}
	}
	if claudeResponse.Usage != nil {
		claudeInfo.Usage.PromptTokens = claudeResponse.Usage.InputTokens
		claudeInfo.Usage.CompletionTokens = claudeResponse.Usage.OutputTokens
		claudeInfo.Usage.TotalTokens = claudeResponse.Usage.InputTokens + claudeResponse.Usage.OutputTokens
		claudeInfo.Usage.UsageSemantic = "anthropic"
		claudeInfo.Usage.PromptTokensDetails.CachedTokens = claudeResponse.Usage.CacheReadInputTokens
		claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens = claudeResponse.Usage.CacheCreationInputTokens
		claudeInfo.Usage.ClaudeCacheCreation5mTokens = claudeResponse.Usage.GetCacheCreation5mTokens()
		claudeInfo.Usage.ClaudeCacheCreation1hTokens = claudeResponse.Usage.GetCacheCreation1hTokens()
	}
	var responseData []byte
	switch info.RelayFormat {
	case types.RelayFormatOpenAI:
		openaiResponse := ResponseClaude2OpenAI(&claudeResponse)
		openaiResponse.Usage = buildOpenAIStyleUsageFromClaudeUsage(claudeInfo.Usage)
		responseData, err = json.Marshal(openaiResponse)
		if err != nil {
			return types.NewError(err, types.ErrorCodeBadResponseBody)
		}
	case types.RelayFormatClaude:
		responseData = data
	case types.RelayFormatOpenAIResponses:
		responsesResponse := ResponseClaude2OpenAIResponses(&claudeResponse, claudeInfo.Usage, claudeInfo.Created)
		responseData, err = common.Marshal(responsesResponse)
		if err != nil {
			return types.NewError(err, types.ErrorCodeBadResponseBody)
		}
	}

	if claudeResponse.Usage != nil && claudeResponse.Usage.ServerToolUse != nil && claudeResponse.Usage.ServerToolUse.WebSearchRequests > 0 {
		c.Set("claude_web_search_requests", claudeResponse.Usage.ServerToolUse.WebSearchRequests)
	}

	service.IOCopyBytesGracefully(c, httpResp, responseData)
	return nil
}

func ClaudeHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	claudeInfo := &ClaudeResponseInfo{
		ResponseId:   helper.GetResponseID(c),
		Created:      common.GetTimestamp(),
		Model:        info.UpstreamModelName,
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	logger.LogDebug(c, "responseBody: %s", responseBody)
	handleErr := HandleClaudeResponseData(c, info, claudeInfo, resp, responseBody)
	if handleErr != nil {
		return nil, handleErr
	}
	return claudeInfo.Usage, nil
}

func mapToolChoice(toolChoice any, parallelToolCalls *bool) *dto.ClaudeToolChoice {
	var claudeToolChoice *dto.ClaudeToolChoice

	// 处理 tool_choice 字符串值
	if toolChoiceStr, ok := toolChoice.(string); ok {
		switch toolChoiceStr {
		case "auto":
			claudeToolChoice = &dto.ClaudeToolChoice{
				Type: "auto",
			}
		case "required":
			claudeToolChoice = &dto.ClaudeToolChoice{
				Type: "any",
			}
		case "none":
			claudeToolChoice = &dto.ClaudeToolChoice{
				Type: "none",
			}
		}
	} else if toolChoiceMap, ok := toolChoice.(map[string]interface{}); ok {
		// 处理 tool_choice 对象值
		if function, ok := toolChoiceMap["function"].(map[string]interface{}); ok {
			if toolName, ok := function["name"].(string); ok {
				claudeToolChoice = &dto.ClaudeToolChoice{
					Type: "tool",
					Name: toolName,
				}
			}
		}
	}

	// 处理 parallel_tool_calls
	if parallelToolCalls != nil {
		if claudeToolChoice == nil {
			// 如果没有 tool_choice，但有 parallel_tool_calls，创建默认的 auto 类型
			claudeToolChoice = &dto.ClaudeToolChoice{
				Type: "auto",
			}
		}

		// Anthropic schema: tool_choice.type=none does not accept extra fields.
		// When tools are disabled, parallel_tool_calls is irrelevant, so we drop it.
		if claudeToolChoice.Type != "none" {
			// 如果 parallel_tool_calls 为 true，则 disable_parallel_tool_use 为 false
			claudeToolChoice.DisableParallelToolUse = !*parallelToolCalls
		}
	}

	return claudeToolChoice
}
