package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
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
