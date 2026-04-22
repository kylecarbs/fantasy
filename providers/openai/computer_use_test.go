package openai

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/openai-go/responses"
	"github.com/stretchr/testify/require"
)

func TestNewComputerUseTool(t *testing.T) {
	t.Parallel()

	displayNumber := int64(2)
	tool := NewComputerUseTool(ComputerUseToolOptions{
		DisplayWidthPx:  1920,
		DisplayHeightPx: 1080,
		DisplayNumber:   &displayNumber,
		Environment:     responses.ComputerUsePreviewToolEnvironmentUbuntu,
	}, func(context.Context, fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.ToolResponse{}, nil
	})

	definition := tool.Definition()
	require.Equal(t, computerUseToolID, definition.ID)
	require.Equal(t, computerUseAPIName, definition.Name)
	require.Equal(t, int64(1920), definition.Args["display_width_px"])
	require.Equal(t, int64(1080), definition.Args["display_height_px"])
	require.Equal(t, responses.ComputerUsePreviewToolEnvironmentUbuntu, definition.Args["environment"])
	require.Equal(t, int64(2), definition.Args["display_number"])
}

func TestParseComputerUseInput(t *testing.T) {
	t.Parallel()

	t.Run("click", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"click","button":"left","x":100,"y":200}`))
		require.NoError(t, err)
		require.Nil(t, input.Actions)

		action, ok := input.Action.(ComputerUseClickAction)
		require.True(t, ok)
		require.Equal(t, ComputerUseActionTypeClick, action.Type())
		require.Equal(t, "left", action.Button)
		require.Equal(t, int64(100), action.X)
		require.Equal(t, int64(200), action.Y)
	})

	t.Run("double click", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"double_click","x":10,"y":20}`))
		require.NoError(t, err)

		action, ok := input.Action.(ComputerUseDoubleClickAction)
		require.True(t, ok)
		require.Equal(t, ComputerUseActionTypeDoubleClick, action.Type())
		require.Equal(t, int64(10), action.X)
		require.Equal(t, int64(20), action.Y)
	})

	t.Run("drag", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"drag","path":[{"x":1,"y":2},{"x":3,"y":4},{"x":5,"y":6}]}`))
		require.NoError(t, err)

		action, ok := input.Action.(ComputerUseDragAction)
		require.True(t, ok)
		require.Equal(t, ComputerUseActionTypeDrag, action.Type())
		require.Equal(t, []ComputerUsePoint{{X: 1, Y: 2}, {X: 3, Y: 4}, {X: 5, Y: 6}}, action.Path)
	})

	t.Run("keypress", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"keypress","keys":["CTRL","L"]}`))
		require.NoError(t, err)

		action, ok := input.Action.(ComputerUseKeypressAction)
		require.True(t, ok)
		require.Equal(t, ComputerUseActionTypeKeypress, action.Type())
		require.Equal(t, []string{"CTRL", "L"}, action.Keys)
	})

	t.Run("move", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"move","x":320,"y":240}`))
		require.NoError(t, err)

		action, ok := input.Action.(ComputerUseMoveAction)
		require.True(t, ok)
		require.Equal(t, ComputerUseActionTypeMove, action.Type())
		require.Equal(t, int64(320), action.X)
		require.Equal(t, int64(240), action.Y)
	})

	t.Run("screenshot", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"screenshot"}`))
		require.NoError(t, err)

		_, ok := input.Action.(ComputerUseScreenshotAction)
		require.True(t, ok)
	})

	t.Run("scroll", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"scroll","x":10,"y":20,"scroll_x":0,"scroll_y":600}`))
		require.NoError(t, err)

		action, ok := input.Action.(ComputerUseScrollAction)
		require.True(t, ok)
		require.Equal(t, ComputerUseActionTypeScroll, action.Type())
		require.Equal(t, int64(10), action.X)
		require.Equal(t, int64(20), action.Y)
		require.Equal(t, int64(0), action.ScrollX)
		require.Equal(t, int64(600), action.ScrollY)
	})

	t.Run("type", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"type","text":"hello"}`))
		require.NoError(t, err)

		action, ok := input.Action.(ComputerUseTypeAction)
		require.True(t, ok)
		require.Equal(t, ComputerUseActionTypeType, action.Type())
		require.Equal(t, "hello", action.Text)
	})

	t.Run("wait", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`{"type":"wait"}`))
		require.NoError(t, err)

		_, ok := input.Action.(ComputerUseWaitAction)
		require.True(t, ok)
	})

	t.Run("batched actions", func(t *testing.T) {
		t.Parallel()

		input, err := ParseComputerUseInput([]byte(`[{"type":"move","x":10,"y":20},{"type":"click","button":"left","x":10,"y":20}]`))
		require.NoError(t, err)
		require.Nil(t, input.Action)
		require.Len(t, input.Actions, 2)

		moveAction, ok := input.Actions[0].(ComputerUseMoveAction)
		require.True(t, ok)
		require.Equal(t, int64(10), moveAction.X)
		require.Equal(t, int64(20), moveAction.Y)

		clickAction, ok := input.Actions[1].(ComputerUseClickAction)
		require.True(t, ok)
		require.Equal(t, "left", clickAction.Button)
	})

	t.Run("empty batched actions error", func(t *testing.T) {
		t.Parallel()

		_, err := ParseComputerUseInput([]byte(`[]`))
		require.EqualError(t, err, "openai computer use input requires at least one action")
	})

	t.Run("unknown action errors", func(t *testing.T) {
		t.Parallel()

		_, err := ParseComputerUseInput([]byte(`{"type":"future_action"}`))
		require.Error(t, err)
	})
}

func TestNewComputerUseScreenshotResult(t *testing.T) {
	t.Parallel()

	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	result := NewComputerUseScreenshotResult("tool_123", pngData)

	require.Equal(t, "tool_123", result.ToolCallID)
	media, ok := result.Output.(fantasy.ToolResultOutputContentMedia)
	require.True(t, ok)
	require.Equal(t, "image/png", media.MediaType)
	require.Equal(t, base64.StdEncoding.EncodeToString(pngData), media.Data)
}

func TestNewComputerUseScreenshotResultWithMediaType(t *testing.T) {
	t.Parallel()

	result := NewComputerUseScreenshotResultWithMediaType("tool_123", "ZmFrZQ==", "image/jpeg")

	require.Equal(t, "tool_123", result.ToolCallID)
	media, ok := result.Output.(fantasy.ToolResultOutputContentMedia)
	require.True(t, ok)
	require.Equal(t, "image/jpeg", media.MediaType)
	require.Equal(t, "ZmFrZQ==", media.Data)
}

func TestNewComputerUseErrorResult(t *testing.T) {
	t.Parallel()

	result := NewComputerUseErrorResult("tool_123", errors.New("boom"))

	require.Equal(t, "tool_123", result.ToolCallID)
	errOutput, ok := result.Output.(fantasy.ToolResultOutputContentError)
	require.True(t, ok)
	require.EqualError(t, errOutput.Error, "boom")
}

func TestNewComputerUseTextResult(t *testing.T) {
	t.Parallel()

	result := NewComputerUseTextResult("tool_123", "done")

	require.Equal(t, "tool_123", result.ToolCallID)
	textOutput, ok := result.Output.(fantasy.ToolResultOutputContentText)
	require.True(t, ok)
	require.Equal(t, "done", textOutput.Text)
}
