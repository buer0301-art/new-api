package per_request_pricing

import (
	"fmt"
	"math"
	"strings"

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

func init() {
	config.GlobalConfig.Register("per_request_pricing", &currentSetting)
}

func GetSetting() *Setting {
	return &currentSetting
}

func GetRule(model string) (PerRequestPriceRule, bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return PerRequestPriceRule{}, false
	}
	rule, ok := currentSetting.Rules[model]
	return rule, ok
}

func GetRulesCopy() map[string]PerRequestPriceRule {
	if len(currentSetting.Rules) == 0 {
		return map[string]PerRequestPriceRule{}
	}
	copied := make(map[string]PerRequestPriceRule, len(currentSetting.Rules))
	for model, rule := range currentSetting.Rules {
		copied[model] = copyRule(rule)
	}
	return copied
}

func RulesToJSONString() string {
	jsonBytes, err := common.Marshal(currentSetting.Rules)
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
	currentSetting.Rules = rules
	return nil
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
		if _, ok := rule.Prices[defaultResolution]; !ok {
			return fmt.Errorf("model %s: default resolution %q must exist in prices", model, defaultResolution)
		}
		for resolution, price := range rule.Prices {
			if strings.TrimSpace(resolution) == "" {
				return fmt.Errorf("model %s: resolution cannot be empty", model)
			}
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
