package per_request_pricing

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const OptionKeyRules = "per_request_pricing.rules"

const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	UnitImage      = "image"
	UnitSecond     = "second"
)

type PerRequestPriceRule struct {
	MediaType         string             `json:"media_type"`
	Unit              string             `json:"unit"`
	Prices            map[string]float64 `json:"prices"`
	DefaultResolution string             `json:"default_resolution"`
	FallbackEnabled   bool               `json:"fallback_enabled"`
}

type Setting struct {
	Rules map[string]PerRequestPriceRule `json:"rules"`
}

var currentSetting = Setting{
	Rules: map[string]PerRequestPriceRule{},
}

var currentSettingMutex sync.RWMutex

func init() {
	config.GlobalConfig.Register("per_request_pricing", &currentSetting)
}

func GetRule(model string) (PerRequestPriceRule, bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return PerRequestPriceRule{}, false
	}
	currentSettingMutex.RLock()
	defer currentSettingMutex.RUnlock()

	rule, ok := currentSetting.Rules[model]
	if !ok {
		return PerRequestPriceRule{}, false
	}
	return copyRule(rule), true
}

func GetRulesCopy() map[string]PerRequestPriceRule {
	currentSettingMutex.RLock()
	defer currentSettingMutex.RUnlock()

	return copyRules(currentSetting.Rules)
}

func RulesToJSONString() string {
	currentSettingMutex.RLock()
	rules := copyRules(currentSetting.Rules)
	currentSettingMutex.RUnlock()

	jsonBytes, err := common.Marshal(rules)
	if err != nil {
		common.SysError("error marshalling per-request pricing rules: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func UpdateRulesByJSONString(jsonStr string) error {
	var rules map[string]PerRequestPriceRule
	if err := common.UnmarshalJsonStr(jsonStr, &rules); err != nil {
		return err
	}
	if err := ValidateRules(rules); err != nil {
		return err
	}
	rules = copyRules(rules)

	currentSettingMutex.Lock()
	defer currentSettingMutex.Unlock()
	currentSetting.Rules = rules
	return nil
}

func copyRules(rules map[string]PerRequestPriceRule) map[string]PerRequestPriceRule {
	if len(rules) == 0 {
		return map[string]PerRequestPriceRule{}
	}
	copied := make(map[string]PerRequestPriceRule, len(rules))
	for model, rule := range rules {
		copied[model] = copyRule(rule)
	}
	return copied
}

func ValidateRules(rules map[string]PerRequestPriceRule) error {
	for model, rule := range rules {
		model = strings.TrimSpace(model)
		if model == "" {
			return fmt.Errorf("model name cannot be empty")
		}
		switch rule.MediaType {
		case MediaTypeImage:
			if rule.Unit != UnitImage {
				return fmt.Errorf("model %s: unit must be %s for media type %s", model, UnitImage, MediaTypeImage)
			}
		case MediaTypeVideo:
			if rule.Unit != UnitSecond {
				return fmt.Errorf("model %s: unit must be %s for media type %s", model, UnitSecond, MediaTypeVideo)
			}
		default:
			return fmt.Errorf("model %s: invalid media type %q", model, rule.MediaType)
		}
		if len(rule.Prices) == 0 {
			return fmt.Errorf("model %s: prices cannot be empty", model)
		}
		defaultResolution := strings.TrimSpace(rule.DefaultResolution)
		if defaultResolution == "" {
			return fmt.Errorf("model %s: default resolution cannot be empty", model)
		}
		if _, ok := matchNormalizedResolution(defaultResolution, rule.Prices); !ok {
			return fmt.Errorf("model %s: default resolution %q must exist in prices", model, defaultResolution)
		}
		seenResolutions := map[string]string{}
		for resolution, price := range rule.Prices {
			if strings.TrimSpace(resolution) == "" {
				return fmt.Errorf("model %s: resolution cannot be empty", model)
			}
			resolutionKey := normalizeResolutionKey(resolution)
			if existing, ok := seenResolutions[resolutionKey]; ok {
				return fmt.Errorf("model %s: duplicate resolution %q conflicts with %q after normalization", model, resolution, existing)
			}
			seenResolutions[resolutionKey] = resolution
			if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
				return fmt.Errorf("model %s: invalid price for resolution %q", model, resolution)
			}
		}
	}
	return nil
}

func copyRule(rule PerRequestPriceRule) PerRequestPriceRule {
	copied := rule
	if len(rule.Prices) > 0 {
		copied.Prices = make(map[string]float64, len(rule.Prices))
		for resolution, price := range rule.Prices {
			copied.Prices[resolution] = price
		}
	}
	return copied
}
