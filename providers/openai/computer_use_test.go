package openai

import (
	"context"
	"encoding/json"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/openai-go/responses"
	"github.com/stretchr/testify/require"
)

func TestNewComputerUseTool(t *testing.T) {
	t.Parallel()

	tool := NewComputerUseTool(func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.NewTextResponse("stub"), nil
	})

	require.Equal(t, "openai.computer_use", tool.Definition().ID)
	require.Equal(t, "computer", tool.Definition().Name)
	require.Equal(t, fantasy.ToolTypeProviderDefined, tool.GetType())
}

func TestIsComputerUseTool(t *testing.T) {
	t.Parallel()

	stub := func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.NewTextResponse("stub"), nil
	}

	tests := []struct {
		name string
		tool fantasy.Tool
		want bool
	}{
		{
			name: "ExecutableProviderTool",
			tool: NewComputerUseTool(stub),
			want: true,
		},
		{
			name: "bare ProviderDefinedTool",
			tool: fantasy.ProviderDefinedTool{ID: "openai.computer_use", Name: "computer"},
			want: true,
		},
		{
			name: "FunctionTool named computer",
			tool: fantasy.FunctionTool{Name: "computer"},
			want: false,
		},
		{
			name: "different provider PDT",
			tool: fantasy.ProviderDefinedTool{ID: "anthropic.computer", Name: "computer"},
			want: false,
		},
		{
			name: "web search tool",
			tool: WebSearchTool(nil),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, IsComputerUseTool(tt.tool))
		})
	}
}

func TestComputerUseMetadataRoundTrip(t *testing.T) {
	t.Parallel()

	original := &ComputerUseMetadata{RawJSON: `{"some":"data"}`}

	encoded, err := json.Marshal(original)
	require.NoError(t, err)
	require.Contains(t, string(encoded), TypeComputerUseMetadata)

	decoded, err := fantasy.UnmarshalProviderMetadata(map[string]json.RawMessage{
		Name: encoded,
	})
	require.NoError(t, err)

	restored, ok := decoded[Name].(*ComputerUseMetadata)
	require.True(t, ok)
	require.Equal(t, original.RawJSON, restored.RawJSON)
}

func TestGetComputerUseMetadata(t *testing.T) {
	t.Parallel()

	t.Run("present", func(t *testing.T) {
		t.Parallel()
		meta := &ComputerUseMetadata{RawJSON: "x"}
		opts := fantasy.ProviderOptions{Name: meta}
		got := GetComputerUseMetadata(opts)
		require.NotNil(t, got)
		require.Equal(t, "x", got.RawJSON)
	})

	t.Run("empty options", func(t *testing.T) {
		t.Parallel()
		opts := fantasy.ProviderOptions{}
		require.Nil(t, GetComputerUseMetadata(opts))
	})

	t.Run("wrong type", func(t *testing.T) {
		t.Parallel()
		opts := fantasy.ProviderOptions{Name: &ResponsesProviderMetadata{}}
		require.Nil(t, GetComputerUseMetadata(opts))
	})
}

func TestComputerCallInput_BatchedActions(t *testing.T) {
	t.Parallel()

	// Build via JSON unmarshal so RawJSON() fields are populated.
	var call responses.ResponseComputerToolCall
	err := json.Unmarshal([]byte(`{
		"type": "computer_call",
		"call_id": "call_1",
		"actions": [
			{"type": "click", "x": 100, "y": 200, "button": "left"},
			{"type": "type", "text": "hello"}
		],
		"status": "completed",
		"id": "item_1"
	}`), &call)
	require.NoError(t, err)

	result, err := computerCallInput(call)
	require.NoError(t, err)

	var parsed map[string]json.RawMessage
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	// call_id
	var callID string
	err = json.Unmarshal(parsed["call_id"], &callID)
	require.NoError(t, err)
	require.Equal(t, "call_1", callID)

	// actions
	require.Contains(t, parsed, "actions")
	require.NotContains(t, parsed, "action")

	var actions []json.RawMessage
	err = json.Unmarshal(parsed["actions"], &actions)
	require.NoError(t, err)
	require.Len(t, actions, 2)
}

func TestComputerCallInput_NoActions(t *testing.T) {
	t.Parallel()

	var call responses.ResponseComputerToolCall
	err := json.Unmarshal([]byte(`{
		"type": "computer_call",
		"call_id": "call_4",
		"status": "completed",
		"id": "item_4"
	}`), &call)
	require.NoError(t, err)

	_, err = computerCallInput(call)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no actions")
}

func TestParseComputerUseInput(t *testing.T) {
	t.Parallel()

	input := `{"call_id":"c1","actions":[{"type":"click","x":10,"y":20,"button":"left"}]}`
	parsed, err := ParseComputerUseInput(input)
	require.NoError(t, err)
	require.Equal(t, "c1", parsed.CallID)
	require.Len(t, parsed.Actions, 1)
	require.Equal(t, "click", parsed.Actions[0].Type)
	require.Equal(t, int64(10), parsed.Actions[0].X)
}

func TestParseComputerUseInput_Invalid(t *testing.T) {
	t.Parallel()

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		_, err := ParseComputerUseInput("")
		require.Error(t, err)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		_, err := ParseComputerUseInput("{bad json")
		require.Error(t, err)
	})
}

func TestComputerUseToolsInToolArray(t *testing.T) {
	t.Parallel()

	stub := func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.NewTextResponse("stub"), nil
	}

	tools := []fantasy.Tool{
		fantasy.FunctionTool{
			Name:        "search",
			Description: "search something",
			InputSchema: map[string]any{"type": "object"},
		},
		NewComputerUseTool(stub),
	}

	openaiTools, _, warnings := toResponsesTools(tools, nil, nil)
	require.Len(t, openaiTools, 2)
	require.NotNil(t, openaiTools[0].OfFunction)
	require.Equal(t, "search", openaiTools[0].OfFunction.Name)
	require.NotNil(t, openaiTools[1].OfComputer)

	// No unsupported-tool warnings.
	for _, w := range warnings {
		require.NotEqual(t, fantasy.CallWarningTypeUnsupportedTool, w.Type)
	}
}

func TestComputerUseToolChoice(t *testing.T) {
	t.Parallel()

	stub := func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.NewTextResponse("stub"), nil
	}

	choice := fantasy.ToolChoice("computer")
	tools := []fantasy.Tool{NewComputerUseTool(stub)}
	_, toolChoice, _ := toResponsesTools(tools, &choice, nil)
	require.NotNil(t, toolChoice.OfHostedTool)
	require.Equal(t, responses.ToolChoiceTypesTypeComputer, toolChoice.OfHostedTool.Type)
}

func TestComputerUseToolChoice_FunctionNamedComputer(t *testing.T) {
	t.Parallel()

	choice := fantasy.ToolChoice("computer")
	tools := []fantasy.Tool{
		fantasy.FunctionTool{
			Name:        "computer",
			Description: "a function named computer",
			InputSchema: map[string]any{"type": "object"},
		},
	}
	_, toolChoice, _ := toResponsesTools(tools, &choice, nil)
	// Should fall back to function tool choice, not hosted tool.
	require.NotNil(t, toolChoice.OfFunctionTool)
	require.Equal(t, "computer", toolChoice.OfFunctionTool.Name)
}
