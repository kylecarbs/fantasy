package openai

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/openai-go/responses"
	"github.com/stretchr/testify/require"
)

func TestIsResponsesModel_ComputerUse(t *testing.T) {
	t.Parallel()

	require.True(t, IsResponsesModel("gpt-4o"))
	require.True(t, IsResponsesModel("computer-use-preview"))
	require.True(t, IsResponsesModel("computer-use-preview-2025-03-11"))
	require.False(t, IsResponsesModel("acme-computer-use-preview"))
	require.False(t, IsResponsesModel("not-a-computer-use-model"))
	require.False(t, IsResponsesModel("not-a-responses-model"))
}

func TestPrepareParams_ComputerUseRequiresStore(t *testing.T) {
	t.Parallel()

	lm := responsesLanguageModel{
		provider: Name,
		modelID:  string(responses.ResponsesModelComputerUsePreview),
	}
	tool := NewComputerUseTool(ComputerUseToolOptions{
		DisplayWidthPx:  1024,
		DisplayHeightPx: 768,
		Environment:     responses.ComputerUsePreviewToolEnvironmentBrowser,
	}, func(context.Context, fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.ToolResponse{}, nil
	})
	prompt := fantasy.Prompt{testTextMessage(fantasy.MessageRoleUser, "take a screenshot")}

	t.Run("missing provider options", func(t *testing.T) {
		t.Parallel()

		_, _, err := lm.prepareParams(fantasy.Call{Prompt: prompt, Tools: []fantasy.Tool{tool}})
		require.EqualError(t, err, computerUseStoreError)
	})

	t.Run("store false", func(t *testing.T) {
		t.Parallel()

		_, _, err := lm.prepareParams(fantasy.Call{
			Prompt: prompt,
			Tools:  []fantasy.Tool{tool},
			ProviderOptions: fantasy.ProviderOptions{
				Name: &ResponsesProviderOptions{Store: fantasy.Opt(false)},
			},
		})
		require.EqualError(t, err, computerUseStoreError)
	})

	t.Run("store true", func(t *testing.T) {
		t.Parallel()

		params, warnings, err := lm.prepareParams(fantasy.Call{
			Prompt: prompt,
			Tools:  []fantasy.Tool{tool},
			ProviderOptions: fantasy.ProviderOptions{
				Name: &ResponsesProviderOptions{Store: fantasy.Opt(true)},
			},
		})
		require.NoError(t, err)
		require.Empty(t, warnings)
		require.True(t, params.Store.Valid())
		require.True(t, params.Store.Value)
		require.Len(t, params.Tools, 1)
		require.NotNil(t, params.Tools[0].OfComputerUsePreview)
	})
}

func TestToResponsesTools_ComputerUsePreview(t *testing.T) {
	t.Parallel()

	displayNumber := int64(3)
	tool := NewComputerUseTool(ComputerUseToolOptions{
		DisplayWidthPx:  1440,
		DisplayHeightPx: 900,
		DisplayNumber:   &displayNumber,
		Environment:     responses.ComputerUsePreviewToolEnvironmentUbuntu,
	}, func(context.Context, fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.ToolResponse{}, nil
	})

	definition := tool.Definition()
	argsJSON, err := json.Marshal(definition.Args)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(argsJSON, &definition.Args))

	tools, toolChoice, warnings := toResponsesTools([]fantasy.Tool{definition}, nil, nil)
	require.Empty(t, warnings)
	require.Empty(t, toolChoice)
	require.Len(t, tools, 1)
	require.NotNil(t, tools[0].OfComputerUsePreview)
	require.Equal(t, int64(900), tools[0].OfComputerUsePreview.DisplayHeight)
	require.Equal(t, int64(1440), tools[0].OfComputerUsePreview.DisplayWidth)
	require.Equal(t, responses.ComputerUsePreviewToolEnvironmentUbuntu, tools[0].OfComputerUsePreview.Environment)
}

func TestResponsesToPrompt_ComputerUseWithStore(t *testing.T) {
	t.Parallel()

	prompt := fantasy.Prompt{
		{
			Role: fantasy.MessageRoleUser,
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: "take a screenshot"},
			},
		},
		{
			Role: fantasy.MessageRoleAssistant,
			Content: []fantasy.MessagePart{
				fantasy.ToolCallPart{
					ToolCallID: "comp_item_01",
					ToolName:   computerUseAPIName,
					Input:      `{"type":"screenshot"}`,
					ProviderOptions: fantasy.ProviderOptions{
						Name: &OpenAIComputerUseCallMetadata{
							CallID: "call_01",
							PendingSafetyChecks: []OpenAIComputerUsePendingSafetyCheck{
								{ID: "safe_01", Code: "account_access", Message: "Confirm access."},
							},
						},
					},
				},
			},
		},
		{
			Role: fantasy.MessageRoleTool,
			Content: []fantasy.MessagePart{
				NewComputerUseScreenshotResultWithMediaType("comp_item_01", "ZmFrZQ==", "image/jpeg"),
			},
		},
	}

	input, warnings := toResponsesPrompt(prompt, "system", true)
	require.Empty(t, warnings)
	require.Len(t, input, 3)
	require.NotNil(t, input[1].OfItemReference)
	require.Equal(t, "comp_item_01", input[1].OfItemReference.ID)

	computerOutput := input[2].OfComputerCallOutput
	require.NotNil(t, computerOutput)
	require.Equal(t, "call_01", computerOutput.CallID)
	require.True(t, computerOutput.Output.ImageURL.Valid())
	require.Equal(t, "data:image/jpeg;base64,ZmFrZQ==", computerOutput.Output.ImageURL.Value)
	require.Len(t, computerOutput.AcknowledgedSafetyChecks, 1)
	require.Equal(t, "safe_01", computerOutput.AcknowledgedSafetyChecks[0].ID)
	require.True(t, computerOutput.AcknowledgedSafetyChecks[0].Code.Valid())
	require.Equal(t, "account_access", computerOutput.AcknowledgedSafetyChecks[0].Code.Value)
	require.True(t, computerOutput.AcknowledgedSafetyChecks[0].Message.Valid())
	require.Equal(t, "Confirm access.", computerOutput.AcknowledgedSafetyChecks[0].Message.Value)
}

func TestOpenAIComputerUseCallMetadata_JSON(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(OpenAIComputerUseCallMetadata{
		CallID: "call_01",
		PendingSafetyChecks: []OpenAIComputerUsePendingSafetyCheck{
			{ID: "safe_01", Code: "account_access", Message: "Confirm access."},
		},
	})
	require.NoError(t, err)

	decoded, err := fantasy.UnmarshalProviderMetadata(map[string]json.RawMessage{
		Name: encoded,
	})
	require.NoError(t, err)

	metadata, ok := decoded[Name].(*OpenAIComputerUseCallMetadata)
	require.True(t, ok)
	require.Equal(t, "call_01", metadata.CallID)
	require.Len(t, metadata.PendingSafetyChecks, 1)
	require.Equal(t, "safe_01", metadata.PendingSafetyChecks[0].ID)
	require.Equal(t, "account_access", metadata.PendingSafetyChecks[0].Code)
	require.Equal(t, "Confirm access.", metadata.PendingSafetyChecks[0].Message)
}

func TestResponsesGenerate_ComputerUseResponse(t *testing.T) {
	t.Parallel()

	server := newMockServer()
	defer server.close()
	server.response = mockResponsesComputerUseResponse()

	provider, err := New(
		WithAPIKey("test-api-key"),
		WithBaseURL(server.server.URL),
		WithUseResponsesAPI(),
	)
	require.NoError(t, err)

	model, err := provider.LanguageModel(context.Background(), string(responses.ResponsesModelComputerUsePreview))
	require.NoError(t, err)

	resp, err := model.Generate(context.Background(), fantasy.Call{
		Prompt: testPrompt,
		ProviderOptions: fantasy.ProviderOptions{
			Name: &ResponsesProviderOptions{Store: fantasy.Opt(true)},
		},
		Tools: []fantasy.Tool{fantasy.ProviderDefinedTool{ID: computerUseToolID, Name: computerUseAPIName}},
	})
	require.NoError(t, err)
	require.Equal(t, "/responses", server.calls[0].path)
	require.Equal(t, fantasy.FinishReasonToolCalls, resp.FinishReason)

	toolCalls := resp.Content.ToolCalls()
	require.Len(t, toolCalls, 1)
	assertComputerUseToolCall(t, toolCalls[0], `[
		{"type":"move","x":320,"y":240},
		{"type":"click","button":"left","x":320,"y":240}
	]`)
}

func TestResponsesStream_ComputerUseResponse(t *testing.T) {
	t.Parallel()

	chunks := []string{
		"event: response.output_item.added\n" +
			`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"computer_call","id":"comp_item_01","status":"in_progress"}}` + "\n\n",
		"event: response.output_item.done\n" +
			`data: {"type":"response.output_item.done","output_index":0,"item":{"type":"computer_call","id":"comp_item_01","call_id":"call_01","status":"completed","pending_safety_checks":[{"id":"safe_01","code":"account_access","message":"Confirm access."}],"actions":[{"type":"move","x":320,"y":240},{"type":"click","button":"left","x":320,"y":240}]}}` + "\n\n",
		"event: response.completed\n" +
			`data: {"type":"response.completed","response":{"id":"resp_01","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":4,"total_tokens":14}}}` + "\n\n",
	}

	sms := newStreamingMockServer()
	defer sms.close()
	sms.chunks = chunks

	provider, err := New(
		WithAPIKey("test-api-key"),
		WithBaseURL(sms.server.URL),
		WithUseResponsesAPI(),
	)
	require.NoError(t, err)

	model, err := provider.LanguageModel(context.Background(), string(responses.ResponsesModelComputerUsePreview))
	require.NoError(t, err)

	stream, err := model.Stream(context.Background(), fantasy.Call{
		Prompt: testPrompt,
		ProviderOptions: fantasy.ProviderOptions{
			Name: &ResponsesProviderOptions{Store: fantasy.Opt(true)},
		},
		Tools: []fantasy.Tool{fantasy.ProviderDefinedTool{ID: computerUseToolID, Name: computerUseAPIName}},
	})
	require.NoError(t, err)

	var (
		toolInputStarts []fantasy.StreamPart
		toolInputEnds   []fantasy.StreamPart
		toolCalls       []fantasy.StreamPart
		finishes        []fantasy.StreamPart
	)
	stream(func(part fantasy.StreamPart) bool {
		switch part.Type {
		case fantasy.StreamPartTypeToolInputStart:
			toolInputStarts = append(toolInputStarts, part)
		case fantasy.StreamPartTypeToolInputEnd:
			toolInputEnds = append(toolInputEnds, part)
		case fantasy.StreamPartTypeToolCall:
			toolCalls = append(toolCalls, part)
		case fantasy.StreamPartTypeFinish:
			finishes = append(finishes, part)
		}
		return true
	})

	require.Len(t, toolInputStarts, 1)
	require.Equal(t, "comp_item_01", toolInputStarts[0].ID)
	require.Equal(t, computerUseAPIName, toolInputStarts[0].ToolCallName)

	require.Len(t, toolInputEnds, 1)
	require.Equal(t, "comp_item_01", toolInputEnds[0].ID)

	require.Len(t, toolCalls, 1)
	require.Equal(t, "comp_item_01", toolCalls[0].ID)
	require.Equal(t, computerUseAPIName, toolCalls[0].ToolCallName)
	require.JSONEq(t, `[
		{"type":"move","x":320,"y":240},
		{"type":"click","button":"left","x":320,"y":240}
	]`, toolCalls[0].ToolCallInput)

	metadata, ok := toolCalls[0].ProviderMetadata[Name].(*OpenAIComputerUseCallMetadata)
	require.True(t, ok)
	require.Equal(t, "call_01", metadata.CallID)
	require.Len(t, metadata.PendingSafetyChecks, 1)
	require.Equal(t, "safe_01", metadata.PendingSafetyChecks[0].ID)

	require.Len(t, finishes, 1)
	require.Equal(t, fantasy.FinishReasonToolCalls, finishes[0].FinishReason)
}

func mockResponsesComputerUseResponse() map[string]any {
	return map[string]any{
		"id":     "resp_01",
		"object": "response",
		"model":  string(responses.ResponsesModelComputerUsePreview),
		"output": []any{
			map[string]any{
				"type":    "computer_call",
				"id":      "comp_item_01",
				"call_id": "call_01",
				"status":  "completed",
				"pending_safety_checks": []any{
					map[string]any{
						"id":      "safe_01",
						"code":    "account_access",
						"message": "Confirm access.",
					},
				},
				"actions": []any{
					map[string]any{"type": "move", "x": 320, "y": 240},
					map[string]any{"type": "click", "button": "left", "x": 320, "y": 240},
				},
			},
		},
		"status": "completed",
		"usage": map[string]any{
			"input_tokens":  10,
			"output_tokens": 4,
			"total_tokens":  14,
		},
	}
}

func assertComputerUseToolCall(t *testing.T, toolCall fantasy.ToolCallContent, wantInput string) {
	t.Helper()

	require.False(t, toolCall.ProviderExecuted)
	require.Equal(t, computerUseAPIName, toolCall.ToolName)
	require.Equal(t, "comp_item_01", toolCall.ToolCallID)
	require.JSONEq(t, wantInput, toolCall.Input)

	metadata, ok := toolCall.ProviderMetadata[Name].(*OpenAIComputerUseCallMetadata)
	require.True(t, ok)
	require.Equal(t, "call_01", metadata.CallID)
	require.Len(t, metadata.PendingSafetyChecks, 1)
	require.Equal(t, "safe_01", metadata.PendingSafetyChecks[0].ID)
	require.Equal(t, "account_access", metadata.PendingSafetyChecks[0].Code)
	require.Equal(t, "Confirm access.", metadata.PendingSafetyChecks[0].Message)
}

func computerUseCassettePaths(t *testing.T, modelName string) []string {
	base := filepath.Join("testdata", t.Name(), modelName)
	return []string{
		filepath.Join(base, "computer_use.yaml"),
		filepath.Join(base, "computer_use_streaming.yaml"),
	}
}
