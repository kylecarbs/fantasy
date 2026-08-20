package google

import (
	"encoding/json"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestToGoogleTools_ConvertsFunctionInputSchema(t *testing.T) {
	t.Parallel()

	tools := []fantasy.Tool{
		fantasy.FunctionTool{
			Name: "schema_test",
			InputSchema: map[string]any{
				"type": "object",
				"$defs": map[string]any{
					"node": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"kind": map[string]any{
								"type": "string",
								"enum": []any{"leaf", "branch"},
							},
							"child": map[string]any{"$ref": "#/$defs/node"},
						},
						"required": []any{"kind"},
					},
				},
				"properties": map[string]any{
					"node": map[string]any{"$ref": "#/$defs/node"},
					"choice": map[string]any{
						"anyOf": []any{
							map[string]any{"type": "string", "enum": []any{"a", "b"}},
							map[string]any{
								"type": "object",
								"properties": map[string]any{
									"count": map[string]any{"type": "integer"},
								},
								"required": []any{"count"},
							},
						},
					},
				},
				"required": []any{"node", "choice"},
			},
		},
	}

	declarations, _, warnings := toGoogleTools(tools, nil)
	require.Empty(t, warnings)
	require.Len(t, declarations, 1)

	wire, err := json.Marshal(&genai.GenerateContentConfig{
		Tools: []*genai.Tool{{FunctionDeclarations: declarations}},
	})
	require.NoError(t, err)

	var payload struct {
		Tools []struct {
			FunctionDeclarations []struct {
				Parameters map[string]any `json:"parameters"`
			} `json:"functionDeclarations"`
		} `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(wire, &payload))
	parameters := payload.Tools[0].FunctionDeclarations[0].Parameters
	require.Equal(t, []any{"node", "choice"}, parameters["required"])

	properties := parameters["properties"].(map[string]any)
	node := properties["node"].(map[string]any)
	require.Equal(t, "OBJECT", node["type"])
	require.Equal(t, []any{"kind"}, node["required"])
	nodeProperties := node["properties"].(map[string]any)
	require.Equal(t, []any{"leaf", "branch"}, nodeProperties["kind"].(map[string]any)["enum"])
	require.NotContains(t, nodeProperties, "child")

	choice := properties["choice"].(map[string]any)
	anyOf := choice["anyOf"].([]any)
	require.Equal(t, []any{"a", "b"}, anyOf[0].(map[string]any)["enum"])
	require.Equal(t, []any{"count"}, anyOf[1].(map[string]any)["required"])
}
