package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/per_request_pricing"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperTieredUsesPreloadedRequestInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"tiered-test-model":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"tiered-test-model":"param(\"stream\") == true ? tier(\"stream\", p * 3) : tier(\"base\", p * 2)"}`,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/channel/test/1", nil)
	req.Body = nil
	req.ContentLength = 0
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "tiered-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"stream":true}`),
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.Equal(t, 1500, priceData.QuotaToPreConsume)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, "stream", info.TieredBillingSnapshot.EstimatedTier)
	require.Equal(t, billing_setting.BillingModeTieredExpr, info.TieredBillingSnapshot.BillingMode)
	require.Equal(t, common.QuotaPerUnit, info.TieredBillingSnapshot.QuotaPerUnit)
}

func TestModelPriceHelperImageResolutionPricing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupImageResolutionPricingRules(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	n := uint(3)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		UserGroup:       "default",
		UsingGroup:      "default",
		Request: &dto.ImageRequest{
			Model: "gpt-image-2",
			Size:  "2K",
			N:     &n,
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 0, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.06, priceData.ModelPrice)
	require.Equal(t, 30000, priceData.QuotaToPreConsume)
	require.NotNil(t, priceData.ResolvedPerRequestPricing)
	require.Equal(t, float64(3), priceData.ResolvedPerRequestPricing.Quantity)
	require.Equal(t, priceData.ResolvedPerRequestPricing, info.PriceData.ResolvedPerRequestPricing)
}

func TestModelPriceHelperImageResolutionPricingMapsPixelSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupImageResolutionPricingRules(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	n := uint(1)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		UserGroup:       "default",
		UsingGroup:      "default",
		Request: &dto.ImageRequest{
			Model: "gpt-image-2",
			Size:  "2560x1440",
			N:     &n,
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 0, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.02, priceData.ModelPrice)
	require.Equal(t, 10000, priceData.QuotaToPreConsume)
	require.NotNil(t, priceData.ResolvedPerRequestPricing)
	require.Equal(t, "2K", priceData.ResolvedPerRequestPricing.Resolution)
}

func TestModelPriceHelperImageResolutionPricingUnknownSizeWithoutFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupImageResolutionPricingRules(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	n := uint(1)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		UserGroup:       "default",
		UsingGroup:      "default",
		Request: &dto.ImageRequest{
			Model: "gpt-image-2",
			Size:  "8K",
			N:     &n,
		},
	}

	_, err := ModelPriceHelper(ctx, info, 0, &types.TokenCountMeta{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown image resolution")
}

func TestModelPriceHelperImageResolutionPricingSkipsNonImageRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupImageResolutionPricingRules(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		UserGroup:       "default",
		UsingGroup:      "default",
		Request:         nil,
	}

	_, err := ModelPriceHelper(ctx, info, 0, &types.TokenCountMeta{})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "expected *dto.ImageRequest")
	require.NotContains(t, err.Error(), "per-request pricing")
}

func TestHasModelBillingConfigAcceptsResolutionPricingRule(t *testing.T) {
	setupImageResolutionPricingRules(t)

	require.True(t, HasModelBillingConfig("gpt-image-2"))
}

func setupImageResolutionPricingRules(t *testing.T) {
	t.Helper()

	original := per_request_pricing.RulesToJSONString()
	t.Cleanup(func() {
		require.NoError(t, per_request_pricing.UpdateRulesByJSONString(original))
	})

	rules := map[string]per_request_pricing.PerRequestPriceRule{
		"gpt-image-2": {
			MediaType:         per_request_pricing.MediaTypeImage,
			Unit:              per_request_pricing.UnitImage,
			Prices:            map[string]float64{"1K": 0.01, "2K": 0.02, "4K": 0.04},
			DefaultResolution: "1K",
			FallbackEnabled:   false,
		},
	}
	rulesJSON, err := common.Marshal(rules)
	require.NoError(t, err)
	require.NoError(t, per_request_pricing.UpdateRulesByJSONString(string(rulesJSON)))
}

func TestModelPriceHelperTieredPreConsumeMaxTokensFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":    `{"tiered-fallback-model":"tiered_expr"}`,
		"billing_setting.billing_expr":    `{"tiered-fallback-model":"tier(\"base\", p * 3 + c * 15)"}`,
		"group_ratio_setting.group_ratio": `{"default":1,"free":0}`,
	}))

	const promptTokens = 1000

	cases := []struct {
		name      string
		group     string
		maxTokens int
		expected  int
	}{
		{
			// max_tokens omitted in a paid group -> fall back to 8192 completion tokens.
			// p*3 + c*15 = 1000*3 + 8192*15 = 125880 -> /1e6 * 500000 = 62940
			name:      "non-free group falls back to 8192 completion tokens",
			group:     "default",
			maxTokens: 0,
			expected:  62940,
		},
		{
			// explicit max_tokens is used verbatim, no fallback.
			// 1000*3 + 100*15 = 4500 -> /1e6 * 500000 = 2250
			name:      "explicit max_tokens is used verbatim",
			group:     "default",
			maxTokens: 100,
			expected:  2250,
		},
		{
			// free group (ratio 0) stays zero; fallback is gated on non-zero group ratio.
			name:      "free group stays zero without fallback",
			group:     "free",
			maxTokens: 0,
			expected:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req
			ctx.Set("group", tc.group)

			info := &relaycommon.RelayInfo{
				OriginModelName: "tiered-fallback-model",
				UserGroup:       tc.group,
				UsingGroup:      tc.group,
				RequestHeaders:  map[string]string{"Content-Type": "application/json"},
				BillingRequestInput: &billingexpr.RequestInput{
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    []byte(`{}`),
				},
			}

			priceData, err := ModelPriceHelper(ctx, info, promptTokens, &types.TokenCountMeta{MaxTokens: tc.maxTokens})
			require.NoError(t, err)
			require.Equal(t, tc.expected, priceData.QuotaToPreConsume)
		})
	}
}
