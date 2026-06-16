package bedrock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func TestNormalizeModelID(t *testing.T) {
	const legacyModel = "anthropic.claude-haiku-4-5-20251001-v1:0"

	tests := []struct {
		name           string
		envRegion      string
		explicitRegion string
		modelID        string
		want           string
	}{
		{
			name:      "qualified us passes through",
			envRegion: "eu-west-1",
			modelID:   "us.anthropic.claude-haiku-4-5-20251001-v1:0",
			want:      "us.anthropic.claude-haiku-4-5-20251001-v1:0",
		},
		{
			name:      "qualified eu passes through",
			envRegion: "us-east-1",
			modelID:   "eu.anthropic.claude-haiku-4-5-20251001-v1:0",
			want:      "eu.anthropic.claude-haiku-4-5-20251001-v1:0",
		},
		{
			name:      "qualified apac passes through",
			envRegion: "us-east-1",
			modelID:   "apac.anthropic.claude-haiku-4-5-20251001-v1:0",
			want:      "apac.anthropic.claude-haiku-4-5-20251001-v1:0",
		},
		{
			name:      "qualified global passes through",
			envRegion: "us-east-1",
			modelID:   "global.anthropic.claude-haiku-4-5-20251001-v1:0",
			want:      "global.anthropic.claude-haiku-4-5-20251001-v1:0",
		},
		{
			name:    "legacy model defaults to us",
			modelID: legacyModel,
			want:    "us." + legacyModel,
		},
		{
			name:      "aws region maps to eu",
			envRegion: "eu-west-1",
			modelID:   legacyModel,
			want:      "eu." + legacyModel,
		},
		{
			name:      "aws region maps to apac",
			envRegion: "ap-northeast-1",
			modelID:   legacyModel,
			want:      "apac." + legacyModel,
		},
		{
			name:      "gov region maps to us",
			envRegion: "us-gov-west-1",
			modelID:   legacyModel,
			want:      "us." + legacyModel,
		},
		{
			name:           "explicit region wins over env",
			envRegion:      "us-east-1",
			explicitRegion: "eu-central-1",
			modelID:        legacyModel,
			want:           "eu." + legacyModel,
		},
		{
			name:      "unknown region falls back to us",
			envRegion: "ca-central-1",
			modelID:   legacyModel,
			want:      "us." + legacyModel,
		},
		{
			name:    "non anthropic models pass through",
			modelID: "amazon.nova-pro-v1:0",
			want:    "amazon.nova-pro-v1:0",
		},
		{
			name:    "empty model passes through",
			modelID: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AWS_REGION", tt.envRegion)
			require.Equal(t, tt.want, normalizeModelID(tt.modelID, tt.explicitRegion))
		})
	}
}

func TestLanguageModelUsesNormalizedModelID(t *testing.T) {
	prompt := fantasy.Prompt{{
		Role:    fantasy.MessageRoleUser,
		Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Hi"}},
	}}

	tests := []struct {
		name           string
		envRegion      string
		explicitRegion string
		modelID        string
		wantPath       string
	}{
		{
			name:           "legacy model is normalized before request",
			explicitRegion: "eu-central-1",
			modelID:        "anthropic.claude-haiku-4-5-20251001-v1:0",
			wantPath:       "/model/eu.anthropic.claude-haiku-4-5-20251001-v1:0/invoke",
		},
		{
			name:           "qualified model is preserved",
			explicitRegion: "us-east-1",
			modelID:        "global.anthropic.claude-haiku-4-5-20251001-v1:0",
			wantPath:       "/model/global.anthropic.claude-haiku-4-5-20251001-v1:0/invoke",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AWS_REGION", tt.envRegion)

			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(mockAnthropicResponse())
			}))
			defer server.Close()

			p, err := New(
				WithAPIKey("k"),
				WithSkipAuth(true),
				WithHTTPClient(&http.Client{Transport: redirectTransport(server.URL)}),
				WithRegion(tt.explicitRegion),
			)
			require.NoError(t, err)

			model, err := p.LanguageModel(t.Context(), tt.modelID)
			require.NoError(t, err)
			_, err = model.Generate(t.Context(), fantasy.Call{Prompt: prompt})
			require.NoError(t, err)

			require.Len(t, paths, 1)
			require.Equal(t, tt.wantPath, paths[0])
		})
	}
}
