package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/fantasy"
	"github.com/charmbracelet/openai-go/responses"
)

const computerUseToolID = "openai.computer_use"

// Type identifier for computer use metadata, registered in
// responses_options.go init().
const TypeComputerUseMetadata = Name + ".responses.computer_use_metadata"

// Type identifier for computer call output options, registered in
// responses_options.go init().
const TypeComputerCallOutputOptions = Name + ".responses.computer_call_output_options"

// ComputerUseMetadata stores the raw wire-format JSON of a computer_call
// output item for faithful round-tripping via param.Override.
type ComputerUseMetadata struct {
	RawJSON string `json:"raw_json"`
}

var _ fantasy.ProviderOptionsData = (*ComputerUseMetadata)(nil)

// Options implements the ProviderOptionsData interface.
func (*ComputerUseMetadata) Options() {}

// MarshalJSON implements custom JSON marshaling with type info.
func (m ComputerUseMetadata) MarshalJSON() ([]byte, error) {
	type plain ComputerUseMetadata
	return fantasy.MarshalProviderType(TypeComputerUseMetadata, plain(m))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info.
func (m *ComputerUseMetadata) UnmarshalJSON(data []byte) error {
	type plain ComputerUseMetadata
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*m = ComputerUseMetadata(p)
	return nil
}

// ComputerCallOutputOptions tunes the wire payload fantasy emits for a
// computer_call_output input item. Set it on a ToolResultPart's
// ProviderOptions under the OpenAI provider key. Detail mirrors the
// `output.detail` field documented in the OpenAI computer-use guide:
// values are "auto", "low", "high", or "original". "original" is
// recommended for full-resolution screenshots so the model sees pixel
// coordinates that match the underlying display.
type ComputerCallOutputOptions struct {
	Detail string `json:"detail,omitempty"`
}

var _ fantasy.ProviderOptionsData = (*ComputerCallOutputOptions)(nil)

// Options implements the ProviderOptionsData interface.
func (*ComputerCallOutputOptions) Options() {}

// MarshalJSON implements custom JSON marshaling with type info.
func (o ComputerCallOutputOptions) MarshalJSON() ([]byte, error) {
	type plain ComputerCallOutputOptions
	return fantasy.MarshalProviderType(TypeComputerCallOutputOptions, plain(o))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info.
func (o *ComputerCallOutputOptions) UnmarshalJSON(data []byte) error {
	type plain ComputerCallOutputOptions
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*o = ComputerCallOutputOptions(p)
	return nil
}

// GetComputerCallOutputOptions extracts ComputerCallOutputOptions from
// provider options, returning nil if not present or of a different
// type.
func GetComputerCallOutputOptions(opts fantasy.ProviderOptions) *ComputerCallOutputOptions {
	if v, ok := opts[Name]; ok {
		if o, ok := v.(*ComputerCallOutputOptions); ok {
			return o
		}
	}
	return nil
}

// NewComputerUseTool creates an executable provider-defined computer use
// tool for OpenAI models. The run function receives a ToolCall whose
// Input is a JSON object containing call_id and actions. Parse it with
// ParseComputerUseInput. Return an image response
// (fantasy.NewImageResponse) with a screenshot.
func NewComputerUseTool(
	run func(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error),
) fantasy.ExecutableProviderTool {
	return fantasy.NewExecutableProviderTool(
		fantasy.ProviderDefinedTool{
			ID:   computerUseToolID,
			Name: "computer",
		},
		run,
	)
}

// IsComputerUseTool reports whether tool is an OpenAI computer use tool.
// It returns true for both ExecutableProviderTool and bare
// ProviderDefinedTool instances with the computer use tool ID.
func IsComputerUseTool(tool fantasy.Tool) bool {
	pdt, ok := asProviderDefinedTool(tool)
	if !ok {
		return false
	}
	return pdt.ID == computerUseToolID
}

// asProviderDefinedTool extracts the underlying ProviderDefinedTool from
// either a ProviderDefinedTool or an ExecutableProviderTool.
func asProviderDefinedTool(tool fantasy.Tool) (fantasy.ProviderDefinedTool, bool) {
	switch t := tool.(type) {
	case fantasy.ProviderDefinedTool:
		return t, true
	case fantasy.ExecutableProviderTool:
		return t.Definition(), true
	default:
		return fantasy.ProviderDefinedTool{}, false
	}
}

// GetComputerUseMetadata extracts ComputerUseMetadata from provider
// options, returning nil if not present or of a different type.
func GetComputerUseMetadata(opts fantasy.ProviderOptions) *ComputerUseMetadata {
	if v, ok := opts[Name]; ok {
		if m, ok := v.(*ComputerUseMetadata); ok {
			return m
		}
	}
	return nil
}

// computerCallInput builds a JSON string from a ResponseComputerToolCall
// using per-action RawJSON() for faithful serialization.
func computerCallInput(call responses.ResponseComputerToolCall) (string, error) {
	callIDJSON, err := json.Marshal(call.CallID)
	if err != nil {
		return "", fmt.Errorf("marshal call_id: %w", err)
	}
	obj := map[string]json.RawMessage{
		"call_id": callIDJSON,
	}

	if len(call.Actions) > 0 {
		rawActions := make([]json.RawMessage, len(call.Actions))
		for i, a := range call.Actions {
			rawActions[i] = json.RawMessage(a.RawJSON())
		}
		actionsJSON, err := json.Marshal(rawActions)
		if err != nil {
			return "", fmt.Errorf("marshal actions: %w", err)
		}
		obj["actions"] = actionsJSON
	} else {
		return "", fmt.Errorf("computer_call has no actions")
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal computer call input: %w", err)
	}
	return string(data), nil
}

// ComputerUseInput is the parsed representation of a computer_call
// tool call input. Use ParseComputerUseInput to create one from the
// raw JSON string passed to the Run function.
type ComputerUseInput struct {
	CallID  string                          `json:"call_id"`
	Actions []responses.ComputerActionUnion `json:"actions,omitempty"`
}

// ParseComputerUseInput parses the JSON input string from a computer
// use tool call into typed SDK structures. Callers can type-switch on
// individual actions via action.AsAny().
func ParseComputerUseInput(input string) (*ComputerUseInput, error) {
	if input == "" {
		return nil, fmt.Errorf("empty computer use input")
	}
	var parsed ComputerUseInput
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		return nil, fmt.Errorf("parse computer use input: %w", err)
	}
	return &parsed, nil
}
