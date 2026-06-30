package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestApplyDynamicModelMappingSelectsFirstMatchingRuleAndTransformsBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	storage, err := common.CreateBodyStorage([]byte(`{
		"model":"omni-flash",
		"prompt":"edit this",
		"input_reference":"https://example.com/a.png"
	}`))
	require.NoError(t, err)
	c.Set(common.KeyBodyStorage, storage)

	info := &relaycommon.RelayInfo{
		OriginModelName: "omni-flash",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "omni-flash",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				DynamicModelMapping: []dto.DynamicModelMappingRule{
					{
						From: "omni-flash",
						To:   "omni_flash_1-2",
						When: []dto.DynamicModelMappingCondition{
							{Path: "input_reference", Op: "len_eq", Value: float64(2)},
						},
					},
					{
						From: "omni-flash",
						To:   "omni_flash_1-1",
						When: []dto.DynamicModelMappingCondition{
							{Path: "input_reference", Op: "len_eq", Value: float64(1)},
						},
						FieldTransforms: []dto.DynamicFieldTransform{
							{Path: "input_reference", To: "array"},
						},
					},
					{
						From: "omni-flash",
						To:   "omni_flash",
					},
				},
			},
		},
	}

	matched, err := ApplyDynamicModelMapping(c, info, nil)
	require.NoError(t, err)
	require.True(t, matched)
	assert.Equal(t, "omni_flash_1-1", info.UpstreamModelName)
	assert.True(t, info.IsModelMapped)

	require.NoError(t, ApplyDynamicFieldTransformsToRequestBody(c, info))
	nextBody, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	body, err := nextBody.Bytes()
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model":"omni-flash",
		"prompt":"edit this",
		"input_reference":["https://example.com/a.png"]
	}`, string(body))
}

func TestApplyDynamicModelMappingMovesConvertedValueToTargetPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	storage, err := common.CreateBodyStorage([]byte(`{
		"model":"veo-3.1",
		"prompt":"animate this",
		"input_reference":"https://example.com/a.png"
	}`))
	require.NoError(t, err)
	c.Set(common.KeyBodyStorage, storage)

	info := &relaycommon.RelayInfo{
		OriginModelName: "veo-3.1",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "veo-3.1",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				DynamicModelMapping: []dto.DynamicModelMappingRule{
					{
						From: "veo-3.1",
						To:   "veo_3_1-components",
						When: []dto.DynamicModelMappingCondition{
							{Path: "input_reference", Op: "len_gte", Value: float64(1)},
						},
						FieldTransforms: []dto.DynamicFieldTransform{
							{
								Path:       "input_reference",
								To:         "array",
								TargetPath: "images",
								Mode:       "move",
							},
						},
					},
				},
			},
		},
	}

	matched, err := ApplyDynamicModelMapping(c, info, nil)
	require.NoError(t, err)
	require.True(t, matched)
	assert.Equal(t, "veo_3_1-components", info.UpstreamModelName)

	require.NoError(t, ApplyDynamicFieldTransformsToRequestBody(c, info))
	nextBody, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	body, err := nextBody.Bytes()
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model":"veo-3.1",
		"prompt":"animate this",
		"images":["https://example.com/a.png"]
	}`, string(body))
	assert.False(t, gjson.GetBytes(body, "input_reference").Exists())
}

func TestModelMappedHelperFallsBackToStaticMappingWhenDynamicRulesEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("model_mapping", `{"veo-3.1":"veo_3_1"}`)

	info := &relaycommon.RelayInfo{
		OriginModelName: "veo-3.1",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "veo-3.1",
		},
	}

	require.NoError(t, ModelMappedHelper(c, info, nil))
	assert.Equal(t, "veo_3_1", info.UpstreamModelName)
	assert.True(t, info.IsModelMapped)
}
