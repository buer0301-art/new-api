package gemini

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

const (
	veoReferenceTypeAsset  = "asset"
	veoStandardRefLimit    = 3
	veoFastFrameImageLimit = 2
)

func IsVeoFastModel(modelName string) bool {
	name := strings.ToLower(strings.TrimSpace(modelName))
	return strings.Contains(name, "veo-3.1-fast") || strings.Contains(name, "veo_3_1_fast")
}

func markGenerateAction(info *relaycommon.RelayInfo) {
	if info == nil {
		return
	}
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	info.Action = constant.TaskActionGenerate
}

func BuildVeoInstance(c *gin.Context, info *relaycommon.RelayInfo, req relaycommon.TaskSubmitReq) (VeoInstance, error) {
	instance := VeoInstance{Prompt: req.Prompt}

	if c != nil {
		if img := ExtractMultipartImage(c, info); img != nil {
			instance.Image = img
			return instance, nil
		}
	}

	refs := req.InputReference.NonEmpty()
	if len(refs) > 0 {
		limit := veoStandardRefLimit
		if IsVeoFastModel(req.Model) || req.FirstLastFrame {
			limit = veoFastFrameImageLimit
		}
		if len(refs) > limit {
			return instance, fmt.Errorf("too many input_reference images: got %d, max %d", len(refs), limit)
		}

		parsedRefs := make([]*VeoImageInput, 0, len(refs))
		for i, ref := range refs {
			parsed := ParseImageInput(ref)
			if parsed == nil {
				return instance, fmt.Errorf("invalid input_reference at index %d", i)
			}
			parsedRefs = append(parsedRefs, parsed)
		}
		markGenerateAction(info)

		if req.FirstLastFrame || IsVeoFastModel(req.Model) {
			instance.Image = parsedRefs[0]
			if len(parsedRefs) > 1 {
				instance.LastFrame = parsedRefs[1]
			}
			return instance, nil
		}

		instance.ReferenceImages = make([]VeoReferenceImage, 0, len(parsedRefs))
		for _, ref := range parsedRefs {
			instance.ReferenceImages = append(instance.ReferenceImages, VeoReferenceImage{
				Image:         ref,
				ReferenceType: veoReferenceTypeAsset,
			})
		}
		return instance, nil
	}

	if len(req.Images) > 0 {
		if parsed := ParseImageInput(req.Images[0]); parsed != nil {
			instance.Image = parsed
			markGenerateAction(info)
		}
	}

	return instance, nil
}

func BuildVeoParameters(req relaycommon.TaskSubmitReq) (*VeoParameters, error) {
	params := &VeoParameters{}
	if err := UnmarshalVeoMetadata(req.Metadata, params); err != nil {
		return nil, err
	}
	if params.DurationSeconds == 0 {
		if req.Duration > 0 {
			params.DurationSeconds = req.Duration
		} else if s, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil && s > 0 {
			params.DurationSeconds = s
		}
	}
	if params.Resolution == "" {
		switch {
		case strings.TrimSpace(req.Resolution) != "":
			params.Resolution = strings.ToLower(strings.TrimSpace(req.Resolution))
		case strings.TrimSpace(req.Size) != "":
			params.Resolution = SizeToVeoResolution(req.Size)
		}
	}
	if params.AspectRatio == "" {
		switch {
		case strings.TrimSpace(req.AspectRatio) != "":
			params.AspectRatio = strings.TrimSpace(req.AspectRatio)
		case strings.TrimSpace(req.Size) != "":
			params.AspectRatio = SizeToVeoAspectRatio(req.Size)
		}
	}
	params.Resolution = strings.ToLower(strings.TrimSpace(params.Resolution))
	params.SampleCount = 1
	return params, nil
}

func UnmarshalVeoMetadata(metadata map[string]any, params *VeoParameters) error {
	if params == nil {
		return nil
	}
	if metadata != nil && len(metadata) > 0 {
		raw, err := common.Marshal(metadata)
		if err != nil {
			return err
		}
		if err := common.Unmarshal(raw, params); err != nil {
			return err
		}
	}
	type veoMetadata struct {
		DurationSeconds int    `json:"durationSeconds,omitempty"`
		DurationSnake   int    `json:"duration_seconds,omitempty"`
		AspectRatio     string `json:"aspectRatio,omitempty"`
		AspectSnake     string `json:"aspect_ratio,omitempty"`
		Resolution      string `json:"resolution,omitempty"`
		NegativePrompt  string `json:"negativePrompt,omitempty"`
		NegativeSnake   string `json:"negative_prompt,omitempty"`
	}
	meta := &veoMetadata{}
	if metadata != nil && len(metadata) > 0 {
		raw, err := common.Marshal(metadata)
		if err != nil {
			return err
		}
		if err := common.Unmarshal(raw, meta); err != nil {
			return err
		}
	}
	if meta.DurationSeconds == 0 {
		meta.DurationSeconds = meta.DurationSnake
	}
	if meta.AspectRatio == "" {
		meta.AspectRatio = meta.AspectSnake
	}
	if meta.DurationSeconds > 0 {
		params.DurationSeconds = meta.DurationSeconds
	}
	if meta.AspectRatio != "" {
		params.AspectRatio = meta.AspectRatio
	}
	if meta.Resolution != "" {
		params.Resolution = meta.Resolution
	}
	if meta.NegativePrompt == "" {
		meta.NegativePrompt = meta.NegativeSnake
	}
	if meta.NegativePrompt != "" {
		params.NegativePrompt = meta.NegativePrompt
	}
	return nil
}
