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

func NormalizeResolution(mediaType, raw string) (string, bool) {
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", false
	}
	switch mediaType {
	case MediaTypeImage:
		switch raw {
		case "1k", "1024x1024", "1024x1536", "1536x1024":
			return "1K", true
		case "2k", "2048x2048":
			return "2K", true
		case "4k", "4096x4096", "3840x2160", "2160x3840":
			return "4K", true
		default:
			if raw == "1k" || raw == "2k" || raw == "4k" {
				return strings.ToUpper(raw), true
			}
		}
	case MediaTypeVideo:
		switch {
		case raw == "480" || raw == "480p" || strings.Contains(raw, "480"):
			return "480", true
		case raw == "980" || raw == "980p" || strings.Contains(raw, "980"):
			return "980", true
		case raw == "1k" || raw == "1080" || raw == "1080p" || strings.Contains(raw, "1080"):
			return "1K", true
		case raw == "2k" || strings.Contains(raw, "1440") || strings.Contains(raw, "2048"):
			return "2K", true
		case raw == "4k" || raw == "2160" || raw == "2160p" || strings.Contains(raw, "2160") || strings.Contains(raw, "3840") || strings.Contains(raw, "4096"):
			return "4K", true
		default:
			if raw == "1k" || raw == "2k" || raw == "4k" {
				return strings.ToUpper(raw), true
			}
		}
	}
	return "", false
}

func ResolveImagePricing(model string, rule PerRequestPriceRule, input ImagePricingInput) (*types.ResolvedPerRequestPricing, error) {
	if rule.MediaType != MediaTypeImage {
		return nil, fmt.Errorf("model %s: media type mismatch, expected %s", model, MediaTypeImage)
	}
	rawResolution := strings.TrimSpace(input.Size)
	resolution := rule.DefaultResolution
	if rawResolution != "" {
		normalized, ok := NormalizeResolution(MediaTypeImage, rawResolution)
		if !ok {
			if !rule.FallbackEnabled {
				return nil, fmt.Errorf("model %s: unknown image resolution %q", model, rawResolution)
			}
		} else {
			resolution = normalized
		}
	}
	if resolution == "" {
		if !rule.FallbackEnabled {
			return nil, fmt.Errorf("model %s: default resolution not configured", model)
		}
		resolution = rule.DefaultResolution
	}
	unitPrice, ok := rule.Prices[resolution]
	if !ok {
		if !rule.FallbackEnabled {
			return nil, fmt.Errorf("model %s: resolution %q not configured", model, resolution)
		}
		resolution = rule.DefaultResolution
		unitPrice, ok = rule.Prices[resolution]
		if !ok {
			return nil, fmt.Errorf("model %s: default resolution %q not configured", model, resolution)
		}
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
	rawResolution := strings.TrimSpace(input.Size)
	if rawResolution == "" {
		rawResolution = strings.TrimSpace(input.MetadataResolution)
	}
	resolution := rule.DefaultResolution
	if rawResolution != "" {
		normalized, ok := NormalizeResolution(MediaTypeVideo, rawResolution)
		if !ok {
			if !rule.FallbackEnabled {
				return nil, fmt.Errorf("model %s: unknown video resolution %q", model, rawResolution)
			}
		} else {
			resolution = normalized
		}
	}
	if resolution == "" {
		if !rule.FallbackEnabled {
			return nil, fmt.Errorf("model %s: default resolution not configured", model)
		}
		resolution = rule.DefaultResolution
	}
	unitPrice, ok := rule.Prices[resolution]
	if !ok {
		if !rule.FallbackEnabled {
			return nil, fmt.Errorf("model %s: resolution %q not configured", model, resolution)
		}
		resolution = rule.DefaultResolution
		unitPrice, ok = rule.Prices[resolution]
		if !ok {
			return nil, fmt.Errorf("model %s: default resolution %q not configured", model, resolution)
		}
	}
	seconds, err := parseSeconds(input.Seconds, input.Duration)
	if err != nil {
		return nil, err
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
