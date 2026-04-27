package anthropic

import (
	"cmp"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/auth/bearer"
	"github.com/charmbracelet/anthropic-sdk-go"
)

func bedrockBasicAuthConfig(apiKey string) aws.Config {
	return aws.Config{
		Region:                  cmp.Or(os.Getenv("AWS_REGION"), "us-east-1"),
		BearerAuthTokenProvider: bearer.StaticTokenProvider{Token: bearer.Token{Value: apiKey}},
	}
}

// flattenSystemForBedrock converts system text blocks into the string
// form required by Bedrock's Anthropic Messages schema. Direct
// Anthropic accepts an array of text blocks, so only Bedrock calls use
// this helper.
func flattenSystemForBedrock(blocks []anthropic.TextBlockParam) (string, bool) {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Text == "" {
			continue
		}
		parts = append(parts, block.Text)
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, "\n\n"), true
}

func bedrockPrefixModelWithRegion(modelID string) string {
	region := os.Getenv("AWS_REGION")
	if len(region) < 2 {
		region = "us-east-1"
	}
	prefix := region[:2] + "."
	if strings.HasPrefix(modelID, prefix) {
		return modelID
	}
	return prefix + modelID
}
