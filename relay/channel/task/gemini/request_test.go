package gemini

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const tinyPNGDataURI = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4//8/AwAI/AL+X7xL9QAAAABJRU5ErkJggg=="

func TestBuildVeoInstance_StandardUsesReferenceImages(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	req := relaycommon.TaskSubmitReq{
		Model:          "veo-3.1",
		Prompt:         "keep subject consistent",
		InputReference: relaycommon.FlexibleStringArray{tinyPNGDataURI, tinyPNGDataURI},
	}

	instance, err := BuildVeoInstance(nil, info, req)
	if err != nil {
		t.Fatalf("BuildVeoInstance returned error: %v", err)
	}
	if instance.Image != nil {
		t.Fatalf("expected no first-frame image for standard reference mode")
	}
	if instance.LastFrame != nil {
		t.Fatalf("expected no lastFrame for standard reference mode")
	}
	if len(instance.ReferenceImages) != 2 {
		t.Fatalf("referenceImages len = %d, want 2", len(instance.ReferenceImages))
	}
	if instance.ReferenceImages[0].ReferenceType != "asset" {
		t.Fatalf("referenceType = %q, want asset", instance.ReferenceImages[0].ReferenceType)
	}
	if info.Action != constant.TaskActionGenerate {
		t.Fatalf("action = %q, want %q", info.Action, constant.TaskActionGenerate)
	}
}

func TestBuildVeoInstance_FastUsesFirstAndLastFrame(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	req := relaycommon.TaskSubmitReq{
		Model:          "veo-3.1-fast",
		Prompt:         "transition between two scenes",
		FirstLastFrame: true,
		InputReference: relaycommon.FlexibleStringArray{tinyPNGDataURI, tinyPNGDataURI},
	}

	instance, err := BuildVeoInstance(nil, info, req)
	if err != nil {
		t.Fatalf("BuildVeoInstance returned error: %v", err)
	}
	if instance.Image == nil {
		t.Fatalf("expected first-frame image for fast mode")
	}
	if instance.LastFrame == nil {
		t.Fatalf("expected lastFrame for fast mode")
	}
	if len(instance.ReferenceImages) != 0 {
		t.Fatalf("referenceImages len = %d, want 0", len(instance.ReferenceImages))
	}
}

func TestParseImageInput_DataURI(t *testing.T) {
	parsed := ParseImageInput(tinyPNGDataURI)
	if parsed == nil {
		t.Fatalf("ParseImageInput returned nil for data URI")
	}
	if parsed.MimeType != "image/png" {
		t.Fatalf("mimeType = %q, want image/png", parsed.MimeType)
	}
	if parsed.BytesBase64Encoded == "" {
		t.Fatalf("expected non-empty base64 payload")
	}
}
