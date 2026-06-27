package relay

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/setting/per_request_pricing"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveVideoPricingInputFromTaskRequestSoraDefaultsDuration(t *testing.T) {
	input := resolveVideoPricingInputFromTaskRequest("sora-2", relaycommon.TaskSubmitReq{
		Size: "720x1280",
	}, 1)

	require.Equal(t, 4, input.Duration)
	require.Empty(t, input.Seconds)
	require.Equal(t, "720x1280", input.Size)
}

func TestTaskModel2DtoIncludesRequestId(t *testing.T) {
	task := &model.Task{
		TaskID: "task_request_id",
		PrivateData: model.TaskPrivateData{
			RequestId: "req-task-123",
		},
	}

	dto := TaskModel2Dto(task)

	require.Equal(t, "req-task-123", dto.RequestId)
}

func TestResolveVideoPricingInputFromTaskRequestUsesMetadataDurationSeconds(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		want     int
	}{
		{name: "camel numeric", metadata: map[string]interface{}{"durationSeconds": float64(8)}, want: 8},
		{name: "snake string", metadata: map[string]interface{}{"duration_seconds": "12"}, want: 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := resolveVideoPricingInputFromTaskRequest("gemini-veo-test", relaycommon.TaskSubmitReq{
				Metadata: tt.metadata,
			}, 1)

			require.Equal(t, tt.want, input.Duration)
			require.Empty(t, input.Seconds)
		})
	}
}

func TestApplyVideoPerRequestPricingSkipsFreshResolveForRemixSnapshot(t *testing.T) {
	priceData := types.PriceData{
		ResolvedPerRequestPricing: &types.ResolvedPerRequestPricing{
			MediaType:  "video",
			Unit:       "second",
			Resolution: "4K",
			UnitPrice:  0.24,
			Quantity:   10,
			PriceUSD:   2.4,
			Quota:      1200000,
		},
		Quota: 1200000,
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "sora-2",
		PriceData:       priceData,
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionRemix,
		},
	}

	applied, taskErr := applyVideoPerRequestPricing(nil, info, priceData)

	require.Nil(t, taskErr)
	require.False(t, applied)
	require.Equal(t, priceData.ResolvedPerRequestPricing, info.PriceData.ResolvedPerRequestPricing)
	require.Equal(t, 1200000, info.PriceData.Quota)
}

func TestApplyVideoPerRequestPricingMapsResolutionAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupVideoResolutionPricingRules(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Size:    "1080p",
		Seconds: "5",
		Metadata: map[string]interface{}{
			"resolution": "2160p",
		},
	})

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-test-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}

	applied, taskErr := applyVideoPerRequestPricing(ctx, info, priceData)

	require.Nil(t, taskErr)
	require.True(t, applied)
	require.NotNil(t, info.PriceData.ResolvedPerRequestPricing)
	require.Equal(t, "4K", info.PriceData.ResolvedPerRequestPricing.Resolution)
	require.Equal(t, 0.24, info.PriceData.ResolvedPerRequestPricing.UnitPrice)
	require.Equal(t, 600000, info.PriceData.Quota)
}

func TestApplyVideoPerRequestPricingPerRequestUnitDoesNotMultiplySeconds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupVideoResolutionPricingRulesWithUnit(t, per_request_pricing.UnitRequest)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Size:    "1080p",
		Seconds: "15",
	})

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-test-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}

	applied, taskErr := applyVideoPerRequestPricing(ctx, info, priceData)

	require.Nil(t, taskErr)
	require.True(t, applied)
	require.NotNil(t, info.PriceData.ResolvedPerRequestPricing)
	require.Equal(t, per_request_pricing.UnitRequest, info.PriceData.ResolvedPerRequestPricing.Unit)
	require.Equal(t, float64(1), info.PriceData.ResolvedPerRequestPricing.Quantity)
	require.Equal(t, 40000, info.PriceData.Quota)
}

func TestApplyVideoPerRequestPricingWorksWithoutBaseModelPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupVideoResolutionPricingRulesWithUnit(t, per_request_pricing.UnitSecond)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	ctx.Set("group", "default")
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Model:   "video-test-model",
		Size:    "1080p",
		Seconds: "5",
	})

	info := &relaycommon.RelayInfo{
		OriginModelName: "video-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: helper.HandleGroupRatio(ctx, info),
	}

	applied, taskErr := applyVideoPerRequestPricing(ctx, info, priceData)

	require.Nil(t, taskErr)
	require.True(t, applied)
	require.NotNil(t, info.PriceData.ResolvedPerRequestPricing)
	require.Equal(t, 200000, info.PriceData.Quota)
}

func TestShouldApplyTaskOtherRatiosSkipsFixedPriceTask(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			UsePrice: true,
			Quota:    int(common.QuotaPerUnit),
			OtherRatios: map[string]float64{
				"seconds": 15,
				"size":    1,
			},
		},
	}

	require.False(t, shouldApplyTaskOtherRatios(info, "grok-imagine-video"))
}

func TestShouldApplyTaskOtherRatiosSkipsRatioPricedTask(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenAI,
		},
		PriceData: types.PriceData{
			UsePrice: false,
			Quota:    int(common.QuotaPerUnit),
			OtherRatios: map[string]float64{
				"seconds": 15,
			},
		},
	}

	require.False(t, shouldApplyTaskOtherRatios(info, "sora-2"))
}

func TestShouldApplyTaskOtherRatiosKeepsProviderDurationPricing(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeAli,
		},
		PriceData: types.PriceData{
			UsePrice: false,
			Quota:    int(common.QuotaPerUnit),
			OtherRatios: map[string]float64{
				"seconds": 15,
			},
		},
	}

	require.True(t, shouldApplyTaskOtherRatios(info, "wanx2.1-t2v-turbo"))
}

func TestShouldApplyTaskOtherRatiosKeepsDoubaoVideoSeedancePricing(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeDoubaoVideo,
		},
		PriceData: types.PriceData{
			UsePrice: false,
			Quota:    int(common.QuotaPerUnit),
			OtherRatios: map[string]float64{
				"seconds": 4,
			},
		},
	}

	require.True(t, shouldApplyTaskOtherRatios(info, "doubao-seedance-2-0-fast-260128"))
}

func setupVideoResolutionPricingRules(t *testing.T) {
	t.Helper()
	setupVideoResolutionPricingRulesWithUnit(t, per_request_pricing.UnitSecond)
}

func setupVideoResolutionPricingRulesWithUnit(t *testing.T, unit string) {
	t.Helper()

	original := per_request_pricing.RulesToJSONString()
	t.Cleanup(func() {
		require.NoError(t, per_request_pricing.UpdateRulesByJSONString(original))
	})

	rules := map[string]per_request_pricing.PerRequestPriceRule{
		"video-test-model": {
			MediaType:         per_request_pricing.MediaTypeVideo,
			Unit:              unit,
			Prices:            map[string]float64{"480": 0.04, "1K": 0.08, "2K": 0.12, "4K": 0.24},
			DefaultResolution: "1K",
			FallbackEnabled:   false,
		},
	}
	rulesJSON, err := common.Marshal(rules)
	require.NoError(t, err)
	require.NoError(t, per_request_pricing.UpdateRulesByJSONString(string(rulesJSON)))
}
