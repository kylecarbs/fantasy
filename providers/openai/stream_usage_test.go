package openai

import (
	"context"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

// Some OpenAI-compatible backends report cumulative usage on delta
// chunks and end with a usage-less finish chunk; the last
// usage-bearing chunk must win.
func TestStreamUsageSurvivesTrailingUsagelessChunk(t *testing.T) {
	t.Parallel()

	server := newStreamingMockServer()
	defer server.close()

	server.chunks = []string{
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"laguna-xs","choices":[{"index":0,"delta":{"role":"assistant","content":"Hel"},"finish_reason":null}],"usage":{"prompt_tokens":100,"completion_tokens":1,"total_tokens":101}}` + "\n\n",
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"laguna-xs","choices":[{"index":0,"delta":{"content":"lo"},"finish_reason":null}],"usage":{"prompt_tokens":100,"completion_tokens":5,"total_tokens":105,"prompt_tokens_details":{"cached_tokens":80}}}` + "\n\n",
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"laguna-xs","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":null}` + "\n\n",
		"data: [DONE]\n\n",
	}

	provider, err := New(
		WithAPIKey("test-api-key"),
		WithBaseURL(server.server.URL),
	)
	require.NoError(t, err)
	model, err := provider.LanguageModel(t.Context(), "laguna-xs")
	require.NoError(t, err)

	stream, err := model.Stream(context.Background(), fantasy.Call{Prompt: testPrompt})
	require.NoError(t, err)

	parts, err := collectStreamParts(stream)
	require.NoError(t, err)

	var finish *fantasy.StreamPart
	for i, part := range parts {
		if part.Type == fantasy.StreamPartTypeFinish {
			finish = &parts[i]
		}
	}
	require.NotNil(t, finish)
	require.Equal(t, fantasy.FinishReasonStop, finish.FinishReason)
	// prompt_tokens includes cached tokens; input is reported net of cache.
	require.Equal(t, int64(20), finish.Usage.InputTokens)
	require.Equal(t, int64(80), finish.Usage.CacheReadTokens)
	require.Equal(t, int64(5), finish.Usage.OutputTokens)
	require.Equal(t, int64(105), finish.Usage.TotalTokens)
}

func TestStreamObjectUsageSurvivesTrailingUsagelessChunk(t *testing.T) {
	t.Parallel()

	server := newStreamingMockServer()
	defer server.close()

	server.chunks = []string{
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"laguna-xs","choices":[{"index":0,"delta":{"role":"assistant","content":"{\"answer\":\"hello\"}"},"finish_reason":null}],"usage":{"prompt_tokens":100,"completion_tokens":5,"total_tokens":105,"prompt_tokens_details":{"cached_tokens":80}}}` + "\n\n",
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"laguna-xs","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":null}` + "\n\n",
		"data: [DONE]\n\n",
	}

	provider, err := New(
		WithAPIKey("test-api-key"),
		WithBaseURL(server.server.URL),
	)
	require.NoError(t, err)
	model, err := provider.LanguageModel(t.Context(), "laguna-xs")
	require.NoError(t, err)

	stream, err := model.StreamObject(context.Background(), fantasy.ObjectCall{
		Prompt: testPrompt,
		Schema: fantasy.Schema{
			Type: "object",
			Properties: map[string]*fantasy.Schema{
				"answer": {Type: "string"},
			},
			Required: []string{"answer"},
		},
	})
	require.NoError(t, err)

	parts := collectObjectStreamParts(stream)
	require.NotEmpty(t, parts)
	finish := parts[len(parts)-1]
	require.Equal(t, fantasy.ObjectStreamPartTypeFinish, finish.Type)
	require.Equal(t, int64(20), finish.Usage.InputTokens)
	require.Equal(t, int64(80), finish.Usage.CacheReadTokens)
	require.Equal(t, int64(5), finish.Usage.OutputTokens)
	require.Equal(t, int64(105), finish.Usage.TotalTokens)
}
