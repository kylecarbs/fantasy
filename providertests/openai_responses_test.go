package providertests

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"charm.land/x/vcr"
	openairesponses "github.com/charmbracelet/openai-go/responses"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesCommon(t *testing.T) {
	var pairs []builderPair
	for _, m := range openaiTestModels {
		pairs = append(pairs, builderPair{m.name, openAIReasoningBuilder(m.model), nil, nil})
	}
	testCommon(t, pairs)
}

func openAIReasoningBuilder(model string) builderFunc {
	return func(t *testing.T, r *vcr.Recorder) (fantasy.LanguageModel, error) {
		provider, err := openai.New(
			openai.WithAPIKey(os.Getenv("FANTASY_OPENAI_API_KEY")),
			openai.WithHTTPClient(&http.Client{Transport: r}),
			openai.WithUseResponsesAPI(),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

func TestOpenAIResponsesWithSummaryThinking(t *testing.T) {
	opts := fantasy.ProviderOptions{
		openai.Name: &openai.ResponsesProviderOptions{
			Include: []openai.IncludeType{
				openai.IncludeReasoningEncryptedContent,
			},
			ReasoningEffort:  openai.ReasoningEffortOption(openai.ReasoningEffortHigh),
			ReasoningSummary: fantasy.Opt("auto"),
		},
	}
	var pairs []builderPair
	for _, m := range openaiTestModels {
		if !m.reasoning {
			continue
		}
		pairs = append(pairs, builderPair{m.name, openAIReasoningBuilder(m.model), opts, nil})
	}
	testThinking(t, pairs, testOpenAIResponsesThinkingWithSummaryThinking)
}

func TestOpenAIResponsesObjectGeneration(t *testing.T) {
	var pairs []builderPair
	for _, m := range openaiTestModels {
		pairs = append(pairs, builderPair{m.name, openAIReasoningBuilder(m.model), nil, nil})
	}
	testObjectGeneration(t, pairs)
}

func testOpenAIResponsesThinkingWithSummaryThinking(t *testing.T, result *fantasy.AgentResult) {
	reasoningContentCount := 0
	encryptedData := 0
	// Test if we got the signature
	for _, step := range result.Steps {
		for _, msg := range step.Messages {
			for _, content := range msg.Content {
				if content.GetType() == fantasy.ContentTypeReasoning {
					reasoningContentCount += 1
					reasoningContent, ok := fantasy.AsContentType[fantasy.ReasoningPart](content)
					if !ok {
						continue
					}
					if len(reasoningContent.ProviderOptions) == 0 {
						continue
					}

					openaiReasoningMetadata, ok := reasoningContent.ProviderOptions[openai.Name]
					if !ok {
						continue
					}
					if typed, ok := openaiReasoningMetadata.(*openai.ResponsesReasoningMetadata); ok {
						require.NotEmpty(t, typed.EncryptedContent)
						encryptedData += 1
					}
				}
			}
		}
	}
	require.Greater(t, reasoningContentCount, 0)
	require.Greater(t, encryptedData, 0)
	require.Equal(t, reasoningContentCount, encryptedData)
}

func TestOpenAIResponsesComputerUse(t *testing.T) {
	modelName := "openai-computer-use-preview"
	for _, cassettePath := range computerUseCassettePaths(t, modelName) {
		if _, err := os.Stat(cassettePath); err != nil {
			t.Skip("requires vcr cassette")
		}
	}

	modelID := string(openairesponses.ResponsesModelComputerUsePreview)
	providerOptions := fantasy.ProviderOptions{
		openai.Name: &openai.ResponsesProviderOptions{
			Store: fantasy.Opt(true),
		},
	}

	t.Run(modelName, func(t *testing.T) {
		t.Run("computer use", func(t *testing.T) {
			r := vcr.NewRecorder(t)

			model, err := openAIReasoningBuilder(modelID)(t, r)
			require.NoError(t, err)

			cuTool := jsonRoundTripTool(t, openai.NewComputerUseTool(openai.ComputerUseToolOptions{
				DisplayWidthPx:  1920,
				DisplayHeightPx: 1080,
				Environment:     openairesponses.ComputerUsePreviewToolEnvironmentBrowser,
			}, noopComputerRun))

			resp, err := model.Generate(t.Context(), fantasy.Call{
				Prompt: fantasy.Prompt{
					{Role: fantasy.MessageRoleSystem, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "You are a helpful assistant"}}},
					{Role: fantasy.MessageRoleUser, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Take a screenshot of the desktop"}}},
				},
				ProviderOptions: providerOptions,
				Tools:           []fantasy.Tool{cuTool},
			})
			require.NoError(t, err)
			require.Equal(t, fantasy.FinishReasonToolCalls, resp.FinishReason)

			toolCalls := resp.Content.ToolCalls()
			require.Len(t, toolCalls, 1)
			require.Equal(t, "computer", toolCalls[0].ToolName)
			require.Contains(t, toolCalls[0].Input, "screenshot")

			resp2, err := model.Generate(t.Context(), fantasy.Call{
				Prompt: fantasy.Prompt{
					{Role: fantasy.MessageRoleSystem, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "You are a helpful assistant"}}},
					{Role: fantasy.MessageRoleUser, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Take a screenshot of the desktop"}}},
					{
						Role: fantasy.MessageRoleAssistant,
						Content: []fantasy.MessagePart{
							fantasy.ToolCallPart{
								ToolCallID:       toolCalls[0].ToolCallID,
								ToolName:         toolCalls[0].ToolName,
								Input:            toolCalls[0].Input,
								ProviderOptions:  fantasy.ProviderOptions(toolCalls[0].ProviderMetadata),
								ProviderExecuted: toolCalls[0].ProviderExecuted,
							},
						},
					},
					{
						Role: fantasy.MessageRoleTool,
						Content: []fantasy.MessagePart{
							fantasy.ToolResultPart{
								ToolCallID: toolCalls[0].ToolCallID,
								Output: fantasy.ToolResultOutputContentMedia{
									Data:      screenshotBase64,
									MediaType: "image/png",
								},
							},
						},
					},
				},
				ProviderOptions: providerOptions,
				Tools:           []fantasy.Tool{cuTool},
			})
			require.NoError(t, err)
			require.NotEmpty(t, resp2.Content.Text())
			require.Contains(t, resp2.Content.Text(), "desktop")
		})

		t.Run("computer use streaming", func(t *testing.T) {
			r := vcr.NewRecorder(t)

			model, err := openAIReasoningBuilder(modelID)(t, r)
			require.NoError(t, err)

			cuTool := jsonRoundTripTool(t, openai.NewComputerUseTool(openai.ComputerUseToolOptions{
				DisplayWidthPx:  1920,
				DisplayHeightPx: 1080,
				Environment:     openairesponses.ComputerUsePreviewToolEnvironmentBrowser,
			}, noopComputerRun))

			stream, err := model.Stream(t.Context(), fantasy.Call{
				Prompt: fantasy.Prompt{
					{Role: fantasy.MessageRoleSystem, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "You are a helpful assistant"}}},
					{Role: fantasy.MessageRoleUser, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Take a screenshot of the desktop"}}},
				},
				ProviderOptions: providerOptions,
				Tools:           []fantasy.Tool{cuTool},
			})
			require.NoError(t, err)

			var toolCallID, toolCallName, toolCallInput string
			var toolCallMetadata fantasy.ProviderMetadata
			var toolCallProviderExecuted bool
			var finishReason fantasy.FinishReason
			stream(func(part fantasy.StreamPart) bool {
				switch part.Type {
				case fantasy.StreamPartTypeToolCall:
					toolCallID = part.ID
					toolCallName = part.ToolCallName
					toolCallInput = part.ToolCallInput
					toolCallMetadata = part.ProviderMetadata
					toolCallProviderExecuted = part.ProviderExecuted
				case fantasy.StreamPartTypeFinish:
					finishReason = part.FinishReason
				}
				return true
			})

			require.Equal(t, fantasy.FinishReasonToolCalls, finishReason)
			require.Equal(t, "computer", toolCallName)
			require.Contains(t, toolCallInput, "screenshot")
			require.NotEmpty(t, toolCallMetadata)

			stream2, err := model.Stream(t.Context(), fantasy.Call{
				Prompt: fantasy.Prompt{
					{Role: fantasy.MessageRoleSystem, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "You are a helpful assistant"}}},
					{Role: fantasy.MessageRoleUser, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Take a screenshot of the desktop"}}},
					{
						Role: fantasy.MessageRoleAssistant,
						Content: []fantasy.MessagePart{
							fantasy.ToolCallPart{
								ToolCallID:       toolCallID,
								ToolName:         toolCallName,
								Input:            toolCallInput,
								ProviderOptions:  fantasy.ProviderOptions(toolCallMetadata),
								ProviderExecuted: toolCallProviderExecuted,
							},
						},
					},
					{
						Role: fantasy.MessageRoleTool,
						Content: []fantasy.MessagePart{
							fantasy.ToolResultPart{
								ToolCallID: toolCallID,
								Output: fantasy.ToolResultOutputContentMedia{
									Data:      screenshotBase64,
									MediaType: "image/png",
								},
							},
						},
					},
				},
				ProviderOptions: providerOptions,
				Tools:           []fantasy.Tool{cuTool},
			})
			require.NoError(t, err)

			var text string
			stream2(func(part fantasy.StreamPart) bool {
				if part.Type == fantasy.StreamPartTypeTextDelta {
					text += part.Delta
				}
				return true
			})
			require.NotEmpty(t, text)
			require.Contains(t, text, "desktop")
		})
	})
}

func computerUseCassettePaths(t *testing.T, modelName string) []string {
	base := filepath.Join("testdata", t.Name(), modelName)
	return []string{
		filepath.Join(base, "computer_use.yaml"),
		filepath.Join(base, "computer_use_streaming.yaml"),
	}
}
