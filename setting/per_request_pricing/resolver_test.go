package per_request_pricing

import (
	"strings"
	"sync"
	"testing"
)

func TestResolveImagePricingCountOnce(t *testing.T) {
	n := uint(3)
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeImage,
		Unit:              UnitImage,
		Prices:            map[string]float64{"2K": 0.02},
		DefaultResolution: "2K",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveImagePricing("test-model", rule, ImagePricingInput{
		Size:         "2K",
		N:            &n,
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err != nil {
		t.Fatalf("ResolveImagePricing error: %v", err)
	}
	if resolved.Resolution != "2K" || resolved.Quantity != 3 || resolved.PriceUSD != 0.06 || resolved.Quota != 30000 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveVideoPricingMapsCommonResolutionAliases(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"480": 0.04, "980": 0.06, "1K": 0.08, "2K": 0.12, "4K": 0.24},
		DefaultResolution: "1K",
		FallbackEnabled:   false,
	}

	tests := []struct {
		name       string
		size       string
		metadata   string
		resolution string
		unitPrice  float64
	}{
		{name: "480p label", size: "480p", resolution: "480", unitPrice: 0.04},
		{name: "tiny pixel size", size: "210*120", resolution: "480", unitPrice: 0.04},
		{name: "720p label", size: "720p", resolution: "980", unitPrice: 0.06},
		{name: "980p label", size: "980p", resolution: "980", unitPrice: 0.06},
		{name: "1080p label", size: "1080p", resolution: "1K", unitPrice: 0.08},
		{name: "1440p label", size: "1440p", resolution: "2K", unitPrice: 0.12},
		{name: "2k pixel size", size: "2560x1440", resolution: "2K", unitPrice: 0.12},
		{name: "2160p label", size: "2160p", resolution: "4K", unitPrice: 0.24},
		{name: "4k pixel size", size: "3840x2160", resolution: "4K", unitPrice: 0.24},
		{name: "large 4k pixel size", size: "4800*2400", resolution: "4K", unitPrice: 0.24},
		{name: "ultrawide 4k by long side", size: "9600x480", resolution: "4K", unitPrice: 0.24},
		{name: "huge 4k pixel size", size: "9600*4800", resolution: "4K", unitPrice: 0.24},
		{name: "metadata takes priority", size: "1080p", metadata: "2160p", resolution: "4K", unitPrice: 0.24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
				Size:               tt.size,
				MetadataResolution: tt.metadata,
				Seconds:            "5",
				GroupRatio:         1,
				QuotaPerUnit:       500000,
			})
			if err != nil {
				t.Fatalf("ResolveVideoPricing error: %v", err)
			}
			if resolved.Resolution != tt.resolution || resolved.UnitPrice != tt.unitPrice {
				t.Fatalf("unexpected resolved pricing: %+v", resolved)
			}
		})
	}
}

func TestResolveVideoPricingMaps980KAs980TierAlias(t *testing.T) {
	tests := []struct {
		name       string
		prices     map[string]float64
		requests   []string
		resolution string
	}{
		{
			name:       "configured as 980",
			prices:     map[string]float64{"980": 0.06, "1K": 0.08},
			requests:   []string{"980p", "980", "980k"},
			resolution: "980",
		},
		{
			name:       "configured as 980p",
			prices:     map[string]float64{"980p": 0.06, "1K": 0.08},
			requests:   []string{"980p", "980", "980k"},
			resolution: "980p",
		},
		{
			name:       "configured as 980k",
			prices:     map[string]float64{"980K": 0.06, "1K": 0.08},
			requests:   []string{"980p", "980", "980k"},
			resolution: "980K",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := PerRequestPriceRule{
				MediaType:         MediaTypeVideo,
				Unit:              UnitSecond,
				Prices:            tt.prices,
				DefaultResolution: "1K",
				FallbackEnabled:   false,
			}
			for _, size := range tt.requests {
				t.Run(size, func(t *testing.T) {
					resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
						Size:         size,
						Seconds:      "5",
						GroupRatio:   1,
						QuotaPerUnit: 500000,
					})
					if err != nil {
						t.Fatalf("ResolveVideoPricing error: %v", err)
					}
					if resolved.Resolution != tt.resolution || resolved.UnitPrice != 0.06 {
						t.Fatalf("unexpected resolved pricing: %+v", resolved)
					}
				})
			}
		})
	}
}

func TestResolveVideoPricingMapsTierLabelsBidirectionally(t *testing.T) {
	tests := []struct {
		name       string
		prices     map[string]float64
		size       string
		resolution string
	}{
		{name: "1k request hits 1080p price", prices: map[string]float64{"1080p": 0.08, "4K": 0.24}, size: "1K", resolution: "1080p"},
		{name: "1080p request hits 1k price", prices: map[string]float64{"1K": 0.08, "4K": 0.24}, size: "1080p", resolution: "1K"},
		{name: "2k request hits 1440p price", prices: map[string]float64{"1440p": 0.12, "4K": 0.24}, size: "2K", resolution: "1440p"},
		{name: "1440p request hits 2k price", prices: map[string]float64{"2K": 0.12, "4K": 0.24}, size: "1440p", resolution: "2K"},
		{name: "4k request hits 2160p price", prices: map[string]float64{"2160p": 0.24}, size: "4K", resolution: "2160p"},
		{name: "2160p request hits 4k price", prices: map[string]float64{"4K": 0.24}, size: "2160p", resolution: "4K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := PerRequestPriceRule{
				MediaType:         MediaTypeVideo,
				Unit:              UnitSecond,
				Prices:            tt.prices,
				DefaultResolution: tt.resolution,
				FallbackEnabled:   false,
			}
			resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
				Size:         tt.size,
				Seconds:      "5",
				GroupRatio:   1,
				QuotaPerUnit: 500000,
			})
			if err != nil {
				t.Fatalf("ResolveVideoPricing error: %v", err)
			}
			if resolved.Resolution != tt.resolution {
				t.Fatalf("unexpected resolved pricing: %+v", resolved)
			}
		})
	}
}

func TestResolveVideoPricingUsesNextConfiguredTierForLowResolution(t *testing.T) {
	tests := []struct {
		name       string
		prices     map[string]float64
		size       string
		resolution string
		unitPrice  float64
	}{
		{
			name:       "low p label falls forward to one k",
			prices:     map[string]float64{"1K": 0.08, "2K": 0.12, "4K": 0.24},
			size:       "360p",
			resolution: "1K",
			unitPrice:  0.08,
		},
		{
			name:       "480p falls forward to one k when 480 is not configured",
			prices:     map[string]float64{"1K": 0.08, "2K": 0.12, "4K": 0.24},
			size:       "480p",
			resolution: "1K",
			unitPrice:  0.08,
		},
		{
			name:       "980p falls forward to one k when 980 is not configured",
			prices:     map[string]float64{"1K": 0.08, "2K": 0.12, "4K": 0.24},
			size:       "980p",
			resolution: "1K",
			unitPrice:  0.08,
		},
		{
			name:       "1080p falls forward to two k when one k is not configured",
			prices:     map[string]float64{"2K": 0.12, "4K": 0.24},
			size:       "1080p",
			resolution: "2K",
			unitPrice:  0.12,
		},
		{
			name:       "1440p falls forward to four k when two k is not configured",
			prices:     map[string]float64{"4K": 0.24},
			size:       "1440p",
			resolution: "4K",
			unitPrice:  0.24,
		},
		{
			name:       "portrait size falls forward to one k",
			prices:     map[string]float64{"1K": 0.08, "2K": 0.12, "4K": 0.24},
			size:       "720x1280",
			resolution: "1K",
			unitPrice:  0.08,
		},
		{
			name:       "portrait star separated size falls forward to one k",
			prices:     map[string]float64{"1K": 0.08, "2K": 0.12, "4K": 0.24},
			size:       "720*1280",
			resolution: "1K",
			unitPrice:  0.08,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := PerRequestPriceRule{
				MediaType:         MediaTypeVideo,
				Unit:              UnitSecond,
				Prices:            tt.prices,
				DefaultResolution: tt.resolution,
				FallbackEnabled:   false,
			}
			resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
				Size:         tt.size,
				Seconds:      "5",
				GroupRatio:   1,
				QuotaPerUnit: 500000,
			})
			if err != nil {
				t.Fatalf("ResolveVideoPricing error: %v", err)
			}
			if resolved.Resolution != tt.resolution || resolved.UnitPrice != tt.unitPrice {
				t.Fatalf("unexpected resolved pricing: %+v", resolved)
			}
		})
	}
}

func TestResolveVideoPricingDocumentsLongSideTierPromotion(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"480": 0.04, "980": 0.06, "1K": 0.08, "2K": 0.12, "4K": 0.24},
		DefaultResolution: "1K",
		FallbackEnabled:   false,
	}

	tests := []struct {
		name       string
		size       string
		resolution string
	}{
		{name: "long side above 1920 promotes to 2k", size: "2048x360", resolution: "2K"},
		{name: "long side above 2560 promotes to 4k", size: "2561x360", resolution: "4K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
				Size:         tt.size,
				Seconds:      "5",
				GroupRatio:   1,
				QuotaPerUnit: 500000,
			})
			if err != nil {
				t.Fatalf("ResolveVideoPricing error: %v", err)
			}
			if resolved.Resolution != tt.resolution {
				t.Fatalf("unexpected resolved pricing: %+v", resolved)
			}
		})
	}
}

func TestResolveVideoPricingNormalizesPixelSeparatorsForCustomResolution(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"180*640": 0.03, "1K": 0.08},
		DefaultResolution: "1K",
		FallbackEnabled:   false,
	}

	for _, size := range []string{"180*640", "180x640", "180X640"} {
		t.Run(size, func(t *testing.T) {
			resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
				Size:         size,
				Seconds:      "5",
				GroupRatio:   1,
				QuotaPerUnit: 500000,
			})
			if err != nil {
				t.Fatalf("ResolveVideoPricing error: %v", err)
			}
			if resolved.Resolution != "180*640" || resolved.UnitPrice != 0.03 {
				t.Fatalf("unexpected resolved pricing: %+v", resolved)
			}
		})
	}
}

func TestResolveImagePricingEmptySizeUsesDefault(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeImage,
		Unit:              UnitImage,
		Prices:            map[string]float64{"2K": 0.02},
		DefaultResolution: "2K",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveImagePricing("test-model", rule, ImagePricingInput{
		Size:         "",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err != nil {
		t.Fatalf("ResolveImagePricing error: %v", err)
	}
	if resolved.Resolution != "2K" || resolved.Quantity != 1 || resolved.PriceUSD != 0.02 || resolved.Quota != 10000 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveImagePricingClassifiesPixelSizeByArea(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeImage,
		Unit:              UnitImage,
		Prices:            map[string]float64{"1K": 0.01, "2K": 0.02, "4k": 0.04},
		DefaultResolution: "1K",
		FallbackEnabled:   false,
	}

	tests := []struct {
		name       string
		size       string
		resolution string
		unitPrice  float64
	}{
		{name: "small star separated one k", size: "640*760", resolution: "1K", unitPrice: 0.01},
		{name: "tiny star separated one k", size: "210*120", resolution: "1K", unitPrice: 0.01},
		{name: "gpt image portrait one k", size: "1024x1536", resolution: "1K", unitPrice: 0.01},
		{name: "wide under two k max area", size: "2560x1400", resolution: "2K", unitPrice: 0.02},
		{name: "wide two k boundary", size: "2560x1440", resolution: "2K", unitPrice: 0.02},
		{name: "ultrawide four k by long side", size: "9600x120", resolution: "4k", unitPrice: 0.04},
		{name: "uhd four k", size: "3840x2160", resolution: "4k", unitPrice: 0.04},
		{name: "large star separated four k", size: "4800*2400", resolution: "4k", unitPrice: 0.04},
		{name: "huge star separated four k", size: "9600*4800", resolution: "4k", unitPrice: 0.04},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveImagePricing("test-model", rule, ImagePricingInput{
				Size:         tt.size,
				GroupRatio:   1,
				QuotaPerUnit: 500000,
			})
			if err != nil {
				t.Fatalf("ResolveImagePricing error: %v", err)
			}
			if resolved.Resolution != tt.resolution || resolved.UnitPrice != tt.unitPrice {
				t.Fatalf("unexpected resolved pricing: %+v", resolved)
			}
		})
	}
}

func TestResolveVideoPricingSeconds(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"4K": 0.24},
		DefaultResolution: "4K",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "4K",
		Seconds:      "10",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err != nil {
		t.Fatalf("ResolveVideoPricing error: %v", err)
	}
	if resolved.Quota != 1200000 {
		t.Fatalf("unexpected quota: %+v", resolved)
	}
}

func TestResolveVideoPricingPerRequestUnitCountsOnce(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitRequest,
		Prices:            map[string]float64{"1080": 1},
		DefaultResolution: "1080",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "1080p",
		Seconds:      "15",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err != nil {
		t.Fatalf("ResolveVideoPricing error: %v", err)
	}
	if resolved.Unit != UnitRequest || resolved.Quantity != 1 || resolved.PriceUSD != 1 || resolved.Quota != 500000 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveVideoPricingPerRequestUnitDoesNotRequireDuration(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitRequest,
		Prices:            map[string]float64{"720": 0.5},
		DefaultResolution: "720",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "720p",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err != nil {
		t.Fatalf("ResolveVideoPricing error: %v", err)
	}
	if resolved.Quantity != 1 || resolved.PriceUSD != 0.5 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveVideoPricingEmptyResolutionUsesDefault(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"4K": 0.24},
		DefaultResolution: "4K",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:               "",
		MetadataResolution: "",
		Seconds:            "10",
		GroupRatio:         1,
		QuotaPerUnit:       500000,
	})
	if err != nil {
		t.Fatalf("ResolveVideoPricing error: %v", err)
	}
	if resolved.Resolution != "4K" || resolved.Quantity != 10 || resolved.PriceUSD != 2.4 || resolved.Quota != 1200000 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveVideoPricingMetadataResolutionTakesPriority(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"1K": 0.12, "4K": 0.24},
		DefaultResolution: "1K",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:               "1K",
		MetadataResolution: "4K",
		Seconds:            "10",
		GroupRatio:         1,
		QuotaPerUnit:       500000,
	})
	if err != nil {
		t.Fatalf("ResolveVideoPricing error: %v", err)
	}
	if resolved.Resolution != "4K" || resolved.UnitPrice != 0.24 || resolved.PriceUSD != 2.4 || resolved.Quota != 1200000 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveVideoPricingUsesConfiguredCustomResolution(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"720p": 0.08, "1080p": 0.12},
		DefaultResolution: "720p",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "1080P",
		Seconds:      "5",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err != nil {
		t.Fatalf("ResolveVideoPricing error: %v", err)
	}
	if resolved.Resolution != "1080p" || resolved.UnitPrice != 0.12 || resolved.PriceUSD != 0.6 || resolved.Quota != 300000 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveVideoPricingMissingDurationRejected(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"4K": 0.24},
		DefaultResolution: "4K",
		FallbackEnabled:   false,
	}
	_, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "4K",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err == nil {
		t.Fatal("expected missing duration error")
	}
	if !strings.Contains(err.Error(), `invalid video seconds ""/duration 0`) {
		t.Fatalf("error missing duration context: %v", err)
	}
}

func TestResolveVideoPricingInvalidSecondsIncludesContext(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"4K": 0.24},
		DefaultResolution: "4K",
		FallbackEnabled:   false,
	}
	_, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "4K",
		Seconds:      "0",
		Duration:     5,
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err == nil {
		t.Fatal("expected invalid seconds error")
	}
	message := err.Error()
	if !strings.Contains(message, "model test-model") || !strings.Contains(message, `invalid video seconds "0"/duration 5`) {
		t.Fatalf("error missing context: %v", err)
	}
}

func TestResolveUnknownResolutionRejected(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeImage,
		Unit:              UnitImage,
		Prices:            map[string]float64{"2K": 0.02},
		DefaultResolution: "2K",
		FallbackEnabled:   false,
	}
	_, err := ResolveImagePricing("test-model", rule, ImagePricingInput{
		Size:         "unknown",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err == nil {
		t.Fatal("expected error for unknown resolution")
	}
}

func TestValidateRulesRejectsDefaultMissing(t *testing.T) {
	err := ValidateRules(map[string]PerRequestPriceRule{
		"test-model": {
			MediaType:         MediaTypeImage,
			Unit:              UnitImage,
			Prices:            map[string]float64{"2K": 0.02},
			DefaultResolution: "1K",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRulesAcceptsNormalizedDefaultResolution(t *testing.T) {
	err := ValidateRules(map[string]PerRequestPriceRule{
		"test-model": {
			MediaType:         MediaTypeVideo,
			Unit:              UnitSecond,
			Prices:            map[string]float64{"180*640": 0.03},
			DefaultResolution: "180x640",
		},
	})
	if err != nil {
		t.Fatalf("expected normalized default resolution to be accepted: %v", err)
	}
}

func TestValidateRulesRejectsDefaultResolutionMatchedOnlyByTierAlias(t *testing.T) {
	err := ValidateRules(map[string]PerRequestPriceRule{
		"test-model": {
			MediaType:         MediaTypeVideo,
			Unit:              UnitSecond,
			Prices:            map[string]float64{"4K": 0.24},
			DefaultResolution: "480",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), `default resolution "480" must exist in prices`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveVideoPricingUsesNormalizedDefaultResolution(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"180*640": 0.03},
		DefaultResolution: "180x640",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Seconds:      "5",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err != nil {
		t.Fatalf("ResolveVideoPricing error: %v", err)
	}
	if resolved.Resolution != "180*640" || resolved.UnitPrice != 0.03 {
		t.Fatalf("unexpected resolved pricing: %+v", resolved)
	}
}

func TestResolveVideoPricingRejectsDefaultResolutionMatchedOnlyByTierAlias(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"4K": 0.24},
		DefaultResolution: "480",
		FallbackEnabled:   false,
	}
	_, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Seconds:      "5",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err == nil {
		t.Fatal("expected default resolution error")
	}
	if !strings.Contains(err.Error(), `resolution "480" not configured`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRulesRejectsDuplicateNormalizedResolution(t *testing.T) {
	err := ValidateRules(map[string]PerRequestPriceRule{
		"test-model": {
			MediaType:         MediaTypeVideo,
			Unit:              UnitSecond,
			Prices:            map[string]float64{"180*640": 0.03, "180x640": 0.08},
			DefaultResolution: "180*640",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), `duplicate resolution`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRulesAccessorsReturnDeepCopies(t *testing.T) {
	original := RulesToJSONString()
	defer func() {
		if err := UpdateRulesByJSONString(original); err != nil {
			t.Fatalf("restore rules: %v", err)
		}
	}()

	if err := UpdateRulesByJSONString(`{"test-model":{"media_type":"image","unit":"image","prices":{"2K":0.02},"default_resolution":"2K","fallback_enabled":false}}`); err != nil {
		t.Fatalf("UpdateRulesByJSONString error: %v", err)
	}

	rule, ok := GetRule("test-model")
	if !ok {
		t.Fatal("expected test-model rule")
	}
	rule.Prices["2K"] = 99
	ruleAgain, _ := GetRule("test-model")
	if ruleAgain.Prices["2K"] != 0.02 {
		t.Fatalf("GetRule exposed mutable prices map: %+v", ruleAgain)
	}

	rules := GetRulesCopy()
	rules["test-model"].Prices["2K"] = 88
	ruleAgain, _ = GetRule("test-model")
	if ruleAgain.Prices["2K"] != 0.02 {
		t.Fatalf("GetRulesCopy exposed mutable prices map: %+v", ruleAgain)
	}

	rules["new-model"] = PerRequestPriceRule{}
	if _, ok := GetRule("new-model"); ok {
		t.Fatal("GetRulesCopy exposed mutable rules map")
	}
}

func TestRulesConcurrentReadWrite(t *testing.T) {
	original := RulesToJSONString()
	defer func() {
		if err := UpdateRulesByJSONString(original); err != nil {
			t.Fatalf("restore rules: %v", err)
		}
	}()

	const iterations = 100
	var wg sync.WaitGroup

	for writer := 0; writer < 4; writer++ {
		writer := writer
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				jsonStr := `{"concurrent-model":{"media_type":"image","unit":"image","prices":{"2K":0.02},"default_resolution":"2K","fallback_enabled":false},"writer-model":{"media_type":"video","unit":"second","prices":{"4K":0.24},"default_resolution":"4K","fallback_enabled":true}}`
				if writer%2 == 1 {
					jsonStr = `{"concurrent-model":{"media_type":"image","unit":"image","prices":{"1K":0.01,"2K":0.02},"default_resolution":"1K","fallback_enabled":true}}`
				}
				if err := UpdateRulesByJSONString(jsonStr); err != nil {
					t.Errorf("UpdateRulesByJSONString error: %v", err)
					return
				}
			}
		}()
	}

	for reader := 0; reader < 8; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_, _ = GetRule("concurrent-model")
				_ = GetRulesCopy()
				_ = RulesToJSONString()
			}
		}()
	}

	wg.Wait()

	rule, ok := GetRule("concurrent-model")
	if !ok {
		t.Fatal("expected concurrent-model rule after concurrent access")
	}
	if rule.DefaultResolution == "" || len(rule.Prices) == 0 {
		t.Fatalf("final rule is not readable: %+v", rule)
	}
}
