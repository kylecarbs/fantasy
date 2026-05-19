package providertests

import (
	"cmp"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"testing"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"charm.land/x/vcr"
	"github.com/charmbracelet/openai-go/responses"
	"github.com/stretchr/testify/require"
)

// openAIComputerUseBuilder creates a builder for the OpenAI Responses API
// with computer use support.
func openAIComputerUseBuilder(model string) builderFunc {
	return func(t *testing.T, r *vcr.Recorder) (fantasy.LanguageModel, error) {
		opts := []openai.Option{
			openai.WithAPIKey(cmp.Or(os.Getenv("FANTASY_OPENAI_API_KEY"), os.Getenv("OPENAI_API_KEY"), "(missing)")),
			openai.WithHTTPClient(&http.Client{Transport: r}),
			openai.WithUseResponsesAPI(),
		}
		provider, err := openai.New(opts...)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

// cannedScreenshotBase64 is a minimal valid 1x1 white PNG encoded as
// base64, used in all computer use VCR tests.
const cannedScreenshotBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="

// cannedScreenshot is the raw bytes of the 1x1 white PNG.
var cannedScreenshot []byte

func init() {
	var err error
	cannedScreenshot, err = base64.StdEncoding.DecodeString(cannedScreenshotBase64)
	if err != nil {
		panic(fmt.Sprintf("decode canned screenshot: %v", err))
	}
}

// testComputerUseTool creates a computer use tool that returns a canned
// screenshot. The returned bool pointer tracks whether the callback was
// invoked.
func testComputerUseTool(t *testing.T) (fantasy.ExecutableProviderTool, *bool) {
	t.Helper()
	called := new(bool)
	tool := openai.NewComputerUseTool(
		func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			*called = true
			parsed, err := openai.ParseComputerUseInput(call.Input)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("parse input: %w", err)
			}
			require.NotEmpty(t, parsed.CallID)

			return fantasy.NewImageResponse(cannedScreenshot, "image/png"), nil
		},
	)
	return tool, called
}

func int64Ptr(v int64) *int64 {
	return &v
}

// TestOpenAIComputerUse tests OpenAI computer use tool support via the
// agent using the Responses API. Cassettes are stored under
// testdata/TestOpenAIComputerUse/.
func TestOpenAIComputerUse(t *testing.T) {
	model := "gpt-5.4"
	providerOpts := fantasy.ProviderOptions{
		openai.Name: &openai.ResponsesProviderOptions{
			ReasoningEffort: openai.ReasoningEffortOption(openai.ReasoningEffortMedium),
		},
	}

	t.Run("generate", func(t *testing.T) {
		r := vcr.NewRecorder(t)

		lm, err := openAIComputerUseBuilder(model)(t, r)
		require.NoError(t, err)

		cuTool, called := testComputerUseTool(t)
		agent := fantasy.NewAgent(
			lm,
			fantasy.WithSystemPrompt("You are a helpful assistant that can control a computer. Take a screenshot when asked."),
			fantasy.WithProviderDefinedTools(cuTool),
		)

		result, err := agent.Generate(t.Context(), fantasy.AgentCall{
			Prompt:          "Take a screenshot of the desktop",
			MaxOutputTokens: int64Ptr(4000),
			ProviderOptions: providerOpts,
		})
		require.NoError(t, err)
		require.True(t, *called, "expected computer use callback to be invoked")

		// The agent should have looped at least twice: tool call then final text.
		require.GreaterOrEqual(t, len(result.Steps), 2)

		// Step 0 should contain a computer tool call.
		var computerToolCalls []fantasy.ToolCallContent
		for _, c := range result.Steps[0].Content {
			if tc, ok := c.(fantasy.ToolCallContent); ok && tc.ToolName == "computer" {
				computerToolCalls = append(computerToolCalls, tc)
			}
		}
		require.NotEmpty(t, computerToolCalls, "expected computer tool call in step 0")
		require.NotEmpty(t, computerToolCalls[0].ToolCallID)

		// Final response should have text.
		got := result.Response.Content.Text()
		require.NotEmpty(t, got, "expected a text response")
	})

	t.Run("stream", func(t *testing.T) {
		r := vcr.NewRecorder(t)

		lm, err := openAIComputerUseBuilder(model)(t, r)
		require.NoError(t, err)

		cuTool, called := testComputerUseTool(t)
		agent := fantasy.NewAgent(
			lm,
			fantasy.WithSystemPrompt("You are a helpful assistant that can control a computer. Take a screenshot when asked."),
			fantasy.WithProviderDefinedTools(cuTool),
		)

		result, err := agent.Stream(t.Context(), fantasy.AgentStreamCall{
			Prompt:          "Take a screenshot of the desktop",
			MaxOutputTokens: int64Ptr(4000),
			ProviderOptions: providerOpts,
		})
		require.NoError(t, err)
		require.True(t, *called, "expected computer use callback to be invoked")

		// The agent should have looped at least twice.
		require.GreaterOrEqual(t, len(result.Steps), 2)

		// Step 0 should contain a computer tool call.
		var computerToolCalls []fantasy.ToolCallContent
		for _, c := range result.Steps[0].Content {
			if tc, ok := c.(fantasy.ToolCallContent); ok && tc.ToolName == "computer" {
				computerToolCalls = append(computerToolCalls, tc)
			}
		}
		require.NotEmpty(t, computerToolCalls, "expected computer tool call in step 0")

		// Final response should have text.
		got := result.Response.Content.Text()
		require.NotEmpty(t, got, "expected a text response")
	})
}

// allComputerActionTypes lists every action type the OpenAI computer
// tool supports. The test below instructs the model to emit each one.
var allComputerActionTypes = []string{
	"click",
	"double_click",
	"drag",
	"keypress",
	"move",
	"screenshot",
	"scroll",
	"type",
	"wait",
}

// testComputerUseToolCollectingActions creates a computer use tool that
// returns a canned screenshot and collects every action it receives.
func testComputerUseToolCollectingActions(t *testing.T) (fantasy.ExecutableProviderTool, *[]responses.ComputerActionUnion) {
	t.Helper()
	var collected []responses.ComputerActionUnion
	tool := openai.NewComputerUseTool(
		func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			parsed, err := openai.ParseComputerUseInput(call.Input)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("parse input: %w", err)
			}
			require.NotEmpty(t, parsed.CallID)
			collected = append(collected, parsed.Actions...)
			return fantasy.NewImageResponse(cannedScreenshot, "image/png"), nil
		},
	)
	return tool, &collected
}

// TestOpenAIComputerUse_AllActions verifies that the agent can exercise
// all 9 computer action types (click, double_click, drag, keypress,
// move, screenshot, scroll, type, wait) in a single conversation,
// and that each action carries the parameters specified in the prompt.
func TestOpenAIComputerUse_AllActions(t *testing.T) {
	model := "gpt-5.4"
	providerOpts := fantasy.ProviderOptions{
		openai.Name: &openai.ResponsesProviderOptions{
			ReasoningEffort: openai.ReasoningEffortOption(openai.ReasoningEffortMedium),
		},
	}

	r := vcr.NewRecorder(t)

	lm, err := openAIComputerUseBuilder(model)(t, r)
	require.NoError(t, err)

	cuTool, collected := testComputerUseToolCollectingActions(t)
	agent := fantasy.NewAgent(
		lm,
		fantasy.WithSystemPrompt(
			"You control a computer. The user will ask you to perform specific "+
				"actions. Execute exactly the actions requested. Each computer_call "+
				"response may contain multiple actions in the actions array. Batch "+
				"as many as possible into each call. When you have executed all "+
				"requested actions, reply with the word DONE.",
		),
		fantasy.WithProviderDefinedTools(cuTool),
	)

	// Prompt explicitly lists all 9 action types with concrete parameters
	// so the model has no room for ambiguity.
	prompt := "Execute every one of these computer actions. " +
		"Batch them into as few calls as possible.\n\n" +
		"1. screenshot\n" +
		"2. click at x=100, y=200 with left button\n" +
		"3. double_click at x=150, y=250\n" +
		"4. type the text \"hello world\"\n" +
		"5. keypress the keys [\"ctrl\", \"s\"]\n" +
		"6. scroll at x=300, y=400 with scroll_x=0, scroll_y=-3\n" +
		"7. move to x=500, y=600\n" +
		"8. drag along the path [{x:10,y:20},{x:30,y:40}]\n" +
		"9. wait\n\n" +
		"After executing all actions, reply DONE."

	result, err := agent.Generate(t.Context(), fantasy.AgentCall{
		Prompt:          prompt,
		MaxOutputTokens: int64Ptr(16000),
		ProviderOptions: providerOpts,
	})
	require.NoError(t, err)

	// Verify the agent completed with at least a tool step and a text step.
	require.GreaterOrEqual(t, len(result.Steps), 2,
		"expected at least one tool-call step and one text step")

	// Index collected actions by type. If the model emits the same type
	// more than once, we keep only the first occurrence.
	byType := make(map[string]responses.ComputerActionUnion, len(allComputerActionTypes))
	for _, a := range *collected {
		if _, exists := byType[a.Type]; !exists {
			byType[a.Type] = a
		}
	}

	// Assert every action type was seen.
	var missing []string
	for _, actionType := range allComputerActionTypes {
		if _, ok := byType[actionType]; !ok {
			missing = append(missing, actionType)
		}
	}
	require.Empty(t, missing,
		"expected all 9 action types to be exercised, missing: %v", missing)

	// Verify parameters for each action type match the prompt.
	// Each action is cast to its narrow SDK type via As*().

	// click at x=100, y=200 with left button
	click := byType["click"].AsClick()
	require.Equal(t, int64(100), click.X, "click x")
	require.Equal(t, int64(200), click.Y, "click y")
	require.Equal(t, "left", click.Button, "click button")

	// double_click at x=150, y=250
	dblClick := byType["double_click"].AsDoubleClick()
	require.Equal(t, int64(150), dblClick.X, "double_click x")
	require.Equal(t, int64(250), dblClick.Y, "double_click y")

	// type the text "hello world"
	typeAction := byType["type"].AsType()
	require.Equal(t, "hello world", typeAction.Text, "type text")

	// keypress the keys ["ctrl", "s"]
	keypress := byType["keypress"].AsKeypress()
	require.Equal(t, []string{"ctrl", "s"}, keypress.Keys, "keypress keys")

	// scroll at x=300, y=400 with scroll_x=0, scroll_y=-3
	scroll := byType["scroll"].AsScroll()
	require.Equal(t, int64(300), scroll.X, "scroll x")
	require.Equal(t, int64(400), scroll.Y, "scroll y")
	require.Equal(t, int64(0), scroll.ScrollX, "scroll scroll_x")
	require.Equal(t, int64(-3), scroll.ScrollY, "scroll scroll_y")

	// move to x=500, y=600
	move := byType["move"].AsMove()
	require.Equal(t, int64(500), move.X, "move x")
	require.Equal(t, int64(600), move.Y, "move y")

	// drag along the path [{x:10,y:20},{x:30,y:40}]
	drag := byType["drag"].AsDrag()
	require.Len(t, drag.Path, 2, "drag path length")
	require.Equal(t, int64(10), drag.Path[0].X, "drag path[0].x")
	require.Equal(t, int64(20), drag.Path[0].Y, "drag path[0].y")
	require.Equal(t, int64(30), drag.Path[1].X, "drag path[1].x")
	require.Equal(t, int64(40), drag.Path[1].Y, "drag path[1].y")

	// screenshot and wait have no parameters beyond their type.
	// Cast to narrow types to confirm deserialization works.
	_ = byType["screenshot"].AsScreenshot()
	_ = byType["wait"].AsWait()
}
