package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"

	"charm.land/fantasy"
	"github.com/charmbracelet/openai-go/packages/param"
	"github.com/charmbracelet/openai-go/responses"
)

const (
	computerUseToolID     = "openai.computer"
	computerUseAPIName    = "computer"
	computerUseStoreError = "openai computer use requires store to be true in openai responses provider options"
	maxExactIntFloat64    = float64(1<<53 - 1)
)

// ComputerUseToolOptions configures the OpenAI computer use tool.
type ComputerUseToolOptions struct {
	DisplayWidthPx  int64
	DisplayHeightPx int64
	DisplayNumber   *int64
	Environment     responses.ComputerUsePreviewToolEnvironment
}

// NewComputerUseTool creates a new executable OpenAI computer use tool.
func NewComputerUseTool(
	opts ComputerUseToolOptions,
	run func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error),
) fantasy.ExecutableProviderTool {
	args := map[string]any{
		"display_width_px":  opts.DisplayWidthPx,
		"display_height_px": opts.DisplayHeightPx,
		"environment":       opts.Environment,
	}
	if opts.DisplayNumber != nil {
		args["display_number"] = *opts.DisplayNumber
	}
	return fantasy.NewExecutableProviderTool(fantasy.ProviderDefinedTool{
		ID:   computerUseToolID,
		Name: computerUseAPIName,
		Args: args,
	}, run)
}

// ComputerUseActionType identifies a single OpenAI computer action.
type ComputerUseActionType string

const (
	ComputerUseActionTypeClick       ComputerUseActionType = "click"
	ComputerUseActionTypeDoubleClick ComputerUseActionType = "double_click"
	ComputerUseActionTypeDrag        ComputerUseActionType = "drag"
	ComputerUseActionTypeKeypress    ComputerUseActionType = "keypress"
	ComputerUseActionTypeMove        ComputerUseActionType = "move"
	ComputerUseActionTypeScreenshot  ComputerUseActionType = "screenshot"
	ComputerUseActionTypeScroll      ComputerUseActionType = "scroll"
	ComputerUseActionTypeType        ComputerUseActionType = "type"
	ComputerUseActionTypeWait        ComputerUseActionType = "wait"
)

// ComputerUseAction represents a parsed OpenAI computer action.
type ComputerUseAction interface {
	Type() ComputerUseActionType
	isComputerUseAction()
}

// ComputerUseInput is the parsed representation of a computer tool call.
// Single-action payloads populate Action. Batched payloads populate Actions.
type ComputerUseInput struct {
	Action  ComputerUseAction
	Actions []ComputerUseAction
}

// ComputerUsePoint represents an x/y point.
type ComputerUsePoint struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
}

// ComputerUseClickAction represents a click action.
type ComputerUseClickAction struct {
	Button string `json:"button"`
	X      int64  `json:"x"`
	Y      int64  `json:"y"`
}

func (ComputerUseClickAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseClickAction) Type() ComputerUseActionType { return ComputerUseActionTypeClick }

// ComputerUseDoubleClickAction represents a double-click action.
type ComputerUseDoubleClickAction struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
}

func (ComputerUseDoubleClickAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseDoubleClickAction) Type() ComputerUseActionType {
	return ComputerUseActionTypeDoubleClick
}

// ComputerUseDragAction represents a drag action.
type ComputerUseDragAction struct {
	Path []ComputerUsePoint `json:"path"`
}

func (ComputerUseDragAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseDragAction) Type() ComputerUseActionType { return ComputerUseActionTypeDrag }

// ComputerUseKeypressAction represents a keypress action.
type ComputerUseKeypressAction struct {
	Keys []string `json:"keys"`
}

func (ComputerUseKeypressAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseKeypressAction) Type() ComputerUseActionType {
	return ComputerUseActionTypeKeypress
}

// ComputerUseMoveAction represents a move action.
type ComputerUseMoveAction struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
}

func (ComputerUseMoveAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseMoveAction) Type() ComputerUseActionType { return ComputerUseActionTypeMove }

// ComputerUseScreenshotAction represents a screenshot action.
type ComputerUseScreenshotAction struct{}

func (ComputerUseScreenshotAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseScreenshotAction) Type() ComputerUseActionType {
	return ComputerUseActionTypeScreenshot
}

// ComputerUseScrollAction represents a scroll action.
type ComputerUseScrollAction struct {
	X       int64 `json:"x"`
	Y       int64 `json:"y"`
	ScrollX int64 `json:"scroll_x"`
	ScrollY int64 `json:"scroll_y"`
}

func (ComputerUseScrollAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseScrollAction) Type() ComputerUseActionType { return ComputerUseActionTypeScroll }

// ComputerUseTypeAction represents a type action.
type ComputerUseTypeAction struct {
	Text string `json:"text"`
}

func (ComputerUseTypeAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseTypeAction) Type() ComputerUseActionType { return ComputerUseActionTypeType }

// ComputerUseWaitAction represents a wait action.
type ComputerUseWaitAction struct{}

func (ComputerUseWaitAction) isComputerUseAction() {}

// Type returns the action discriminator.
func (ComputerUseWaitAction) Type() ComputerUseActionType { return ComputerUseActionTypeWait }

// ParseComputerUseInput parses a raw OpenAI computer-use input payload.
// Single actions are encoded as a JSON object. Batched actions are encoded as a
// JSON array.
func ParseComputerUseInput(input []byte) (ComputerUseInput, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return ComputerUseInput{}, fmt.Errorf("computer use input is empty")
	}

	switch trimmed[0] {
	case '{':
		action, err := parseComputerUseAction(trimmed)
		if err != nil {
			return ComputerUseInput{}, err
		}
		return ComputerUseInput{Action: action}, nil
	case '[':
		var rawActions []json.RawMessage
		if err := json.Unmarshal(trimmed, &rawActions); err != nil {
			return ComputerUseInput{}, err
		}
		if len(rawActions) == 0 {
			return ComputerUseInput{}, fmt.Errorf("openai computer use input requires at least one action")
		}
		actions := make([]ComputerUseAction, 0, len(rawActions))
		for _, rawAction := range rawActions {
			action, err := parseComputerUseAction(rawAction)
			if err != nil {
				return ComputerUseInput{}, err
			}
			actions = append(actions, action)
		}
		return ComputerUseInput{Actions: actions}, nil
	default:
		return ComputerUseInput{}, fmt.Errorf("computer use input must be a JSON object or array")
	}
}

func parseComputerUseAction(input []byte) (ComputerUseAction, error) {
	var header struct {
		Type ComputerUseActionType `json:"type"`
	}
	if err := json.Unmarshal(input, &header); err != nil {
		return nil, err
	}

	switch header.Type {
	case ComputerUseActionTypeClick:
		var action ComputerUseClickAction
		return action, json.Unmarshal(input, &action)
	case ComputerUseActionTypeDoubleClick:
		var action ComputerUseDoubleClickAction
		return action, json.Unmarshal(input, &action)
	case ComputerUseActionTypeDrag:
		var action ComputerUseDragAction
		if err := json.Unmarshal(input, &action); err != nil {
			return nil, err
		}
		if len(action.Path) == 0 {
			return nil, fmt.Errorf("computer use drag action requires a non-empty path")
		}
		return action, nil
	case ComputerUseActionTypeKeypress:
		var action ComputerUseKeypressAction
		return action, json.Unmarshal(input, &action)
	case ComputerUseActionTypeMove:
		var action ComputerUseMoveAction
		return action, json.Unmarshal(input, &action)
	case ComputerUseActionTypeScreenshot:
		var action ComputerUseScreenshotAction
		return action, json.Unmarshal(input, &action)
	case ComputerUseActionTypeScroll:
		var action ComputerUseScrollAction
		return action, json.Unmarshal(input, &action)
	case ComputerUseActionTypeType:
		var action ComputerUseTypeAction
		return action, json.Unmarshal(input, &action)
	case ComputerUseActionTypeWait:
		var action ComputerUseWaitAction
		return action, json.Unmarshal(input, &action)
	default:
		return nil, fmt.Errorf("unsupported computer use action type %q", header.Type)
	}
}

// NewComputerUseScreenshotResult returns a screenshot tool result with PNG data.
// Screenshot and media outputs are the only computer-use tool results that
// round-trip through the OpenAI Responses API. Use this helper for OpenAI
// computer-use submissions.
func NewComputerUseScreenshotResult(toolCallID string, screenshotPNG []byte) fantasy.ToolResultPart {
	return fantasy.ToolResultPart{
		ToolCallID: toolCallID,
		Output: fantasy.ToolResultOutputContentMedia{
			Data:      base64.StdEncoding.EncodeToString(screenshotPNG),
			MediaType: "image/png",
		},
	}
}

// NewComputerUseScreenshotResultWithMediaType returns a screenshot tool result
// with caller-provided base64 data and media type. Like
// NewComputerUseScreenshotResult, this round-trips through OpenAI Responses
// computer-use because it produces media output.
func NewComputerUseScreenshotResultWithMediaType(
	toolCallID string,
	base64Data string,
	mediaType string,
) fantasy.ToolResultPart {
	return fantasy.ToolResultPart{
		ToolCallID: toolCallID,
		Output: fantasy.ToolResultOutputContentMedia{
			Data:      base64Data,
			MediaType: mediaType,
		},
	}
}

// NewComputerUseErrorResult returns an error tool result.
// OpenAI Responses computer-use does not accept this output format on replay.
// Use it for local display, logging, or other non-OpenAI flows that want to
// preserve the failure alongside the tool call ID.
func NewComputerUseErrorResult(toolCallID string, err error) fantasy.ToolResultPart {
	return fantasy.ToolResultPart{
		ToolCallID: toolCallID,
		Output: fantasy.ToolResultOutputContentError{
			Error: err,
		},
	}
}

// NewComputerUseTextResult returns a text tool result.
// OpenAI Responses computer-use does not accept this output format on replay.
// Use it for local display, logging, or tests that need to keep a textual
// association with the original tool call ID.
func NewComputerUseTextResult(toolCallID string, text string) fantasy.ToolResultPart {
	return fantasy.ToolResultPart{
		ToolCallID: toolCallID,
		Output: fantasy.ToolResultOutputContentText{
			Text: text,
		},
	}
}

func asProviderDefinedTool(tool fantasy.Tool) (fantasy.ProviderDefinedTool, bool) {
	if pdt, ok := tool.(fantasy.ProviderDefinedTool); ok {
		return pdt, true
	}
	if ept, ok := tool.(fantasy.ExecutableProviderTool); ok {
		return ept.Definition(), true
	}
	return fantasy.ProviderDefinedTool{}, false
}

func hasComputerUseTool(tools []fantasy.Tool) bool {
	for _, tool := range tools {
		pt, ok := asProviderDefinedTool(tool)
		if ok && pt.ID == computerUseToolID {
			return true
		}
	}
	return false
}

func anyToInt64(v any) (int64, bool) {
	switch typed := v.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		u64 := uint64(typed)
		if u64 > math.MaxInt64 {
			return 0, false
		}
		return int64(u64), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case float32:
		f := float64(typed)
		if math.Trunc(f) != f || math.IsNaN(f) || math.IsInf(f, 0) || f < -maxExactIntFloat64 || f > maxExactIntFloat64 {
			return 0, false
		}
		return int64(f), true
	case float64:
		if math.Trunc(typed) != typed || math.IsNaN(typed) || math.IsInf(typed, 0) || typed < -maxExactIntFloat64 || typed > maxExactIntFloat64 {
			return 0, false
		}
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func anyToComputerUseEnvironment(v any) (responses.ComputerUsePreviewToolEnvironment, bool) {
	switch typed := v.(type) {
	case responses.ComputerUsePreviewToolEnvironment:
		return typed, true
	case string:
		return responses.ComputerUsePreviewToolEnvironment(typed), typed != ""
	default:
		return "", false
	}
}

func toComputerUseToolParam(pt fantasy.ProviderDefinedTool) (responses.ToolUnionParam, error) {
	displayHeight, ok := anyToInt64(pt.Args["display_height_px"])
	if !ok || displayHeight <= 0 {
		return responses.ToolUnionParam{}, fmt.Errorf("computer use tool has invalid display_height_px")
	}
	displayWidth, ok := anyToInt64(pt.Args["display_width_px"])
	if !ok || displayWidth <= 0 {
		return responses.ToolUnionParam{}, fmt.Errorf("computer use tool has invalid display_width_px")
	}
	environment, ok := anyToComputerUseEnvironment(pt.Args["environment"])
	if !ok {
		return responses.ToolUnionParam{}, fmt.Errorf("computer use tool has invalid environment")
	}
	return responses.ToolParamOfComputerUsePreview(displayHeight, displayWidth, environment), nil
}

func getComputerUseCallMetadata(options fantasy.ProviderOptions) *OpenAIComputerUseCallMetadata {
	if options == nil {
		return nil
	}
	if providerOptions, ok := options[Name]; ok {
		if metadata, ok := providerOptions.(*OpenAIComputerUseCallMetadata); ok {
			return metadata
		}
	}
	return nil
}

func computerUseToolCallMetadata(call responses.ResponseComputerToolCall) *OpenAIComputerUseCallMetadata {
	metadata := &OpenAIComputerUseCallMetadata{
		CallID: call.CallID,
	}
	if len(call.PendingSafetyChecks) > 0 {
		metadata.PendingSafetyChecks = make([]OpenAIComputerUsePendingSafetyCheck, 0, len(call.PendingSafetyChecks))
		for _, check := range call.PendingSafetyChecks {
			metadata.PendingSafetyChecks = append(metadata.PendingSafetyChecks, OpenAIComputerUsePendingSafetyCheck{
				ID:      check.ID,
				Code:    check.Code,
				Message: check.Message,
			})
		}
	}
	return metadata
}

func computerUseSafetyChecksToAcknowledgedParams(checks []OpenAIComputerUsePendingSafetyCheck) []responses.ResponseInputItemComputerCallOutputAcknowledgedSafetyCheckParam {
	if len(checks) == 0 {
		return nil
	}
	paramsList := make([]responses.ResponseInputItemComputerCallOutputAcknowledgedSafetyCheckParam, 0, len(checks))
	for _, check := range checks {
		ack := responses.ResponseInputItemComputerCallOutputAcknowledgedSafetyCheckParam{
			ID: check.ID,
		}
		if check.Code != "" {
			ack.Code = param.NewOpt(check.Code)
		}
		if check.Message != "" {
			ack.Message = param.NewOpt(check.Message)
		}
		paramsList = append(paramsList, ack)
	}
	return paramsList
}

func computerUseToolResultInput(toolResult fantasy.ToolResultPart, metadata *OpenAIComputerUseCallMetadata) (responses.ResponseInputItemUnionParam, error) {
	if metadata == nil || metadata.CallID == "" {
		return responses.ResponseInputItemUnionParam{}, fmt.Errorf("openai computer tool call metadata is missing call_id")
	}
	output, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentMedia](toolResult.Output)
	if !ok {
		return responses.ResponseInputItemUnionParam{}, fmt.Errorf("openai computer tool results must use media output")
	}
	if output.MediaType == "" {
		return responses.ResponseInputItemUnionParam{}, fmt.Errorf("openai computer tool results must include a media type")
	}
	item := responses.ResponseInputItemParamOfComputerCallOutput(metadata.CallID, responses.ResponseComputerToolCallOutputScreenshotParam{
		ImageURL: param.NewOpt(fmt.Sprintf("data:%s;base64,%s", output.MediaType, output.Data)),
	})
	item.OfComputerCallOutput.AcknowledgedSafetyChecks = computerUseSafetyChecksToAcknowledgedParams(metadata.PendingSafetyChecks)
	return item, nil
}

func computerUseToolCallContent(call responses.ResponseComputerToolCall) (fantasy.ToolCallContent, error) {
	input, err := computerUseToolCallInput(call)
	if err != nil {
		return fantasy.ToolCallContent{}, err
	}
	return fantasy.ToolCallContent{
		ToolCallID:       call.ID,
		ToolName:         computerUseAPIName,
		Input:            input,
		ProviderExecuted: false,
		ProviderMetadata: fantasy.ProviderMetadata{
			Name: computerUseToolCallMetadata(call),
		},
	}, nil
}

func computerUseToolCallInput(call responses.ResponseComputerToolCall) (string, error) {
	if len(call.Actions) > 0 {
		payload, err := computerUseActionsJSON(call.Actions)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	}

	payload, err := computerUseResponseActionJSON(call.Action)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func computerUseActionsJSON(actions responses.ComputerActionList) ([]byte, error) {
	if len(actions) == 0 {
		return nil, fmt.Errorf("computer use tool call is missing actions")
	}
	rawActions := make([]json.RawMessage, 0, len(actions))
	for _, action := range actions {
		payload, err := computerUseActionJSON(action)
		if err != nil {
			return nil, err
		}
		rawActions = append(rawActions, payload)
	}
	return json.Marshal(rawActions)
}

func computerUseActionJSON(action responses.ComputerActionUnion) (json.RawMessage, error) {
	if raw := action.RawJSON(); raw != "" {
		return json.RawMessage(raw), nil
	}
	variant := action.AsAny()
	if variant == nil {
		return nil, fmt.Errorf("computer use tool call is missing action payload")
	}
	payload, err := json.Marshal(variant)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func computerUseResponseActionJSON(action responses.ResponseComputerToolCallActionUnion) (json.RawMessage, error) {
	if raw := action.RawJSON(); raw != "" {
		return json.RawMessage(raw), nil
	}
	variant := action.AsAny()
	if variant == nil {
		return nil, fmt.Errorf("computer use tool call is missing action payload")
	}
	payload, err := json.Marshal(variant)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}
