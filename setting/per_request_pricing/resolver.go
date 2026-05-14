package per_request_pricing

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/types"
)

type ImagePricingInput struct {
	Size         string
	N            *uint
	GroupRatio   float64
	QuotaPerUnit float64
}

type VideoPricingInput struct {
	Size               string
	MetadataResolution string
	Seconds            string
	Duration           int
	GroupRatio         float64
	QuotaPerUnit       float64
}

func ResolveImagePricing(model string, rule PerRequestPriceRule, input ImagePricingInput) (*types.ResolvedPerRequestPricing, error) {
	if rule.MediaType != MediaTypeImage {
		return nil, fmt.Errorf("model %s: media type mismatch, expected %s", model, MediaTypeImage)
	}
	resolution, unitPrice, err := resolveRulePrice(model, rule, MediaTypeImage, input.Size)
	if err != nil {
		return nil, err
	}
	quantity := float64(1)
	if input.N != nil && *input.N > 0 {
		quantity = float64(*input.N)
	}
	priceUSD := unitPrice * quantity
	quota := billingexpr.QuotaRound(priceUSD * input.GroupRatio * input.QuotaPerUnit)
	return &types.ResolvedPerRequestPricing{
		Mode:       "resolution",
		MediaType:  MediaTypeImage,
		Unit:       UnitImage,
		Resolution: resolution,
		UnitPrice:  unitPrice,
		Quantity:   quantity,
		PriceUSD:   priceUSD,
		Quota:      quota,
	}, nil
}

func ResolveVideoPricing(model string, rule PerRequestPriceRule, input VideoPricingInput) (*types.ResolvedPerRequestPricing, error) {
	if rule.MediaType != MediaTypeVideo {
		return nil, fmt.Errorf("model %s: media type mismatch, expected %s", model, MediaTypeVideo)
	}
	rawResolution := strings.TrimSpace(input.MetadataResolution)
	if rawResolution == "" {
		rawResolution = strings.TrimSpace(input.Size)
	}
	resolution, unitPrice, err := resolveRulePrice(model, rule, MediaTypeVideo, rawResolution)
	if err != nil {
		return nil, err
	}
	seconds, err := parseSeconds(input.Seconds, input.Duration)
	if err != nil {
		return nil, fmt.Errorf("model %s: invalid video seconds %q/duration %d: %w", model, input.Seconds, input.Duration, err)
	}
	quantity := float64(seconds)
	priceUSD := unitPrice * quantity
	quota := billingexpr.QuotaRound(priceUSD * input.GroupRatio * input.QuotaPerUnit)
	return &types.ResolvedPerRequestPricing{
		Mode:       "resolution",
		MediaType:  MediaTypeVideo,
		Unit:       UnitSecond,
		Resolution: resolution,
		UnitPrice:  unitPrice,
		Quantity:   quantity,
		PriceUSD:   priceUSD,
		Quota:      quota,
	}, nil
}

func resolveRulePrice(model string, rule PerRequestPriceRule, mediaType, raw string) (string, float64, error) {
	raw = strings.TrimSpace(raw)
	resolution := rule.DefaultResolution
	if raw != "" {
		if configuredResolution, ok := matchConfiguredResolution(raw, rule.Prices); ok {
			resolution = configuredResolution
		} else if !rule.FallbackEnabled {
			return "", 0, fmt.Errorf("model %s: unknown %s resolution %q", model, mediaType, raw)
		}
	}
	if resolution == "" {
		return "", 0, fmt.Errorf("model %s: default resolution not configured", model)
	}
	unitPrice, ok := rule.Prices[resolution]
	if ok {
		return resolution, unitPrice, nil
	}
	if !rule.FallbackEnabled {
		return "", 0, fmt.Errorf("model %s: resolution %q not configured", model, resolution)
	}
	resolution = rule.DefaultResolution
	unitPrice, ok = rule.Prices[resolution]
	if !ok {
		return "", 0, fmt.Errorf("model %s: default resolution %q not configured", model, resolution)
	}
	return resolution, unitPrice, nil
}

func matchConfiguredResolution(raw string, prices map[string]float64) (string, bool) {
	raw = normalizeResolutionKey(raw)
	if raw == "" {
		return "", false
	}
	for resolution := range prices {
		if normalizeResolutionKey(resolution) == raw {
			return resolution, true
		}
	}
	return "", false
}

func normalizeResolutionKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), ""))
}

func parseSeconds(seconds string, duration int) (int, error) {
	seconds = strings.TrimSpace(seconds)
	if seconds != "" {
		parsed, err := strconv.Atoi(seconds)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("video seconds must be positive")
		}
		return parsed, nil
	}
	if duration <= 0 {
		return 0, fmt.Errorf("video seconds must be positive")
	}
	return duration, nil
}
