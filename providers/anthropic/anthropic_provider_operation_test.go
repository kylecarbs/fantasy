package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func TestGenerate_WebSearchResponseRejectsDuplicateProviderOperationIDs(t *testing.T) {
	t.Parallel()

	response := mockAnthropicWebSearchResponse()
	content, ok := response["content"].([]any)
	require.True(t, ok)
	response["content"] = []any{content[0], content[0], content[1]}

	server, _ := newAnthropicJSONServer(response)
	defer server.Close()

	provider, err := New(
		WithAPIKey("test-api-key"),
		WithBaseURL(server.URL),
	)
	require.NoError(t, err)

	model, err := provider.LanguageModel(context.Background(), "claude-sonnet-4-20250514")
	require.NoError(t, err)

	_, err = model.Generate(context.Background(), fantasy.Call{
		Prompt: testPrompt(),
		Tools: []fantasy.Tool{
			WebSearchTool(nil),
		},
	})
	require.ErrorContains(t, err, "duplicate ID")
}

func TestStream_WebSearchResponseRejectsOrphanResultBeforeSources(t *testing.T) {
	t.Parallel()

	parts := collectAnthropicStreamPartsFromChunks(t, concatAnthropicChunkSets(
		anthropicWebSearchMessageStartChunks(),
		anthropicWebSearchToolResultChunks(
			t,
			0,
			"srvtoolu_01",
			anthropicWebSearchResultItem(
				"https://example.com/orphan",
				"Orphan Result",
				"encrypted_orphan",
				"1 hour ago",
			),
		),
		anthropicWebSearchMessageStopChunks(),
	))

	var providerToolEvents []fantasy.StreamPart
	var sourceParts []fantasy.StreamPart
	var errorParts []fantasy.StreamPart
	for _, part := range parts {
		switch part.Type {
		case fantasy.StreamPartTypeToolInputStart,
			fantasy.StreamPartTypeToolInputEnd,
			fantasy.StreamPartTypeToolCall,
			fantasy.StreamPartTypeToolResult:
			if part.ProviderExecuted {
				providerToolEvents = append(providerToolEvents, part)
			}
		case fantasy.StreamPartTypeSource:
			sourceParts = append(sourceParts, part)
		case fantasy.StreamPartTypeError:
			errorParts = append(errorParts, part)
		}
	}

	require.Empty(t, providerToolEvents)
	require.Empty(t, sourceParts)
	require.Len(t, errorParts, 1)
	require.ErrorContains(t, errorParts[0].Error, "without a matching server tool use")
}

func TestStream_WebSearchResponseSkipsProviderInputDeltas(t *testing.T) {
	t.Parallel()

	parts := collectAnthropicStreamPartsFromChunks(t, concatAnthropicChunkSets(
		anthropicWebSearchMessageStartChunks(),
		anthropicWebSearchServerToolUseChunks(
			0,
			"srvtoolu_01",
			`{"query":"latest `,
			`AI news"}`,
		),
		anthropicWebSearchToolResultChunks(
			t,
			1,
			"srvtoolu_01",
			anthropicWebSearchResultItem(
				"https://example.com/ai-news",
				"Latest AI News",
				"encrypted_abc123",
				"2 hours ago",
			),
		),
		anthropicWebSearchMessageStopChunks(),
	))

	var inputDeltaParts []fantasy.StreamPart
	var providerToolCalls []fantasy.StreamPart
	for _, part := range parts {
		switch part.Type {
		case fantasy.StreamPartTypeToolInputDelta:
			inputDeltaParts = append(inputDeltaParts, part)
		case fantasy.StreamPartTypeToolCall:
			if part.ProviderExecuted {
				providerToolCalls = append(providerToolCalls, part)
			}
		}
	}

	require.Empty(t, inputDeltaParts)
	require.Len(t, providerToolCalls, 1)
	require.JSONEq(t, `{"query":"latest AI news"}`, providerToolCalls[0].ToolCallInput)
}

func TestStream_WebSearchResponseHandlesMultipleProviderOperations(t *testing.T) {
	t.Parallel()

	parts := collectAnthropicStreamPartsFromChunks(t, concatAnthropicChunkSets(
		anthropicWebSearchMessageStartChunks(),
		anthropicWebSearchServerToolUseChunks(0, "srvtoolu_01"),
		anthropicWebSearchToolResultChunks(
			t,
			1,
			"srvtoolu_01",
			anthropicWebSearchResultItem(
				"https://example.com/first",
				"First Result",
				"encrypted_first",
				"1 hour ago",
			),
		),
		anthropicWebSearchServerToolUseChunks(2, "srvtoolu_02"),
		anthropicWebSearchToolResultChunks(
			t,
			3,
			"srvtoolu_02",
			anthropicWebSearchResultItem(
				"https://example.com/second",
				"Second Result",
				"encrypted_second",
				"3 hours ago",
			),
		),
		anthropicWebSearchMessageStopChunks(),
	))

	var providerToolCallIDs []string
	var providerToolResultIDs []string
	for _, part := range parts {
		if !part.ProviderExecuted {
			continue
		}
		switch part.Type {
		case fantasy.StreamPartTypeToolCall:
			providerToolCallIDs = append(providerToolCallIDs, part.ID)
		case fantasy.StreamPartTypeToolResult:
			providerToolResultIDs = append(providerToolResultIDs, part.ID)
		}
	}

	require.Equal(t, []string{"srvtoolu_01", "srvtoolu_02"}, providerToolCallIDs)
	require.Equal(t, []string{"srvtoolu_01", "srvtoolu_02"}, providerToolResultIDs)
}

func TestStream_WebSearchResponseRejectsDuplicateProviderOperationIDs(t *testing.T) {
	t.Parallel()

	parts := collectAnthropicStreamPartsFromChunks(t, concatAnthropicChunkSets(
		anthropicWebSearchMessageStartChunks(),
		anthropicWebSearchServerToolUseChunks(0, "srvtoolu_01"),
		anthropicWebSearchServerToolUseChunks(1, "srvtoolu_01"),
		anthropicWebSearchMessageStopChunks(),
	))

	var providerToolEvents []fantasy.StreamPart
	var errorParts []fantasy.StreamPart
	for _, part := range parts {
		switch part.Type {
		case fantasy.StreamPartTypeToolInputStart,
			fantasy.StreamPartTypeToolInputEnd,
			fantasy.StreamPartTypeToolCall,
			fantasy.StreamPartTypeToolResult:
			if part.ProviderExecuted {
				providerToolEvents = append(providerToolEvents, part)
			}
		case fantasy.StreamPartTypeError:
			errorParts = append(errorParts, part)
		}
	}

	require.Empty(t, providerToolEvents)
	require.Len(t, errorParts, 1)
	require.ErrorContains(t, errorParts[0].Error, "duplicate ID")
}

func TestStream_WebSearchResponseSurfacesIncompleteOperationOnStreamError(t *testing.T) {
	t.Parallel()

	parts := collectAnthropicStreamPartsFromChunks(t, concatAnthropicChunkSets(
		anthropicWebSearchMessageStartChunks(),
		anthropicWebSearchServerToolUseChunks(0, "srvtoolu_01"),
		[]string{
			"event: content_block_delta\n",
			"data: {not-json}\n\n",
		},
	))

	var providerToolEvents []fantasy.StreamPart
	var errorParts []fantasy.StreamPart
	for _, part := range parts {
		switch part.Type {
		case fantasy.StreamPartTypeToolInputStart,
			fantasy.StreamPartTypeToolInputEnd,
			fantasy.StreamPartTypeToolCall,
			fantasy.StreamPartTypeToolResult:
			if part.ProviderExecuted {
				providerToolEvents = append(providerToolEvents, part)
			}
		case fantasy.StreamPartTypeError:
			errorParts = append(errorParts, part)
		}
	}

	require.Empty(t, providerToolEvents)
	require.Len(t, errorParts, 1)
	require.ErrorContains(t, errorParts[0].Error, "ended without a matching result")
	require.ErrorContains(t, errorParts[0].Error, "invalid character")
}

func TestStream_WebSearchResponsePreservesProviderMetadata(t *testing.T) {
	t.Parallel()

	parts := collectAnthropicStreamPartsFromChunks(t, concatAnthropicChunkSets(
		anthropicWebSearchMessageStartChunks(),
		anthropicWebSearchServerToolUseChunks(0, "srvtoolu_01"),
		anthropicWebSearchToolResultChunks(
			t,
			1,
			"srvtoolu_01",
			anthropicWebSearchResultItem(
				"https://example.com/ai-news",
				"Latest AI News",
				"encrypted_abc123",
				"2 hours ago",
			),
			anthropicWebSearchResultItem(
				"https://example.com/ml-update",
				"ML Update",
				"encrypted_def456",
				"",
			),
		),
		anthropicWebSearchMessageStopChunks(),
	))

	var providerToolResults []fantasy.StreamPart
	for _, part := range parts {
		if part.Type == fantasy.StreamPartTypeToolResult && part.ProviderExecuted {
			providerToolResults = append(providerToolResults, part)
		}
	}

	require.Len(t, providerToolResults, 1)
	searchMeta, ok := providerToolResults[0].ProviderMetadata[Name]
	require.True(t, ok)
	webMeta, ok := searchMeta.(*WebSearchResultMetadata)
	require.True(t, ok)
	require.Len(t, webMeta.Results, 2)
	require.Equal(t, "https://example.com/ai-news", webMeta.Results[0].URL)
	require.Equal(t, "encrypted_abc123", webMeta.Results[0].EncryptedContent)
	require.Equal(t, "2 hours ago", webMeta.Results[0].PageAge)
	require.Equal(t, "https://example.com/ml-update", webMeta.Results[1].URL)
	require.Equal(t, "encrypted_def456", webMeta.Results[1].EncryptedContent)
	require.Empty(t, webMeta.Results[1].PageAge)
}

func concatAnthropicChunkSets(chunkSets ...[]string) []string {
	var chunks []string
	for _, chunkSet := range chunkSets {
		chunks = append(chunks, chunkSet...)
	}
	return chunks
}

func anthropicWebSearchMessageStartChunks() []string {
	return []string{
		"event: message_start\n",
		`data: {"type":"message_start","message":{"id":"msg_01WebSearch","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":100,"output_tokens":0}}}` + "\n\n",
	}
}

func anthropicWebSearchMessageStopChunks() []string {
	return []string{
		"event: message_stop\n",
		`data: {"type":"message_stop"}` + "\n\n",
	}
}

func anthropicWebSearchServerToolUseChunks(index int, id string, partialJSONDeltas ...string) []string {
	chunks := []string{
		"event: content_block_start\n",
		fmt.Sprintf(
			`data: {"type":"content_block_start","index":%d,"content_block":{"type":"server_tool_use","id":%q,"name":"web_search","input":{}}}`,
			index,
			id,
		) + "\n\n",
	}
	for _, delta := range partialJSONDeltas {
		chunks = append(chunks,
			"event: content_block_delta\n",
			fmt.Sprintf(
				`data: {"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":%q}}`,
				index,
				delta,
			)+"\n\n",
		)
	}
	chunks = append(chunks,
		"event: content_block_stop\n",
		fmt.Sprintf(`data: {"type":"content_block_stop","index":%d}`, index)+"\n\n",
	)
	return chunks
}

func anthropicWebSearchToolResultChunks(t *testing.T, index int, toolUseID string, items ...map[string]any) []string {
	t.Helper()

	if items == nil {
		items = []map[string]any{}
	}
	content, err := json.Marshal(items)
	require.NoError(t, err)

	return []string{
		"event: content_block_start\n",
		fmt.Sprintf(
			`data: {"type":"content_block_start","index":%d,"content_block":{"type":"web_search_tool_result","tool_use_id":%q,"content":%s}}`,
			index,
			toolUseID,
			string(content),
		) + "\n\n",
		"event: content_block_stop\n",
		fmt.Sprintf(`data: {"type":"content_block_stop","index":%d}`, index) + "\n\n",
	}
}

func anthropicWebSearchResultItem(url, title, encryptedContent, pageAge string) map[string]any {
	item := map[string]any{
		"type":              "web_search_result",
		"url":               url,
		"title":             title,
		"encrypted_content": encryptedContent,
	}
	if pageAge != "" {
		item["page_age"] = pageAge
	}
	return item
}
