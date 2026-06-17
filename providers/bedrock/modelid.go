package bedrock

import (
	"cmp"
	"context"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
)

func normalizeModelID(ctx context.Context, modelID, explicitRegion string, useDefaultChain bool) string {
	if modelID == "" {
		return modelID
	}
	if hasQualifiedAnthropicPrefix(modelID) {
		return modelID
	}
	if !strings.HasPrefix(modelID, "anthropic.") {
		return modelID
	}
	return regionInferenceProfilePrefix(resolveRegion(ctx, explicitRegion, useDefaultChain)) + modelID
}

// resolveRegion returns the region used to choose the cross-region
// inference-profile prefix for a legacy (un-qualified) Bedrock model ID. It must
// mirror the region the inner Anthropic Bedrock provider actually signs the
// request for, otherwise the prefix and the request target diverge and Bedrock
// rejects the call.
//
// The inner provider (see providers/anthropic/anthropic.go LanguageModel)
// resolves the request region differently depending on the auth mode:
//
//   - API key / skip-auth: cmp.Or(bedrockRegion, "us-east-1") via
//     bedrockBasicAuthConfig. The default credential chain is never consulted,
//     so AWS_DEFAULT_REGION, shared-config profiles, and IMDS are ignored.
//   - default credential chain: cmp.Or(bedrockRegion, cfg.Region) where
//     cfg.Region comes from config.LoadDefaultConfig (AWS_REGION ->
//     AWS_DEFAULT_REGION -> shared config profile -> IMDS).
//
// bedrockRegion above is the explicit WithRegion value or AWS_REGION. We only
// reach for the default credential chain when the caller actually relies on it;
// doing so on the API-key path would prefix for a region the request never uses.
func resolveRegion(ctx context.Context, explicitRegion string, useDefaultChain bool) string {
	region := cmp.Or(strings.TrimSpace(explicitRegion), strings.TrimSpace(os.Getenv("AWS_REGION")))
	if region == "" && useDefaultChain {
		if cfg, err := config.LoadDefaultConfig(ctx); err == nil {
			region = strings.TrimSpace(cfg.Region)
		}
	}
	return cmp.Or(region, "us-east-1")
}

func regionInferenceProfilePrefix(region string) string {
	region = strings.ToLower(strings.TrimSpace(region))
	switch {
	case strings.HasPrefix(region, "global"):
		return "global."
	case strings.HasPrefix(region, "us-gov-"), strings.HasPrefix(region, "us-"):
		return "us."
	case strings.HasPrefix(region, "eu-"):
		return "eu."
	case strings.HasPrefix(region, "ap-"):
		return "apac."
	default:
		return "us."
	}
}

func hasQualifiedAnthropicPrefix(modelID string) bool {
	return strings.HasPrefix(modelID, "us.anthropic.") ||
		strings.HasPrefix(modelID, "eu.anthropic.") ||
		strings.HasPrefix(modelID, "apac.anthropic.") ||
		strings.HasPrefix(modelID, "global.anthropic.")
}
