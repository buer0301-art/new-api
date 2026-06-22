package sora

import (
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
