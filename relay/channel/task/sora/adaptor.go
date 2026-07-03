package sora

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID                 string        `json:"id"`
	TaskID             string        `json:"task_id,omitempty"` //兼容旧接口
	Data               *responseTask `json:"data,omitempty"`
	Object             string        `json:"object"`
	Model              string        `json:"model"`
	Status             string        `json:"status"`
	Progress           int           `json:"progress"`
	CreatedAt          int64         `json:"created_at"`
	CompletedAt        int64         `json:"completed_at,omitempty"`
	ExpiresAt          int64         `json:"expires_at,omitempty"`
	Seconds            string        `json:"seconds,omitempty"`
	Size               string        `json:"size,omitempty"`
	RemixedFromVideoID string        `json:"remixed_from_video_id,omitempty"`
	Error              *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
	URL       string   `json:"url,omitempty"`
	VideoURL  string   `json:"video_url,omitempty"`
	ResultURL string   `json:"result_url,omitempty"`
	Output    []string `json:"output,omitempty"`
	Video     *struct {
		URL string `json:"url"`
	} `json:"video,omitempty"`
	Usage *taskUsage `json:"usage,omitempty"`
}

type taskUsage struct {
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

func (r responseTask) resultURL() string {
	if r.Video != nil && strings.TrimSpace(r.Video.URL) != "" {
		return strings.TrimSpace(r.Video.URL)
	}
	for _, url := range []string{r.ResultURL, r.VideoURL, r.URL} {
		if strings.TrimSpace(url) != "" {
			return strings.TrimSpace(url)
		}
	}
	for _, url := range r.Output {
		if strings.TrimSpace(url) != "" {
			return strings.TrimSpace(url)
		}
	}
	return ""
}

func (r responseTask) usage() taskUsage {
	if r.Usage != nil {
		return *r.Usage
	}
	if r.Data != nil {
		return r.Data.usage()
	}
	return taskUsage{}
}

func (r responseTask) hasUsage() bool {
	usage := r.usage()
	return usage.TotalTokens > 0 || usage.CompletionTokens > 0
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	// 存储原始请求到 context，与 ValidateMultipartDirect 路径保持一致
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.Action == constant.TaskActionRemix {
		return validateRemixRequest(c)
	}
	return relaycommon.ValidateMultipartDirect(c, info)
}

// EstimateBilling 根据用户请求的 seconds 和 size 计算 OtherRatios。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	// remix 路径的 OtherRatios 已在 ResolveOriginTask 中设置
	if info.Action == constant.TaskActionRemix {
		return nil
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		seconds = 4
	}

	size := req.Size
	if size == "" {
		size = "720x1280"
	}

	ratios := map[string]float64{
		"seconds": float64(seconds),
		"size":    1,
	}
	if size == "1792x1024" || size == "1024x1792" {
		ratios["size"] = 1.666667
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, info.OriginTaskID), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			bodyMap["model"] = info.UpstreamModelName
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("model", info.UpstreamModelName)
		for key, values := range formData.Value {
			if key == "model" {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					// Re-open after sniffing so the full content is copied below
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.ReaderOnly(storage), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Sora response
	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// 使用公开 task_xxxx ID 返回给客户端
	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	c.JSON(http.StatusOK, dResp)
	return upstreamID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}
	billingContext, _ := body["billing_context"].(*model.TaskBillingContext)

	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return a.addVideoGenerationsUsageIfNeeded(client, resp, baseUrl, key, taskID, billingContext)
}

func shouldFetchVideoGenerationsUsage(bc *model.TaskBillingContext) bool {
	if bc == nil {
		return false
	}
	if bc.PerCallBilling || bc.ResolvedPerRequestPricing != nil {
		return false
	}
	return bc.ModelRatio > 0 && bc.ModelPrice < 0
}

func (a *TaskAdaptor) addVideoGenerationsUsageIfNeeded(client *http.Client, resp *http.Response, baseUrl, key, taskID string, billingContext *model.TaskBillingContext) (*http.Response, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	var resTask responseTask
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		common.SysLog(fmt.Sprintf("sora video fallback trace: task=%s parse_primary_failed err=%v primary_body_len=%d", taskID, err, len(respBody)))
		return resp, nil
	}
	if resTask.Status == "" && resTask.Data != nil {
		resTask = *resTask.Data
	}
	status := strings.ToLower(strings.TrimSpace(resTask.Status))
	common.SysLog(fmt.Sprintf(
		"sora video fallback trace: task=%s fallback_enabled=%t primary_status=%s primary_has_usage=%t primary_body_len=%d",
		taskID,
		shouldFetchVideoGenerationsUsage(billingContext),
		status,
		resTask.hasUsage(),
		len(respBody),
	))
	if shouldFetchVideoGenerationsFailure(status, resTask, billingContext) {
		if merged, ok := a.mergeVideoGenerationsFailureIfNeeded(client, resp, respBody, baseUrl, key, taskID); ok {
			return merged, nil
		}
	}
	if !shouldFetchVideoGenerationsUsage(billingContext) {
		return resp, nil
	}
	if status != "completed" && status != "done" && status != "succeeded" && status != "success" && status != "unknown" {
		return resp, nil
	}
	if resTask.hasUsage() {
		return resp, nil
	}

	usageURI := fmt.Sprintf("%s/v1/video/generations/%s", baseUrl, taskID)
	usageReq, err := http.NewRequest(http.MethodGet, usageURI, nil)
	if err != nil {
		return resp, nil
	}
	usageReq.Header.Set("Authorization", "Bearer "+key)

	usageResp, err := client.Do(usageReq)
	if err != nil {
		common.SysLog(fmt.Sprintf("sora video fallback trace: task=%s usage_request_failed err=%v", taskID, err))
		return resp, nil
	}
	defer usageResp.Body.Close()

	usageBody, err := io.ReadAll(usageResp.Body)
	if err != nil {
		common.SysLog(fmt.Sprintf("sora video fallback trace: task=%s read_usage_failed err=%v", taskID, err))
		return resp, nil
	}
	usage := extractVideoGenerationsUsage(usageBody)
	usageStatus := strings.ToLower(strings.TrimSpace(firstGJSONString(usageBody, "data.status", "status")))
	resultURL := strings.TrimSpace(firstGJSONString(usageBody, "data.result_url", "result_url", "data.data.content.video_url", "data.content.video_url"))
	common.SysLog(fmt.Sprintf(
		"sora video fallback trace: task=%s usage_status_code=%d usage_status=%s total_tokens=%d completion_tokens=%d result_url_set=%t usage_body_len=%d",
		taskID,
		usageResp.StatusCode,
		usageStatus,
		usage.TotalTokens,
		usage.CompletionTokens,
		resultURL != "",
		len(usageBody),
	))
	if usage.TotalTokens <= 0 && usage.CompletionTokens <= 0 {
		var bodyMap map[string]any
		if err := common.Unmarshal(respBody, &bodyMap); err != nil {
			return resp, nil
		}
		bodyMap["status"] = "in_progress"
		bodyMap["progress"] = 99
		pendingBody, err := common.Marshal(bodyMap)
		if err != nil {
			return resp, nil
		}
		resp.Body = io.NopCloser(bytes.NewReader(pendingBody))
		resp.ContentLength = int64(len(pendingBody))
		return resp, nil
	}
	var bodyMap map[string]any
	if err := common.Unmarshal(respBody, &bodyMap); err != nil {
		return resp, nil
	}
	bodyMap["usage"] = usage
	if resultURL != "" {
		bodyMap["result_url"] = resultURL
	}
	if usageStatus == "completed" || usageStatus == "done" || usageStatus == "succeeded" || usageStatus == "success" {
		bodyMap["status"] = "completed"
		bodyMap["progress"] = 100
	}
	mergedBody, err := common.Marshal(bodyMap)
	if err != nil {
		return resp, nil
	}
	resp.Body = io.NopCloser(bytes.NewReader(mergedBody))
	resp.ContentLength = int64(len(mergedBody))
	return resp, nil
}

func shouldFetchVideoGenerationsFailure(status string, task responseTask, bc *model.TaskBillingContext) bool {
	switch status {
	case "queued", "pending", "processing", "in_progress", "unknown":
	default:
		return false
	}
	modelName := strings.ToLower(strings.TrimSpace(task.Model))
	if strings.HasPrefix(modelName, "grok-") {
		return true
	}
	if bc == nil {
		return false
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(bc.OriginModelName)), "grok-")
}

func (a *TaskAdaptor) mergeVideoGenerationsFailureIfNeeded(client *http.Client, resp *http.Response, respBody []byte, baseUrl, key, taskID string) (*http.Response, bool) {
	usageURI := fmt.Sprintf("%s/v1/video/generations/%s", baseUrl, taskID)
	usageReq, err := http.NewRequest(http.MethodGet, usageURI, nil)
	if err != nil {
		return resp, false
	}
	usageReq.Header.Set("Authorization", "Bearer "+key)

	usageResp, err := client.Do(usageReq)
	if err != nil {
		common.SysLog(fmt.Sprintf("sora video failure fallback trace: task=%s request_failed err=%v", taskID, err))
		return resp, false
	}
	defer usageResp.Body.Close()

	usageBody, err := io.ReadAll(usageResp.Body)
	if err != nil {
		common.SysLog(fmt.Sprintf("sora video failure fallback trace: task=%s read_failed err=%v", taskID, err))
		return resp, false
	}
	status := strings.ToLower(strings.TrimSpace(firstGJSONString(usageBody, "status", "data.status")))
	common.SysLog(fmt.Sprintf(
		"sora video failure fallback trace: task=%s usage_status_code=%d usage_status=%s usage_body_len=%d",
		taskID,
		usageResp.StatusCode,
		status,
		len(usageBody),
	))
	if status != "failed" && status != "failure" && status != "cancelled" && status != "canceled" {
		return resp, false
	}

	var bodyMap map[string]any
	if err := common.Unmarshal(respBody, &bodyMap); err != nil {
		return resp, false
	}
	reason := strings.TrimSpace(firstGJSONString(usageBody, "error.message", "data.fail_reason", "data.error.message", "message"))
	if reason == "" {
		reason = "task failed"
	}
	bodyMap["status"] = "failed"
	bodyMap["progress"] = 100
	bodyMap["error"] = map[string]any{
		"message": reason,
		"code":    "task_failed",
	}
	mergedBody, err := common.Marshal(bodyMap)
	if err != nil {
		return resp, false
	}
	resp.Body = io.NopCloser(bytes.NewReader(mergedBody))
	resp.ContentLength = int64(len(mergedBody))
	return resp, true
}

func extractVideoGenerationsUsage(body []byte) taskUsage {
	paths := []struct {
		completion string
		total      string
	}{
		{"data.data.usage.completion_tokens", "data.data.usage.total_tokens"},
		{"data.usage.completion_tokens", "data.usage.total_tokens"},
		{"usage.completion_tokens", "usage.total_tokens"},
	}
	for _, p := range paths {
		usage := taskUsage{
			CompletionTokens: int(gjson.GetBytes(body, p.completion).Int()),
			TotalTokens:      int(gjson.GetBytes(body, p.total).Int()),
		}
		if usage.CompletionTokens > 0 || usage.TotalTokens > 0 {
			return usage
		}
	}
	return taskUsage{}
}

func firstGJSONString(body []byte, paths ...string) string {
	for _, path := range paths {
		if value := gjson.GetBytes(body, path); value.Exists() {
			return value.String()
		}
	}
	return ""
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	if resTask.Status == "" && resTask.Data != nil {
		resTask = *resTask.Data
	}
	usage := resTask.usage()

	taskResult := relaycommon.TaskInfo{
		Code:             0,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}

	switch strings.ToLower(strings.TrimSpace(resTask.Status)) {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "unknown":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "99%"
	case "completed", "done", "succeeded", "success":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = resTask.resultURL()
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	data := task.Data
	var err error
	if data, err = sjson.SetBytes(data, "id", task.TaskID); err != nil {
		return nil, errors.Wrap(err, "set id failed")
	}
	return data, nil
}
