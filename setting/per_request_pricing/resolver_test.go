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

func TestResolvePricingOnlyMatchesConfiguredResolution(t *testing.T) {
	rule := PerRequestPriceRule{
		MediaType:         MediaTypeVideo,
		Unit:              UnitSecond,
		Prices:            map[string]float64{"1K": 0.12},
		DefaultResolution: "1K",
		FallbackEnabled:   false,
	}
	_, err := ResolveVideoPricing("test-model", rule, VideoPricingInput{
		Size:         "1080p",
		Seconds:      "5",
		GroupRatio:   1,
		QuotaPerUnit: 500000,
	})
	if err == nil {
		t.Fatal("expected unconfigured resolution to be rejected")
	}
	if !strings.Contains(err.Error(), `unknown video resolution "1080p"`) {
		t.Fatalf("unexpected error: %v", err)
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
