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
		if configuredResolution, ok := matchConfiguredResolution(raw, mediaType, rule.Prices); ok {
			resolution = configuredResolution
		} else if !rule.FallbackEnabled {
			return "", 0, fmt.Errorf("model %s: unknown %s resolution %q", model, mediaType, raw)
		}
	}
	if resolution == "" {
		return "", 0, fmt.Errorf("model %s: default resolution not configured", model)
	}
	if configuredResolution, ok := matchNormalizedResolution(resolution, rule.Prices); ok {
		resolution = configuredResolution
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

func matchConfiguredResolution(raw string, mediaType string, prices map[string]float64) (string, bool) {
	normalized := normalizeResolutionKey(raw)
	if normalized == "" {
		return "", false
	}

	candidates := []string{normalized}
	candidates = append(candidates, resolutionAliasCandidates(mediaType, normalized)...)
	for _, candidate := range candidates {
		if resolution, ok := matchNormalizedResolution(candidate, prices); ok {
			return resolution, true
		}
	}
	return "", false
}

func matchNormalizedResolution(raw string, prices map[string]float64) (string, bool) {
	normalized := normalizeResolutionKey(raw)
	if normalized == "" {
		return "", false
	}
	for resolution := range prices {
		if normalizeResolutionKey(resolution) == normalized {
			return resolution, true
		}
	}
	return "", false
}

func resolutionAliasCandidates(mediaType string, normalized string) []string {
	switch mediaType {
	case MediaTypeImage:
		return imageResolutionAliasCandidates(normalized)
	case MediaTypeVideo:
		return videoResolutionAliasCandidates(normalized)
	default:
		return nil
	}
}

func imageResolutionAliasCandidates(normalized string) []string {
	switch normalized {
	case "1024x1024", "1024x1536", "1536x1024":
		return []string{"1k"}
	case "4096x4096", "3840x2160", "2160x3840":
		return []string{"4k"}
	}

	width, height, ok := parsePixelResolution(normalized)
	if !ok {
		return nil
	}
	longSide, _ := resolutionSides(width, height)
	switch {
	case longSide > 2560:
		return []string{"4k"}
	case longSide > 1536:
		return []string{"2k"}
	}
	area := width * height
	switch {
	case area <= 1024*1536:
		return []string{"1k"}
	case area <= 2560*1440:
		return []string{"2k"}
	default:
		return []string{"4k"}
	}
}

func parsePixelResolution(normalized string) (int, int, bool) {
	parts := strings.Split(normalized, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil || width <= 0 {
		return 0, 0, false
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func videoResolutionAliasCandidates(normalized string) []string {
	if width, height, ok := parsePixelResolution(normalized); ok {
		return videoResolutionCandidatesFromDimensions(width, height)
	}

	switch normalized {
	case "480", "480p":
		return videoResolutionCandidatesFromTier(0, true)
	case "980", "980p", "980k":
		return videoResolutionCandidatesFromTier(1, true)
	case "1k", "1080", "1080p":
		return videoResolutionCandidatesFromTier(2, true)
	case "2k", "1440", "1440p":
		return videoResolutionCandidatesFromTier(3, true)
	case "4k", "2160", "2160p":
		return videoResolutionCandidatesFromTier(4, true)
	}

	label := strings.TrimSuffix(normalized, "p")
	height, err := strconv.Atoi(label)
	if err != nil || height <= 0 {
		return nil
	}
	return videoResolutionCandidatesFromHeight(height)
}

func videoResolutionCandidatesFromDimensions(width int, height int) []string {
	longSide, shortSide := resolutionSides(width, height)

	if shortSide <= 0 {
		return nil
	}
	switch {
	case longSide > 2560:
		return videoResolutionCandidatesFromTier(4, true)
	case longSide > 1920:
		return videoResolutionCandidatesFromTier(3, true)
	case shortSide <= 480:
		return videoResolutionCandidatesFromTier(0, true)
	case shortSide <= 980:
		return videoResolutionCandidatesFromTier(1, true)
	case shortSide <= 1080:
		return videoResolutionCandidatesFromTier(2, true)
	case shortSide <= 1440 || longSide <= 2560:
		return videoResolutionCandidatesFromTier(3, true)
	default:
		return videoResolutionCandidatesFromTier(4, true)
	}
}

func videoResolutionCandidatesFromHeight(height int) []string {
	switch {
	case height <= 480:
		return videoResolutionCandidatesFromTier(0, true)
	case height <= 980:
		return videoResolutionCandidatesFromTier(1, true)
	case height <= 1080:
		return videoResolutionCandidatesFromTier(2, true)
	case height <= 1440:
		return videoResolutionCandidatesFromTier(3, true)
	default:
		return videoResolutionCandidatesFromTier(4, true)
	}
}

func videoResolutionCandidatesFromTier(tier int, includeHigherTiers bool) []string {
	tierCandidates := [][]string{
		{"480", "480p"},
		{"980", "980p", "980k"},
		{"1k", "1080p", "1080"},
		{"2k", "1440p", "1440"},
		{"4k", "2160p", "2160"},
	}
	if tier < 0 || tier >= len(tierCandidates) {
		return nil
	}
	if !includeHigherTiers {
		return append([]string{}, tierCandidates[tier]...)
	}
	candidates := make([]string, 0)
	for _, values := range tierCandidates[tier:] {
		candidates = append(candidates, values...)
	}
	return candidates
}

func resolutionSides(width int, height int) (int, int) {
	longSide := width
	shortSide := height
	if shortSide > longSide {
		longSide, shortSide = shortSide, longSide
	}
	return longSide, shortSide
}

func normalizeResolutionKey(value string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), ""))
	return strings.ReplaceAll(normalized, "*", "x")
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
