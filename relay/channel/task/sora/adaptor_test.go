package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaskResultDoneVideoURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id": "task_VknNypW89I3atmXDI9dhibtk29wjPC7l",
		"model": "grok-image-video",
		"status": "done",
		"progress": 100,
		"video": {
			"url": "https://vidgen.x.ai/xai-video.mp4",
			"duration": 15
		}
	}`)

	got, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, got.Status)
	assert.Equal(t, "https://vidgen.x.ai/xai-video.mp4", got.Url)
}

func TestParseTaskResultDoneXAICompatibilityFields(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id": "task_HGhBhRKxvWRvdtHv9zD1p7hcMISZ8Hki",
		"url": "https://vidgen.x.ai/xai-video-top.mp4",
		"model": "grok-image-video",
		"status": "done",
		"progress": 100,
		"video_url": "https://vidgen.x.ai/xai-video-video-url.mp4",
		"result_url": "https://vidgen.x.ai/xai-video-result-url.mp4",
		"output": [
			"https://vidgen.x.ai/xai-video-output.mp4"
		]
	}`)

	got, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, got.Status)
	assert.Equal(t, "https://vidgen.x.ai/xai-video-result-url.mp4", got.Url)
}

func TestParseTaskResultDoneWrappedXAIResponse(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"data": {
			"id": "task_HGhBhRKxvWRvdtHv9zD1p7hcMISZ8Hki",
			"model": "grok-image-video",
			"status": "done",
			"progress": 100,
			"video": {
				"url": "https://vidgen.x.ai/xai-video.mp4",
				"duration": 15
			}
		}
	}`)

	got, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, got.Status)
	assert.Equal(t, "https://vidgen.x.ai/xai-video.mp4", got.Url)
}

func TestParseTaskResultWrappedVideoGenerationsUsage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"code": "success",
		"data": {
			"status": "SUCCESS",
			"data": {
				"usage": {
					"completion_tokens": 87300,
					"total_tokens": 87300
				}
			}
		}
	}`)

	got, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, got.Status)
	assert.Equal(t, 87300, got.CompletionTokens)
	assert.Equal(t, 87300, got.TotalTokens)
}

func TestParseTaskResultUnknownKeepsInProgress(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id": "task_upstream",
		"status": "unknown",
		"progress": 0,
		"metadata": {
			"url": ""
		}
	}`)

	got, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, got.Status)
	assert.Equal(t, "99%", got.Progress)
}

func TestFetchTaskFallsBackToVideoGenerationsWhenCompletedWithoutUsage(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/videos/task_upstream":
			_, _ = io.WriteString(w, `{"id":"task_upstream","status":"completed","progress":100}`)
		case "/v1/video/generations/task_upstream":
			_, _ = io.WriteString(w, `{"code":"success","data":{"status":"SUCCESS","data":{"usage":{"total_tokens":87300}}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"billing_context": &model.TaskBillingContext{
			ModelPrice: -1,
			ModelRatio: 12.5,
		},
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	got, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, got.Status)
	assert.Equal(t, 87300, got.TotalTokens)
	assert.Equal(t, []string{"/v1/videos/task_upstream", "/v1/video/generations/task_upstream"}, paths)
}

func TestFetchTaskFallsBackToVideoGenerationsWithNumericWrappedID(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/videos/task_upstream":
			_, _ = io.WriteString(w, `{"id":"task_upstream","status":"completed","progress":100}`)
		case "/v1/video/generations/task_upstream":
			_, _ = io.WriteString(w, `{
				"code": "success",
				"data": {
					"id": 242,
					"task_id": "task_upstream",
					"status": "SUCCESS",
					"result_url": "https://example.com/video.mp4",
					"data": {
						"usage": {
							"completion_tokens": 108900,
							"total_tokens": 108900
						}
					}
				}
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"billing_context": &model.TaskBillingContext{
			ModelPrice: -1,
			ModelRatio: 12.5,
		},
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	got, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, got.Status)
	assert.Equal(t, "https://example.com/video.mp4", got.Url)
	assert.Equal(t, 108900, got.TotalTokens)
	assert.Equal(t, []string{"/v1/videos/task_upstream", "/v1/video/generations/task_upstream"}, paths)
}

func TestFetchTaskFallsBackToVideoGenerationsWhenVideoStatusUnknown(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/videos/task_upstream":
			_, _ = io.WriteString(w, `{"id":"task_upstream","status":"unknown","progress":0}`)
		case "/v1/video/generations/task_upstream":
			_, _ = io.WriteString(w, `{
				"code": "success",
				"data": {
					"task_id": "task_upstream",
					"status": "SUCCESS",
					"result_url": "https://example.com/video.mp4",
					"data": {
						"usage": {
							"completion_tokens": 108900,
							"total_tokens": 108900
						}
					}
				}
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"billing_context": &model.TaskBillingContext{
			ModelPrice: -1,
			ModelRatio: 12.5,
		},
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	got, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, got.Status)
	assert.Equal(t, "https://example.com/video.mp4", got.Url)
	assert.Equal(t, 108900, got.TotalTokens)
	assert.Equal(t, []string{"/v1/videos/task_upstream", "/v1/video/generations/task_upstream"}, paths)
}

func TestFetchTaskKeepsInProgressWhenCompletedUsageNotReady(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/videos/task_upstream":
			_, _ = io.WriteString(w, `{"id":"task_upstream","status":"completed","progress":100}`)
		case "/v1/video/generations/task_upstream":
			_, _ = io.WriteString(w, `{"code":"success","data":{"status":"SUCCESS","data":{}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"billing_context": &model.TaskBillingContext{
			ModelPrice: -1,
			ModelRatio: 12.5,
		},
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	got, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, got.Status)
	assert.Equal(t, "99%", got.Progress)
	assert.Equal(t, 0, got.TotalTokens)
	assert.Equal(t, []string{"/v1/videos/task_upstream", "/v1/video/generations/task_upstream"}, paths)
}

func TestFetchTaskSkipsVideoGenerationsFallbackForFixedPrice(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = io.WriteString(w, `{"id":"task_upstream","status":"completed","progress":100}`)
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"billing_context": &model.TaskBillingContext{
			ModelPrice: 0.3,
			ModelRatio: 0,
		},
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, []string{"/v1/videos/task_upstream"}, paths)
}
