package anthropic

import (
	"testing"

	fantasy "charm.land/fantasy"
)

func TestMapFinishReasonRefusal(t *testing.T) {
	t.Parallel()

	cases := map[string]fantasy.FinishReason{
		"end_turn":      fantasy.FinishReasonStop,
		"pause_turn":    fantasy.FinishReasonStop,
		"stop_sequence": fantasy.FinishReasonStop,
		"max_tokens":    fantasy.FinishReasonLength,
		"tool_use":      fantasy.FinishReasonToolCalls,
		"refusal":       fantasy.FinishReasonContentFilter,
		"":              fantasy.FinishReasonUnknown,
		"something_new": fantasy.FinishReasonUnknown,
	}
	for in, want := range cases {
		if got := mapFinishReason(in); got != want {
			t.Errorf("mapFinishReason(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseAnthropicRefusal(t *testing.T) {
	t.Parallel()

	t.Run("Refusal", func(t *testing.T) {
		t.Parallel()
		delta := `{"stop_reason":"refusal","stop_details":{"type":"refusal","category":"cyber","explanation":"blocked under policy"}}`
		got := parseAnthropicRefusal(delta)
		if got == nil {
			t.Fatal("expected refusal metadata, got nil")
		}
		if got.Category != "cyber" {
			t.Errorf("category = %q, want cyber", got.Category)
		}
		if got.Explanation != "blocked under policy" {
			t.Errorf("explanation = %q, want %q", got.Explanation, "blocked under policy")
		}
	})

	t.Run("NoStopDetails", func(t *testing.T) {
		t.Parallel()
		if got := parseAnthropicRefusal(`{"stop_reason":"end_turn"}`); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("EmptyAndInvalid", func(t *testing.T) {
		t.Parallel()
		if got := parseAnthropicRefusal(""); got != nil {
			t.Errorf("expected nil for empty, got %+v", got)
		}
		if got := parseAnthropicRefusal("not json"); got != nil {
			t.Errorf("expected nil for invalid, got %+v", got)
		}
	})
}

func TestRefusalMetadataRoundTrip(t *testing.T) {
	t.Parallel()

	in := refusalProviderMetadata(&RefusalMetadata{Category: "cyber", Explanation: "blocked"})
	got := GetRefusalMetadata(in)
	if got == nil {
		t.Fatal("expected refusal metadata, got nil")
	}
	if got.Category != "cyber" || got.Explanation != "blocked" {
		t.Errorf("unexpected metadata: %+v", got)
	}
	if GetRefusalMetadata(refusalProviderMetadata(nil)) != nil {
		t.Error("expected nil metadata for nil refusal")
	}
}
