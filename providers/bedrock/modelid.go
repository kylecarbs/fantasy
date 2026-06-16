package bedrock

import (
	"cmp"
	"os"
	"strings"
)

func normalizeModelID(modelID, explicitRegion string) string {
	if modelID == "" {
		return modelID
	}
	if hasQualifiedAnthropicPrefix(modelID) {
		return modelID
	}
	if !strings.HasPrefix(modelID, "anthropic.") {
		return modelID
	}
	return regionInferenceProfilePrefix(resolveRegion(explicitRegion)) + modelID
}

func resolveRegion(explicitRegion string) string {
	return cmp.Or(strings.TrimSpace(explicitRegion), strings.TrimSpace(os.Getenv("AWS_REGION")), "us-east-1")
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
