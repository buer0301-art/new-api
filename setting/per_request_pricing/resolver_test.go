package per_request_pricing

import "testing"

func TestNormalizeResolutionImageAliases(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"1k", "1K", true},
		{"1024x1024", "1K", true},
		{"1024x1536", "1K", true},
		{"1536x1024", "1K", true},
		{"2048x2048", "2K", true},
		{"4096x4096", "4K", true},
		{"3840x2160", "4K", true},
		{"2160x3840", "4K", true},
	}
	for _, tt := range tests {
		got, ok := NormalizeResolution(MediaTypeImage, tt.raw)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("NormalizeResolution(image, %q) = %q, %v; want %q, %v", tt.raw, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNormalizeResolutionVideoAliases(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"480", "480", true},
		{"480p", "480", true},
		{"contains-480-here", "480", true},
		{"980", "980", true},
		{"980p", "980", true},
		{"1080", "1K", true},
		{"1080p", "1K", true},
		{"contains-1080-here", "1K", true},
		{"1440", "2K", true},
		{"2048", "2K", true},
		{"2160", "4K", true},
		{"2160p", "4K", true},
		{"3840", "4K", true},
		{"4096", "4K", true},
	}
	for _, tt := range tests {
		got, ok := NormalizeResolution(MediaTypeVideo, tt.raw)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("NormalizeResolution(video, %q) = %q, %v; want %q, %v", tt.raw, got, ok, tt.want, tt.ok)
		}
	}
}

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
		Size:         "2048x2048",
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

func TestResolveVideoPricingSeconds(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"4K": 0.24},
		DefaultResolution: "4K",
		FallbackEnabled:   false,
	}
	resolved, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "4k",
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
